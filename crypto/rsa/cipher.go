package rsa

import (
	"crypto/rand"
	cryptorsa "crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// Encrypt encrypts plaintext with RSA-OAEP using SHA-256.
// This is the recommended scheme for new code.
func Encrypt(publicKey *cryptorsa.PublicKey, plaintext []byte) ([]byte, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("rsa: public key is nil")
	}
	cipher, err := cryptorsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, plaintext, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa: encrypt: %w", err)
	}
	return cipher, nil
}

// Decrypt decrypts ciphertext encrypted via Encrypt (RSA-OAEP, SHA-256).
func Decrypt(privateKey *cryptorsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("rsa: private key is nil")
	}
	plain, err := cryptorsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa: decrypt: %w", err)
	}
	return plain, nil
}

// EncryptBase64 is a convenience wrapper around Encrypt that takes a PEM
// public key and a plaintext string and returns base64-encoded ciphertext.
func EncryptBase64(publicKeyPEM string, plaintext string) (string, error) {
	pub, err := StringToPublicKey(publicKeyPEM)
	if err != nil {
		return "", err
	}
	cipher, err := Encrypt(pub, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(cipher), nil
}

// DecryptBase64 is the inverse of EncryptBase64.
func DecryptBase64(privateKeyPEM string, ciphertextB64 string) (string, error) {
	priv, err := StringToPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	cipher, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("rsa: decode base64: %w", err)
	}
	plain, err := Decrypt(priv, cipher)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// EncryptPKCS1v15 encrypts using legacy RSAES-PKCS1-v1_5 padding.
//
// Use this only for interop with existing systems that already use this
// scheme. New code should use Encrypt (OAEP).
func EncryptPKCS1v15(publicKey *cryptorsa.PublicKey, plaintext []byte) ([]byte, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("rsa: public key is nil")
	}
	cipher, err := cryptorsa.EncryptPKCS1v15(rand.Reader, publicKey, plaintext)
	if err != nil {
		return nil, fmt.Errorf("rsa: encrypt PKCS1v15: %w", err)
	}
	return cipher, nil
}

// DecryptPKCS1v15 decrypts ciphertext produced by EncryptPKCS1v15.
//
// Use this only for interop with existing systems. New code should use Decrypt.
func DecryptPKCS1v15(privateKey *cryptorsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("rsa: private key is nil")
	}
	plain, err := cryptorsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("rsa: decrypt PKCS1v15: %w", err)
	}
	return plain, nil
}
