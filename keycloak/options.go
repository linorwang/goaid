package keycloak

import "net/http"

type clientOptions struct {
	httpClient *http.Client
}

// Option customizes Client creation.
type Option func(*clientOptions)

// WithHTTPClient configures the HTTP client used for OIDC discovery and calls.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(opts *clientOptions) {
		if httpClient != nil {
			opts.httpClient = httpClient
		}
	}
}
