package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var frontendFS embed.FS

// spaHandler returns an http.Handler that serves the embedded SPA.
// It serves static files from the embedded filesystem and falls back
// to 200.html for client-side routing (Nuxt generates 200.html for SPA fallback).
func spaHandler() http.Handler {
	// Strip the "dist" prefix from the embedded filesystem
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		urlPath := path.Clean(r.URL.Path)

		// Try to find the file
		filePath := strings.TrimPrefix(urlPath, "/")
		if filePath == "" {
			filePath = "index.html"
		}

		// Check if the exact file exists
		if file, err := distFS.Open(filePath); err == nil {
			file.Close()
			http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
			return
		}

		// Check if it's a directory with an index.html
		indexPath := path.Join(filePath, "index.html")
		if file, err := distFS.Open(indexPath); err == nil {
			file.Close()
			// Serve the directory's index.html
			r.URL.Path = "/" + indexPath
			http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
			return
		}

		// Fallback to 200.html for SPA routing
		content, err := fs.ReadFile(distFS, "200.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	})
}
