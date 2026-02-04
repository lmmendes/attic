package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
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
	"github.com/lmmendes/attic/internal/auth"
	"github.com/lmmendes/attic/internal/config"
	"github.com/lmmendes/attic/internal/database"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/handler"
	"github.com/lmmendes/attic/internal/plugin"
	"github.com/lmmendes/attic/internal/plugin/bgg"
	"github.com/lmmendes/attic/internal/plugin/googlebooks"
	"github.com/lmmendes/attic/internal/plugin/tmdb"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/storage"
)

//go:embed openapi.yaml
var openapiSpec []byte

// Version is set by ldflags during build
var Version = "dev"

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func main() {
	// CLI flags for password reset
	resetPassword := flag.Bool("reset-password", false, "Reset a user's password")
	email := flag.String("email", "", "User email for password reset")
	newPassword := flag.String("new-password", "", "New password for the user")
	flag.Parse()

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

	// Handle CLI password reset
	if *resetPassword {
		handlePasswordReset(ctx, db, cfg, *email, *newPassword)
		return
	}

	slog.Info("connected to database")

	// Initialize file storage (S3 or local)
	var fileStorage storage.FileStorage
	if cfg.UseS3Storage() {
		s3Client, err := storage.NewS3Client(ctx, storage.S3Config{
			Endpoint:  cfg.S3Endpoint,
			Region:    cfg.S3Region,
			Bucket:    cfg.S3Bucket,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
		})
		if err != nil {
			slog.Warn("failed to connect to S3, attachments will be disabled", "error", err)
		} else {
			slog.Info("using S3 storage", "bucket", cfg.S3Bucket)
			fileStorage = s3Client
		}
	} else {
		localStorage, err := storage.NewLocalStorage(storage.LocalConfig{
			BasePath: cfg.LocalStoragePath,
			BaseURL:  cfg.BaseURL + "/files",
			PUID:     cfg.PUID,
			PGID:     cfg.PGID,
		})
		if err != nil {
			slog.Warn("failed to initialize local storage, attachments will be disabled", "error", err)
		} else {
			if cfg.HasFileOwnership() {
				slog.Info("using local file storage", "path", cfg.LocalStoragePath, "puid", *cfg.PUID, "pgid", *cfg.PGID)
			} else {
				slog.Info("using local file storage", "path", cfg.LocalStoragePath)
			}
			fileStorage = localStorage
		}
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

	// Bootstrap admin user if needed
	if err := bootstrapAdmin(ctx, userRepo, cfg); err != nil {
		slog.Error("failed to bootstrap admin", "error", err)
		os.Exit(1)
	}

	// Session manager for local auth
	sessionManager := auth.NewSessionManager(cfg.SessionSecret, cfg.SessionDurationHours)

	// Auth middleware
	authMiddleware, err := auth.NewMiddleware(ctx, auth.Config{
		IssuerURL:   cfg.OIDCIssuer,
		ClientID:    cfg.OIDCClientID,
		Disabled:    cfg.AuthDisabled,
		OIDCEnabled: cfg.OIDCEnabled,
	})
	if err != nil {
		slog.Error("failed to initialize auth", "error", err)
		os.Exit(1)
	}

	// Set session manager for local auth
	authMiddleware.SetSessionManager(sessionManager)

	// OAuth handler for OIDC login flow (only if OIDC enabled)
	var oauthHandler *auth.OAuthHandler
	if cfg.OIDCEnabled {
		oauthHandler, err = auth.NewOAuthHandler(ctx, auth.OAuthConfig{
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
		authMiddleware.SetOAuthHandler(oauthHandler)
	}

	// User provisioner (for OIDC mode)
	userProvisioner := auth.NewUserProvisioner(userRepo, defaultOrgID)

	if cfg.AuthDisabled {
		slog.Warn("authentication is disabled")
	} else if cfg.OIDCEnabled {
		slog.Info("OIDC authentication enabled", "issuer", cfg.OIDCIssuer)
	} else {
		slog.Info("local (email/password) authentication enabled")
	}

	// Initialize plugin registry
	pluginRegistry := plugin.NewRegistry()
	if err := pluginRegistry.Register(googlebooks.New()); err != nil {
		slog.Error("failed to register Google Books plugin", "error", err)
	}
	if err := pluginRegistry.Register(tmdb.NewMoviesPlugin()); err != nil {
		slog.Error("failed to register TMDB Movies plugin", "error", err)
	}
	if err := pluginRegistry.Register(tmdb.NewSeriesPlugin()); err != nil {
		slog.Error("failed to register TMDB Series plugin", "error", err)
	}
	if err := pluginRegistry.Register(bgg.New()); err != nil {
		slog.Error("failed to register BGG plugin", "error", err)
	}
	slog.Info("registered plugins", "count", len(pluginRegistry.List()))

	// Initialize handlers
	h := handler.New(db, repos, fileStorage)
	pluginHandler := handler.NewPluginHandler(pluginRegistry, repos, fileStorage)
	authHandler := handler.NewAuthHandler(userRepo, sessionManager, cfg.PasswordMinLength, cfg.OIDCEnabled)
	userMgmtHandler := handler.NewUserManagementHandler(userRepo, sessionManager, cfg.PasswordMinLength, defaultOrgID)

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

	// Serve local files (only when using local storage)
	if localStorage, ok := fileStorage.(*storage.LocalStorage); ok {
		fileServer := http.StripPrefix("/files/", http.FileServer(http.Dir(localStorage.BasePath())))
		r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
			fileServer.ServeHTTP(w, r)
		})
	}

	// Auth routes (no auth required)
	r.Route("/auth", func(r chi.Router) {
		// Local auth endpoints
		r.Post("/login", authHandler.Login)
		r.Post("/logout", authHandler.Logout)
		r.Get("/session", authHandler.GetSession)
		r.Get("/mode", authHandler.GetAuthMode)

		// OIDC endpoints (only when OIDC enabled)
		if cfg.OIDCEnabled && oauthHandler != nil {
			r.Get("/oidc/login", oauthHandler.Login)
			r.Get("/oidc/callback", oauthHandler.Callback)
			r.Get("/oidc/logout", oauthHandler.Logout)
			// Keep old routes for backwards compatibility
			r.Get("/login", oauthHandler.Login)
			r.Get("/callback", oauthHandler.Callback)
		}
	})

	// API routes (auth required)
	r.Route("/api", func(r chi.Router) {
		// Apply auth middleware to all /api routes
		r.Use(authMiddleware.Authenticate)

		// Only use user provisioner for OIDC mode
		if cfg.OIDCEnabled {
			r.Use(userProvisioner.Provision)
		}

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok","version":"` + Version + `"}`))
		})

		// Auth endpoints (requires authentication)
		r.Route("/auth", func(r chi.Router) {
			r.Put("/password", authHandler.ChangePassword)
		})

		// Current user info
		r.Get("/me", h.GetCurrentUser)

		// User management (admin only)
		r.Route("/users", func(r chi.Router) {
			r.Use(auth.RequireAdmin(sessionManager))
			r.Get("/", userMgmtHandler.ListUsers)
			r.Post("/", userMgmtHandler.CreateUser)
			r.Get("/{id}", userMgmtHandler.GetUser)
			r.Put("/{id}", userMgmtHandler.UpdateUser)
			r.Delete("/{id}", userMgmtHandler.DeleteUser)
			r.Post("/{id}/reset-password", userMgmtHandler.ResetPassword)
		})

		// Categories
		r.Route("/categories", func(r chi.Router) {
			r.Get("/", h.ListCategories)
			r.Post("/", h.CreateCategory)
			r.Get("/asset-counts", h.GetCategoryAssetCounts)
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
			r.Get("/stats", h.GetAssetStats)
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

			// Main image
			r.Put("/{id}/main-image/{attachmentId}", h.SetMainAttachment)
			r.Delete("/{id}/main-image", h.ClearMainAttachment)
		})

		// Attachment operations (by attachment ID)
		r.Route("/attachments", func(r chi.Router) {
			r.Get("/{attachmentId}", h.GetAttachment)
			r.Delete("/{attachmentId}", h.DeleteAttachment)
		})

		// Warranties overview
		r.Get("/warranties", h.ListWarranties)
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

// bootstrapAdmin creates the initial admin user if no users exist
func bootstrapAdmin(ctx context.Context, userRepo *repository.UserRepository, cfg *config.Config) error {
	count, err := userRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("counting users: %w", err)
	}

	if count > 0 {
		return nil // Users exist, skip bootstrap
	}

	// Hash the password
	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("hashing admin password: %w", err)
	}

	// Create admin user
	admin := &domain.User{
		OrganizationID: defaultOrgID,
		Email:          cfg.AdminEmail,
		PasswordHash:   &hash,
		Role:           domain.UserRoleAdmin,
	}
	displayName := "Administrator"
	admin.DisplayName = &displayName

	if err := userRepo.Create(ctx, admin); err != nil {
		return fmt.Errorf("creating admin user: %w", err)
	}

	slog.Info("created bootstrap admin user", "email", cfg.AdminEmail)

	// Warn if using default credentials
	if cfg.AdminEmail == "admin" && cfg.AdminPassword == "admin" {
		slog.Warn("using default admin credentials - please change them immediately!")
	}

	return nil
}

// handlePasswordReset handles the CLI password reset command
func handlePasswordReset(ctx context.Context, db *database.DB, cfg *config.Config, email, newPassword string) {
	if email == "" {
		fmt.Fprintln(os.Stderr, "error: --email is required")
		os.Exit(1)
	}
	if newPassword == "" {
		fmt.Fprintln(os.Stderr, "error: --new-password is required")
		os.Exit(1)
	}

	if err := auth.ValidatePassword(newPassword, cfg.PasswordMinLength); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(db.Pool)

	user, err := userRepo.GetByEmail(ctx, email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to find user: %s\n", err)
		os.Exit(1)
	}
	if user == nil {
		fmt.Fprintf(os.Stderr, "error: user with email '%s' not found\n", email)
		os.Exit(1)
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to hash password: %s\n", err)
		os.Exit(1)
	}

	if err := userRepo.UpdatePassword(ctx, user.ID, hash); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to update password: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Password updated successfully for user '%s'\n", email)
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
