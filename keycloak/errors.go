package keycloak

import "errors"

var (
	ErrMissingBaseURL        = errors.New("keycloak: base url is required")
	ErrMissingRealm          = errors.New("keycloak: realm is required")
	ErrMissingClientID       = errors.New("keycloak: client id is required")
	ErrMissingToken          = errors.New("keycloak: token is required")
	ErrMissingCode           = errors.New("keycloak: authorization code is required")
	ErrMissingRefreshToken   = errors.New("keycloak: refresh token is required")
	ErrInvalidAuthorization  = errors.New("keycloak: invalid authorization header")
	ErrMissingBearerToken    = errors.New("keycloak: missing bearer token")
	ErrMissingPostLogoutURL  = errors.New("keycloak: post logout redirect url is required")
	ErrUnsupportedClaimShape = errors.New("keycloak: unsupported claim shape")
)
