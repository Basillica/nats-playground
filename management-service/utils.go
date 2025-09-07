package main

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
)

func GenerateSASToken(orgID, deviceID string) (string, error) {
	claims := jwt.NewUserClaims(deviceID)
	claims.Permissions.Pub.Allow.Add(fmt.Sprintf("data.%s.%s.>", orgID, deviceID))

	// Set a short expiration time for security
	claims.Expires = time.Now().Add(12 * time.Hour).Unix()
	claims.NotBefore = time.Now().Unix()

	token, err := claims.Encode(signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode JWT: %w", err)
	}
	return token, nil
}
