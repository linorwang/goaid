package keycloak

import (
	"fmt"
	"net/url"
	"strings"
)

// Endpoints contains the common Keycloak OIDC endpoints for a realm.
type Endpoints struct {
	Issuer   string
	Auth     string
	Token    string
	UserInfo string
	Logout   string
	JWKS     string
}

// BuildIssuerURL returns the Keycloak realm issuer URL.
func BuildIssuerURL(baseURL, realm string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	realm = strings.Trim(strings.TrimSpace(realm), "/")
	if baseURL == "" {
		return "", ErrMissingBaseURL
	}
	if realm == "" {
		return "", ErrMissingRealm
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return "", fmt.Errorf("keycloak: invalid base url: %w", err)
	}
	return baseURL + "/realms/" + url.PathEscape(realm), nil
}

// BuildEndpoints returns the standard Keycloak OIDC endpoints for an issuer.
func BuildEndpoints(issuerURL string) (Endpoints, error) {
	issuerURL = strings.TrimRight(strings.TrimSpace(issuerURL), "/")
	if issuerURL == "" {
		return Endpoints{}, ErrMissingBaseURL
	}
	if _, err := url.ParseRequestURI(issuerURL); err != nil {
		return Endpoints{}, fmt.Errorf("keycloak: invalid issuer url: %w", err)
	}

	openid := issuerURL + "/protocol/openid-connect"
	return Endpoints{
		Issuer:   issuerURL,
		Auth:     openid + "/auth",
		Token:    openid + "/token",
		UserInfo: openid + "/userinfo",
		Logout:   openid + "/logout",
		JWKS:     openid + "/certs",
	}, nil
}
