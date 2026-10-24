package auth

import (
	"context"
	"log/slog"
	"time"
)

// RunOIDCSessionCleanup removes expired sessions until ctx is cancelled.
// Call in a goroutine only when OIDC authentication is active.
func RunOIDCSessionCleanup(ctx context.Context, store OIDCSessionStore) {
	runOIDCSessionCleanup(ctx, store, time.Hour)
}

func runOIDCSessionCleanup(ctx context.Context, store OIDCSessionStore, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := store.DeleteExpired(cleanupCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.Error("failed to clean expired OIDC sessions", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
