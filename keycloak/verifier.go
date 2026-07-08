package keycloak

import (
	"context"
	"strings"
)

// VerifyAccessToken verifies a Keycloak access token and returns normalized claims.
func (c *Client) VerifyAccessToken(ctx context.Context, token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrMissingToken
	}

	idToken, err := c.accessVerifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	return ClaimsFromIDToken(idToken)
}

// VerifyIDToken verifies a Keycloak ID token and returns normalized claims.
func (c *Client) VerifyIDToken(ctx context.Context, token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrMissingToken
	}

	idToken, err := c.idVerifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	return ClaimsFromIDToken(idToken)
}

// BearerToken extracts a bearer token from an Authorization header value.
func BearerToken(authorization string) (string, error) {
	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return "", ErrMissingBearerToken
	}

	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidAuthorization
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", ErrMissingBearerToken
	}
	return parts[1], nil
}
