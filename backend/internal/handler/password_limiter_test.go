package handler

import (
	"net/http/httptest"
	"testing"
	"time"
)

func Test_PasswordAttemptLimiter_BlocksAndExpiresFailures(t *testing.T) {
	limiter := newPasswordAttemptLimiter()
	request := httptest.NewRequest("POST", "/oauth/password", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	now := time.Now().UTC()

	for range passwordAttemptLimit {
		if !limiter.allow(request, "user@example.com", now) {
			t.Fatal("expected attempt to remain allowed before the limit")
		}
		limiter.failure(request, "user@example.com", now)
	}
	if limiter.allow(request, "user@example.com", now) {
		t.Fatal("expected attempts to be blocked at the limit")
	}
	if !limiter.allow(request, "user@example.com", now.Add(passwordAttemptWindow)) {
		t.Fatal("expected attempts to be allowed after the window")
	}
}

func Test_PasswordAttemptLimiter_IsolatesAccountsAndClearsSuccess(t *testing.T) {
	limiter := newPasswordAttemptLimiter()
	request := httptest.NewRequest("POST", "/oauth/password", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	now := time.Now().UTC()

	limiter.failure(request, "one@example.com", now)
	if !limiter.allow(request, "two@example.com", now) {
		t.Fatal("expected a different account to remain allowed")
	}
	limiter.success(request, "one@example.com")
	if !limiter.allow(request, "one@example.com", now) {
		t.Fatal("expected successful authentication to clear failures")
	}
}
