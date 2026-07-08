package keycloak

import (
	"strings"
)

// Config contains the Keycloak OIDC client settings.
type Config struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string

	// IssuerURL overrides BaseURL + Realm when set.
	IssuerURL string

	// AccessTokenAudience enables audience validation for access tokens.
	// Leave empty to validate signature, issuer and expiry only.
	AccessTokenAudience string
}

// DefaultScopes returns the minimal OIDC scopes used by most applications.
func DefaultScopes() []string {
	return []string{"openid", "profile", "email"}
}

// Validate checks whether the config has enough information to build a client.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ClientID) == "" {
		return ErrMissingClientID
	}
	if strings.TrimSpace(c.IssuerURL) != "" {
		return nil
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return ErrMissingBaseURL
	}
	if strings.TrimSpace(c.Realm) == "" {
		return ErrMissingRealm
	}
	return nil
}

func normalizeConfig(c Config) (Config, error) {
	if err := c.Validate(); err != nil {
		return Config{}, err
	}

	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	c.Realm = strings.Trim(strings.TrimSpace(c.Realm), "/")
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.ClientSecret = strings.TrimSpace(c.ClientSecret)
	c.RedirectURL = strings.TrimSpace(c.RedirectURL)
	c.IssuerURL = strings.TrimRight(strings.TrimSpace(c.IssuerURL), "/")
	c.AccessTokenAudience = strings.TrimSpace(c.AccessTokenAudience)

	if len(c.Scopes) == 0 {
		c.Scopes = DefaultScopes()
	}
	if c.IssuerURL == "" {
		issuer, err := BuildIssuerURL(c.BaseURL, c.Realm)
		if err != nil {
			return Config{}, err
		}
		c.IssuerURL = issuer
	}

	return c, nil
}
