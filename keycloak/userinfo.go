package keycloak

import (
	"context"
	"strings"

	"golang.org/x/oauth2"
)

// UserInfo contains normalized OIDC userinfo data.
type UserInfo struct {
	Subject  string         `json:"sub,omitempty"`
	Username string         `json:"preferred_username,omitempty"`
	Email    string         `json:"email,omitempty"`
	Name     string         `json:"name,omitempty"`
	Groups   []string       `json:"groups,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}

// UserInfo requests the Keycloak userinfo endpoint.
func (c *Client) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, ErrMissingToken
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	oidcUserInfo, err := c.provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return nil, err
	}

	var info UserInfo
	if err := oidcUserInfo.Claims(&info); err != nil {
		return nil, err
	}
	info.Subject = oidcUserInfo.Subject

	var raw map[string]any
	if err := oidcUserInfo.Claims(&raw); err == nil {
		info.Raw = raw
	}

	return &info, nil
}
