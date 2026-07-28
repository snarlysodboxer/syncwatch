package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// IdentityConfig controls how the acting user is extracted from a request.
// SyncWatch has no authentication of its own; it trusts whatever identity the
// authenticating proxy in front of it forwards. Tokens are NOT verified here:
// the app should only be reachable through that proxy, which did the
// verifying — and the proxy must strip/overwrite these headers and cookies on
// inbound requests so clients can't spoof them.
type IdentityConfig struct {
	// Header is a request header holding the identity: either plaintext
	// (e.g. oauth2-proxy's X-Forwarded-Email) or a JWT to read Claim from
	// (e.g. Authorization, AWS ALB's x-amzn-oidc-data). Checked first.
	Header string
	// CookiePrefix matches cookies by name prefix whose value is an OIDC ID
	// token JWT to read Claim from — "IdToken" matches the "IdToken-<suffix>"
	// cookies set by Envoy Gateway SecurityPolicies. Empty disables.
	CookiePrefix string
	// Claim is the JWT claim recorded as the identity, e.g. "email".
	Claim string
	// Fallback is used when nothing above matches (local development).
	Fallback string
}

// identityFromRequest extracts the acting user per cfg: configured header
// first, then ID token cookies, then the fallback.
func identityFromRequest(r *http.Request, cfg IdentityConfig) string {
	if cfg.Header != "" {
		if v := strings.TrimPrefix(r.Header.Get(cfg.Header), "Bearer "); v != "" {
			if claim := claimFromJWT(v, cfg.Claim); claim != "" {
				return claim
			}

			return v // not a JWT: treat as a plaintext email/username
		}
	}
	if cfg.CookiePrefix != "" {
		for _, c := range r.Cookies() {
			if strings.HasPrefix(c.Name, cfg.CookiePrefix) {
				if claim := claimFromJWT(c.Value, cfg.Claim); claim != "" {
					return claim
				}
			}
		}
	}

	return cfg.Fallback
}

// claimFromJWT returns a string claim from an (unverified) JWT's payload, or
// "" if the token doesn't parse or the claim is absent.
func claimFromJWT(token, claim string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	s, _ := claims[claim].(string)

	return s
}
