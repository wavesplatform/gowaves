package jwt

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/golang-jwt/jwt/v4"
)

const (
	jwtHexSize = 64
	jwtSize    = 32
)

func GenerateJWTToken(jwtSecretPath string) (string, error) {
	jwtFile, err := os.Open(filepath.Clean(jwtSecretPath))
	if err != nil {
		return "", fmt.Errorf("failed to open JWT secret file: %w", err)
	}

	defer func() {
		if clErr := jwtFile.Close(); clErr != nil {
			zap.S().Errorf("Failed to close jwt secret: %v", clErr)
		}
	}()

	var jwtHex [jwtHexSize]byte
	_, err = jwtFile.Read(jwtHex[:])
	if err != nil {
		return "", fmt.Errorf("failed to read jwt secret: %w", err)
	}

	var jwtSecret [jwtSize]byte
	if _, errHex := hex.Decode(jwtSecret[:], jwtHex[:]); errHex != nil {
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
