package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

// ValidateBasicAuth validates HTTP Basic Authentication credentials
func ValidateBasicAuth(authHeader, expectedUsername, expectedPassword string) bool {
	if authHeader == "" {
		return false
	}

	// Remove "Basic " prefix
	if !strings.HasPrefix(authHeader, "Basic ") {
		return false
	}

	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return false
	}

	username := parts[0]
	password := parts[1]

	// Use constant-time comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1

	return usernameMatch && passwordMatch
}

// CreateBasicAuthHeader creates a Basic Auth header value
func CreateBasicAuthHeader(username, password string) string {
	credentials := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encoded
}
