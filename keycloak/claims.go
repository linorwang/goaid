package keycloak

import (
	"encoding/json"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Claims contains normalized Keycloak user and role claims.
type Claims struct {
	Subject     string              `json:"sub,omitempty"`
	Username    string              `json:"preferred_username,omitempty"`
	Email       string              `json:"email,omitempty"`
	Name        string              `json:"name,omitempty"`
	Groups      []string            `json:"groups,omitempty"`
	RealmRoles  []string            `json:"realm_roles,omitempty"`
	ClientRoles map[string][]string `json:"client_roles,omitempty"`
	Issuer      string              `json:"iss,omitempty"`
	Audience    []string            `json:"aud,omitempty"`
	Expiry      time.Time           `json:"exp,omitempty"`
	IssuedAt    time.Time           `json:"iat,omitempty"`
	Raw         map[string]any      `json:"raw,omitempty"`
}

type rawClaims struct {
	Subject        string                   `json:"sub"`
	Username       string                   `json:"preferred_username"`
	Email          string                   `json:"email"`
	Name           string                   `json:"name"`
	Groups         []string                 `json:"groups"`
	RealmAccess    rawRoleAccess            `json:"realm_access"`
	ResourceAccess map[string]rawRoleAccess `json:"resource_access"`
	Issuer         string                   `json:"iss"`
	Audience       audienceClaim            `json:"aud"`
	Expiry         unixTime                 `json:"exp"`
	IssuedAt       unixTime                 `json:"iat"`
}

type rawRoleAccess struct {
	Roles []string `json:"roles"`
}

type audienceClaim []string

func (a *audienceClaim) UnmarshalJSON(data []byte) error {
	var many []string
	if err := json.Unmarshal(data, &many); err == nil {
		*a = many
		return nil
	}

	var one string
	if err := json.Unmarshal(data, &one); err != nil {
		return err
	}
	*a = []string{one}
	return nil
}

type unixTime time.Time

func (u *unixTime) UnmarshalJSON(data []byte) error {
	var seconds int64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return err
	}
	*u = unixTime(time.Unix(seconds, 0))
	return nil
}

func (u unixTime) Time() time.Time {
	return time.Time(u)
}

// ClaimsFromIDToken converts a verified OIDC token into normalized claims.
func ClaimsFromIDToken(idToken *oidc.IDToken) (*Claims, error) {
	var raw rawClaims
	if err := idToken.Claims(&raw); err != nil {
		return nil, err
	}

	var rawMap map[string]any
	if err := idToken.Claims(&rawMap); err != nil {
		return nil, err
	}

	clientRoles := make(map[string][]string, len(raw.ResourceAccess))
	for clientID, access := range raw.ResourceAccess {
		clientRoles[clientID] = uniqueStrings(access.Roles)
	}

	return &Claims{
		Subject:     raw.Subject,
		Username:    raw.Username,
		Email:       raw.Email,
		Name:        raw.Name,
		Groups:      uniqueStrings(raw.Groups),
		RealmRoles:  uniqueStrings(raw.RealmAccess.Roles),
		ClientRoles: clientRoles,
		Issuer:      raw.Issuer,
		Audience:    uniqueStrings([]string(raw.Audience)),
		Expiry:      raw.Expiry.Time(),
		IssuedAt:    raw.IssuedAt.Time(),
		Raw:         rawMap,
	}, nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
