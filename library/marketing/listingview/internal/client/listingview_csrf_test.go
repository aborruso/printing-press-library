package client

import (
	"net/http"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/config"
)

func TestCSRFTokenFromCookie(t *testing.T) {
	cases := []struct {
		name   string
		cookie string
		want   string
	}{
		{"present", "lv_landing=x; csrf_token=abc-123; __stripe_mid=y", "abc-123"},
		{"url-encoded", "csrf_token=a%2Bb%3Dc", "a+b=c"},
		{"case-insensitive name", "CSRF_TOKEN=xyz", "xyz"},
		{"absent", "session=foo; other=bar", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := csrfTokenFromCookie(tc.cookie); got != tc.want {
				t.Fatalf("csrfTokenFromCookie(%q) = %q, want %q", tc.cookie, got, tc.want)
			}
		})
	}
}

func TestApplyListingViewAuth(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://app.listingview.io/api/proxy/x", nil)
	cfg := &config.Config{Headers: map[string]string{"shopid": "65974856"}}
	applyListingViewAuth(req, "Bearer session=abc; csrf_token=tok-99", cfg)

	if got := req.Header.Get("Cookie"); got != "session=abc; csrf_token=tok-99" {
		t.Fatalf("Cookie header = %q, want stripped raw cookie string", got)
	}
	if got := req.Header.Get("X-CSRF-Token"); got != "tok-99" {
		t.Fatalf("X-CSRF-Token = %q, want tok-99", got)
	}
	if got := req.Header.Get("shopid"); got != "65974856" {
		t.Fatalf("shopid = %q, want 65974856", got)
	}
}

func TestApplyListingViewAuthNoCSRF(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://app.listingview.io/api/auth/me", nil)
	applyListingViewAuth(req, "session=only", nil)
	if got := req.Header.Get("Cookie"); got != "session=only" {
		t.Fatalf("Cookie = %q, want session=only", got)
	}
	if got := req.Header.Get("X-CSRF-Token"); got != "" {
		t.Fatalf("X-CSRF-Token = %q, want empty when no csrf_token cookie", got)
	}
}
