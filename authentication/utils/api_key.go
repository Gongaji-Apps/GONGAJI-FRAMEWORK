package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func GenerateApiKey(prefix string) (string, error) {
    b := make([]byte, 32)
	
    if _, err := rand.Read(b); err != nil {
        return "", err
    }

    return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

func HashApiKey(key string) string {
    h := sha256.Sum256([]byte(key))

    return hex.EncodeToString(h[:])
}
