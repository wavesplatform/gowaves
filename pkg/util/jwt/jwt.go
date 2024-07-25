package jwt

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func GenerateJWTToken(jwtSecretPath string) (string, error) {
	jwtHex, err := os.ReadFile(filepath.Clean(jwtSecretPath))
	if err != nil {
		return "", fmt.Errorf("failed to read JWT secret: %w", err)
	}

	var jwtSecret [32]byte
	if _, errHex := hex.Decode(jwtSecret[:], jwtHex); errHex != nil {
		return "", fmt.Errorf("failed to decode JWT secret from hex: %w", errHex)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": &jwt.NumericDate{Time: time.Now()},
	})

	s, err := token.SignedString(jwtSecret[:])
	if err != nil {
		return "", fmt.Errorf("failed to create JWT token: %w", err)
	}
	return s, nil
}
