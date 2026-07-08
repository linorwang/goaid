package keycloak

// HasRealmRole reports whether the token has a realm role.
func (c *Claims) HasRealmRole(role string) bool {
	return containsString(c.RealmRoles, role)
}

// HasAnyRealmRole reports whether the token has at least one realm role.
func (c *Claims) HasAnyRealmRole(roles ...string) bool {
	for _, role := range roles {
		if c.HasRealmRole(role) {
			return true
		}
	}
	return false
}

// HasAllRealmRoles reports whether the token has all realm roles.
func (c *Claims) HasAllRealmRoles(roles ...string) bool {
	for _, role := range roles {
		if !c.HasRealmRole(role) {
			return false
		}
	}
	return true
}

// HasClientRole reports whether the token has a role for a client.
func (c *Claims) HasClientRole(clientID, role string) bool {
	if c.ClientRoles == nil {
		return false
	}
	return containsString(c.ClientRoles[clientID], role)
}

// HasAnyClientRole reports whether the token has at least one role for a client.
func (c *Claims) HasAnyClientRole(clientID string, roles ...string) bool {
	for _, role := range roles {
		if c.HasClientRole(clientID, role) {
			return true
		}
	}
	return false
}

// HasAllClientRoles reports whether the token has all roles for a client.
func (c *Claims) HasAllClientRoles(clientID string, roles ...string) bool {
	for _, role := range roles {
		if !c.HasClientRole(clientID, role) {
			return false
		}
	}
	return true
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
