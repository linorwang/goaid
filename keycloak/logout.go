package keycloak

import (
	"net/url"
	"strings"
)

// LogoutURL returns the Keycloak end-session URL.
func (c *Client) LogoutURL(idTokenHint, postLogoutRedirectURL string) (string, error) {
	postLogoutRedirectURL = strings.TrimSpace(postLogoutRedirectURL)
	if postLogoutRedirectURL == "" {
		return "", ErrMissingPostLogoutURL
	}

	values := url.Values{}
	values.Set("client_id", c.config.ClientID)
	values.Set("post_logout_redirect_uri", postLogoutRedirectURL)
	if strings.TrimSpace(idTokenHint) != "" {
		values.Set("id_token_hint", strings.TrimSpace(idTokenHint))
	}

	return c.endpoints.Logout + "?" + values.Encode(), nil
}
