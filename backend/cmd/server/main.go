package main

import (
	"context"
	_ "embed"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/mendelui/attic/internal/auth"
	"github.com/mendelui/attic/internal/config"
	"github.com/mendelui/attic/internal/database"
	"github.com/mendelui/attic/internal/handler"
	"github.com/mendelui/attic/internal/plugin"
	"github.com/mendelui/attic/internal/plugin/googlebooks"
	"github.com/mendelui/attic/internal/repository"
	"github.com/mendelui/attic/internal/storage"
)

//go:embed openapi.yaml
var openapiSpec []byte

// Version is set by ldflags during build
var Version = "dev"

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Database connection
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to database")

	// S3 client
	s3Client, err := storage.NewS3Client(ctx, storage.S3Config{
		Endpoint:  cfg.S3Endpoint,
		Region:    cfg.S3Region,
		Bucket:    cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
	})
	if err != nil {
		slog.Warn("failed to connect to S3, attachments will be disabled", "error", err)
		s3Client = nil
	} else {
		slog.Info("connected to S3", "bucket", cfg.S3Bucket)
	}

	// OAuth handler for login flow
	oauthHandler, err := auth.NewOAuthHandler(ctx, auth.OAuthConfig{
		IssuerURL:     cfg.OIDCIssuer,
		ClientID:      cfg.OIDCClientID,
		BaseURL:       cfg.BaseURL,
		SessionSecret: cfg.SessionSecret,
		Disabled:      cfg.AuthDisabled,
	})
	if err != nil {
		slog.Error("failed to initialize OAuth handler", "error", err)
		os.Exit(1)
	}

	// Auth middleware
	authMiddleware, err := auth.NewMiddleware(ctx, auth.Config{
		IssuerURL: cfg.OIDCIssuer,
		ClientID:  cfg.OIDCClientID,
		Disabled:  cfg.AuthDisabled,
	})
	if err != nil {
		slog.Error("failed to initialize auth", "error", err)
		os.Exit(1)
	}

	// Link OAuth handler to middleware for cookie-based auth
	authMiddleware.SetOAuthHandler(oauthHandler)

	if cfg.AuthDisabled {
		slog.Warn("authentication is disabled")
	} else {
		slog.Info("OIDC authentication enabled", "issuer", cfg.OIDCIssuer)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.Pool)
	repos := &handler.Repositories{
		Organizations: repository.NewOrganizationRepository(db.Pool),
		Users:         userRepo,
		Categories:    repository.NewCategoryRepository(db.Pool),
		Locations:     repository.NewLocationRepository(db.Pool),
		Conditions:    repository.NewConditionRepository(db.Pool),
		Assets:        repository.NewAssetRepository(db.Pool),
		Warranties:    repository.NewWarrantyRepository(db.Pool),
		Attachments:   repository.NewAttachmentRepository(db.Pool),
		Attributes:    repository.NewAttributeRepository(db.Pool),
	}

	// User provisioner
	userProvisioner := auth.NewUserProvisioner(userRepo, defaultOrgID)

	// Initialize plugin registry
	pluginRegistry := plugin.NewRegistry()
	if err := pluginRegistry.Register(googlebooks.New()); err != nil {
		slog.Error("failed to register Google Books plugin", "error", err)
	}
	slog.Info("registered plugins", "count", len(pluginRegistry.List()))

	// Initialize handlers
	h := handler.New(db, repos, s3Client)
	pluginHandler := handler.NewPluginHandler(pluginRegistry, repos, s3Client)

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   strings.Split(cfg.CORSOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check (no auth required)
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)

	// OpenAPI documentation
	r.Get("/api/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(openapiSpec)
	})
	r.Get("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(swaggerUIHTML))
	})

	// Auth routes (no auth required)
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", oauthHandler.Login)
		r.Get("/callback", oauthHandler.Callback)
		r.Get("/logout", oauthHandler.Logout)
		r.Get("/session", oauthHandler.GetSession)
	})

	// API routes (auth required)
	r.Route("/api", func(r chi.Router) {
		// Apply auth middleware to all /api routes
		r.Use(authMiddleware.Authenticate)
		r.Use(userProvisioner.Provision)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
		})

		// Current user info
		r.Get("/me", h.GetCurrentUser)

		// Categories
		r.Route("/categories", func(r chi.Router) {
			r.Get("/", h.ListCategories)
			r.Post("/", h.CreateCategory)
			r.Get("/{id}", h.GetCategory)
			r.Put("/{id}", h.UpdateCategory)
			r.Delete("/{id}", h.DeleteCategory)
		})

		// Attributes
		r.Route("/attributes", func(r chi.Router) {
			r.Get("/", h.ListAttributes)
			r.Post("/", h.CreateAttribute)
			r.Get("/{id}", h.GetAttribute)
			r.Put("/{id}", h.UpdateAttribute)
			r.Delete("/{id}", h.DeleteAttribute)
		})

		// Locations
		r.Route("/locations", func(r chi.Router) {
			r.Get("/", h.ListLocations)
			r.Post("/", h.CreateLocation)
			r.Get("/{id}", h.GetLocation)
			r.Put("/{id}", h.UpdateLocation)
			r.Delete("/{id}", h.DeleteLocation)
		})

		// Conditions
		r.Route("/conditions", func(r chi.Router) {
			r.Get("/", h.ListConditions)
			r.Post("/", h.CreateCondition)
			r.Get("/{id}", h.GetCondition)
			r.Put("/{id}", h.UpdateCondition)
			r.Delete("/{id}", h.DeleteCondition)
		})

		// Assets
		r.Route("/assets", func(r chi.Router) {
			r.Get("/", h.ListAssets)
			r.Post("/", h.CreateAsset)
			r.Get("/{id}", h.GetAsset)
			r.Put("/{id}", h.UpdateAsset)
			r.Delete("/{id}", h.DeleteAsset)

			// Warranty (nested under asset)
			r.Get("/{id}/warranty", h.GetWarranty)
			r.Post("/{id}/warranty", h.CreateWarranty)
			r.Put("/{id}/warranty", h.UpdateWarranty)
			r.Delete("/{id}/warranty", h.DeleteWarranty)

			// Attachments (nested under asset)
			r.Get("/{id}/attachments", h.ListAttachments)
			r.Post("/{id}/attachments", h.UploadAttachment)
		})

		// Attachment operations (by attachment ID)
		r.Route("/attachments", func(r chi.Router) {
			r.Get("/{attachmentId}", h.GetAttachment)
			r.Delete("/{attachmentId}", h.DeleteAttachment)
		})

		// Warranties overview
		r.Get("/warranties/expiring", h.ListExpiringWarranties)

		// Import Plugins
		r.Route("/plugins", func(r chi.Router) {
			r.Get("/", pluginHandler.ListPlugins)
			r.Get("/{pluginId}", pluginHandler.GetPlugin)
			r.Get("/{pluginId}/search", pluginHandler.Search)
			r.Post("/{pluginId}/import", pluginHandler.Import)
		})
	})

	// Serve embedded frontend for all non-API routes
	r.Handle("/*", spaHandler())

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("starting server", "port", cfg.Port, "version", Version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Attic API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "/api/openapi.yaml",
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
