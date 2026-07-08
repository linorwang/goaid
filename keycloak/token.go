package keycloak

import (
	"context"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// TokenResult is the token response returned by Keycloak.
type TokenResult struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	TokenType    string
	Expiry       time.Time

	OAuth2Token *oauth2.Token
}

// AuthCodeURL returns a Keycloak login URL for the authorization code flow.
func (c *Client) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c.oauth2Config.AuthCodeURL(state, opts...)
}

// Exchange exchanges an authorization code for tokens.
func (c *Client) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*TokenResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrMissingCode
	}

	token, err := c.oauth2Config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}
	return newTokenResult(token), nil
}

// Refresh exchanges a refresh token for a new token set.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (*TokenResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}

	source := c.oauth2Config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := source.Token()
	if err != nil {
		return nil, err
	}
	return newTokenResult(token), nil
}

func newTokenResult(token *oauth2.Token) *TokenResult {
	if token == nil {
		return &TokenResult{}
	}

	idToken, _ := token.Extra("id_token").(string)
	return &TokenResult{
		AccessToken:  token.AccessToken,
		IDToken:      idToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
		OAuth2Token:  token,
	}
}
