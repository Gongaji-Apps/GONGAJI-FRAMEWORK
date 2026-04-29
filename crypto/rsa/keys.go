// Package rsa provides RSA key management and encryption helpers.
//
// Because this package is named "rsa" it collides with crypto/rsa from the
// standard library. Consumers should alias on import, e.g.
//
//	import gongajirsa "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/crypto/rsa"
package rsa

import (
	cryptorsa "crypto/rsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// DefaultKeySize is the bit size used by GenerateKeyPair.
const DefaultKeySize = 2048

const (
	pemTypePrivateKey = "RSA PRIVATE KEY"
	pemTypePublicKey  = "PUBLIC KEY"
)

// GenerateKeyPair generates a new RSA key pair of DefaultKeySize bits.
func GenerateKeyPair() (*cryptorsa.PrivateKey, *cryptorsa.PublicKey, error) {
	return GenerateKeyPairWithSize(DefaultKeySize)
}

// GenerateKeyPairWithSize generates a new RSA key pair of the given bit size.
// Sizes below 2048 are rejected.
func GenerateKeyPairWithSize(bits int) (*cryptorsa.PrivateKey, *cryptorsa.PublicKey, error) {
	if bits < 2048 {
		return nil, nil, fmt.Errorf("rsa: key size %d below 2048-bit minimum", bits)
	}
	priv, err := cryptorsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("rsa: generate key: %w", err)
	}
	return priv, &priv.PublicKey, nil
}

// PrivateKeyToString serializes a private key as a PEM-encoded PKCS#1 string.
func PrivateKeyToString(key *cryptorsa.PrivateKey) string {
	if key == nil {
		return ""
	}
	block := &pem.Block{
		Type:  pemTypePrivateKey,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

// PublicKeyToString serializes a public key as a PEM-encoded PKIX string.
func PublicKeyToString(key *cryptorsa.PublicKey) (string, error) {
	if key == nil {
		return "", errors.New("rsa: public key is nil")
	}
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("rsa: marshal public key: %w", err)
	}
	block := &pem.Block{
		Type:  pemTypePublicKey,
		Bytes: der,
	}
	return string(pem.EncodeToMemory(block)), nil
}

// StringToPrivateKey parses a PEM-encoded private key. Accepts both PKCS#1
// ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE KEY") blocks.
func StringToPrivateKey(pemStr string) (*cryptorsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("rsa: invalid PEM block")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("rsa: parse private key: %w", err)
	}
	rsaKey, ok := parsed.(*cryptorsa.PrivateKey)
	if !ok {
		return nil, errors.New("rsa: PEM does not contain an RSA private key")
	}
	return rsaKey, nil
}

// StringToPublicKey parses a PEM-encoded public key. Accepts both PKIX
// ("PUBLIC KEY") and legacy PKCS#1 ("RSA PUBLIC KEY") blocks.
func StringToPublicKey(pemStr string) (*cryptorsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("rsa: invalid PEM block")
	}

	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*cryptorsa.PublicKey)
		if !ok {
			return nil, errors.New("rsa: PEM does not contain an RSA public key")
		}
		return rsaKey, nil
	}

	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.New("rsa: parse public key: unsupported format")
}
