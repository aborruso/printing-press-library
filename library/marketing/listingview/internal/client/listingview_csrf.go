// Hand-authored, not generated. ListingView cookie + CSRF + shop-context auth.
//
// ListingView's backend (a NestJS app reached through the same-origin
// /api/proxy rewrite) authenticates with the httpOnly session cookie set when
// you log in to app.listingview.io, and additionally requires an
// X-CSRF-Token header on requests whose value is the readable `csrf_token`
// cookie (CSRF double-submit). Shop-scoped endpoints also expect a `shopid`
// header naming the active Etsy shop.
//
// `auth login --chrome` imports the whole cookie jar into Config (cookie auth
// stores it as a "name=value; name2=value2" string carried via AuthHeader()
// with a "Bearer " scheme prefix we strip here). We send it as the Cookie
// header and derive X-CSRF-Token from it. shopid is optional: set
// LISTINGVIEW_SHOP_ID or a `[headers] shopid = "..."` config entry when a
// shop-scoped call needs it (global research works without it).
package client

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/marketing/listingview/internal/config"
)

// applyListingViewAuth sets the Cookie header from the imported cookie string
// (stripping any Bearer scheme), derives and sets the X-CSRF-Token header from
// the csrf_token cookie, and sets an optional shopid header. It never
// overwrites a header an explicit per-request override already set.
func applyListingViewAuth(req *http.Request, authHeader string, cfg *config.Config) {
	cookieStr := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(authHeader), "Bearer "))
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
		if tok := csrfTokenFromCookie(cookieStr); tok != "" && req.Header.Get("X-CSRF-Token") == "" {
			req.Header.Set("X-CSRF-Token", tok)
		}
	}
	if req.Header.Get("shopid") == "" {
		if shop := listingViewShopID(cfg); shop != "" {
			req.Header.Set("shopid", shop)
		}
	}
}

// csrfTokenFromCookie extracts and URL-decodes the csrf_token cookie value from
// a "name=value; name2=value2" cookie string.
func csrfTokenFromCookie(cookieStr string) string {
	if cookieStr == "" {
		return ""
	}
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		name, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "csrf_token") {
			if decoded, err := url.QueryUnescape(value); err == nil {
				return decoded
			}
			return value
		}
	}
	return ""
}

// listingViewShopID resolves the active Etsy shop id for the shopid header from
// LISTINGVIEW_SHOP_ID or a configured [headers] shopid value. Returns "" when
// unset, in which case no shopid header is sent.
func listingViewShopID(cfg *config.Config) string {
	if v := strings.TrimSpace(os.Getenv("LISTINGVIEW_SHOP_ID")); v != "" {
		return v
	}
	if cfg != nil && cfg.Headers != nil {
		if v := strings.TrimSpace(cfg.Headers["shopid"]); v != "" {
			return v
		}
	}
	return ""
}
