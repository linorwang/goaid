package keycloak

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client is a Keycloak OIDC client.
type Client struct {
	config         Config
	endpoints      Endpoints
	provider       *oidc.Provider
	oauth2Config   oauth2.Config
	idVerifier     *oidc.IDTokenVerifier
	accessVerifier *oidc.IDTokenVerifier
}

// New creates a Keycloak OIDC client and performs OIDC discovery.
func New(ctx context.Context, cfg Config, opts ...Option) (*Client, error) {
	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	options := clientOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	if options.httpClient != nil {
		ctx = oidc.ClientContext(ctx, options.httpClient)
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	endpoints, err := BuildEndpoints(cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}

	accessVerifierConfig := &oidc.Config{SkipClientIDCheck: true}
	if cfg.AccessTokenAudience != "" {
		accessVerifierConfig = &oidc.Config{ClientID: cfg.AccessTokenAudience}
	}

	return &Client{
		config:         cfg,
		endpoints:      endpoints,
		provider:       provider,
		oauth2Config:   oauth2Config,
		idVerifier:     provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		accessVerifier: provider.Verifier(accessVerifierConfig),
	}, nil
}

// Config returns a copy of the normalized client config.
func (c *Client) Config() Config {
	return c.config
}

// Endpoints returns the common Keycloak OIDC endpoints.
func (c *Client) Endpoints() Endpoints {
	return c.endpoints
}
