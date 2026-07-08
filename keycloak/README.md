# keycloak

Lightweight Keycloak OpenID Connect client for Go.

This package only wraps reusable Keycloak/OIDC capabilities:

- authorization code login URL generation
- authorization code token exchange
- refresh token exchange
- access token verification
- ID token verification
- OIDC discovery and JWKS-backed verification
- normalized claims parsing
- realm role and client role helpers
- userinfo lookup
- logout URL generation

It does not provide framework middleware, session storage, database integration,
business RBAC, menu permissions, or platform-specific authorization.

## Install

```bash
go get github.com/linorwang/goaid/keycloak
```

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/linorwang/goaid/keycloak"
)

func main() {
	kc, err := keycloak.New(context.Background(), keycloak.Config{
		BaseURL:      "https://keycloak.example.com",
		Realm:        "master",
		ClientID:     "ops-platform",
		ClientSecret: "secret",
		RedirectURL:  "https://ops.example.com/callback",
	})
	if err != nil {
		log.Fatal(err)
	}

	loginURL := kc.AuthCodeURL("state-value")
	_ = loginURL
}
```

## Verify a bearer token

```go
token, err := keycloak.BearerToken(r.Header.Get("Authorization"))
if err != nil {
	// return 401
}

claims, err := kc.VerifyAccessToken(r.Context(), token)
if err != nil {
	// return 401
}

if claims.HasClientRole("ops-platform", "admin") {
	// let the business application decide what admin means
}
```

By default, access token verification checks signature, issuer, and expiry. Set
`Config.AccessTokenAudience` when your Keycloak access tokens include the API
audience you want to enforce.
