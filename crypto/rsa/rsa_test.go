package rsa

import (
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair error: %v", err)
	}
	if priv == nil || pub == nil {
		t.Fatal("nil keys returned")
	}
	if priv.N.BitLen() < DefaultKeySize {
		t.Errorf("priv.N.BitLen() = %d, want >= %d", priv.N.BitLen(), DefaultKeySize)
	}
	if &priv.PublicKey != pub {
		t.Error("returned public key should be a pointer to priv.PublicKey")
	}
}

func TestGenerateKeyPair_RejectsSmallSize(t *testing.T) {
	if _, _, err := GenerateKeyPairWithSize(1024); err == nil {
		t.Fatal("expected error for 1024-bit key, got nil")
	}
}

func TestPEMRoundTrip_Private(t *testing.T) {
	priv, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	encoded := PrivateKeyToString(priv)
	if !strings.Contains(encoded, "RSA PRIVATE KEY") {
		t.Errorf("encoded PEM missing header: %q", encoded)
	}

	decoded, err := StringToPrivateKey(encoded)
	if err != nil {
		t.Fatalf("StringToPrivateKey: %v", err)
	}
	if decoded.N.Cmp(priv.N) != 0 {
		t.Error("decoded modulus does not match original")
	}
}

func TestPEMRoundTrip_Public(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := PublicKeyToString(pub)
	if err != nil {
		t.Fatalf("PublicKeyToString: %v", err)
	}
	if !strings.Contains(encoded, "PUBLIC KEY") {
		t.Errorf("encoded PEM missing header: %q", encoded)
	}

	decoded, err := StringToPublicKey(encoded)
	if err != nil {
		t.Fatalf("StringToPublicKey: %v", err)
	}
	if decoded.N.Cmp(pub.N) != 0 {
		t.Error("decoded modulus does not match original")
	}
}

func TestStringToPrivateKey_InvalidPEM(t *testing.T) {
	if _, err := StringToPrivateKey("not pem"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStringToPublicKey_InvalidPEM(t *testing.T) {
	if _, err := StringToPublicKey("not pem"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEncryptDecrypt_OAEP(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("rahasia: 1234567890")

	cipher, err := Encrypt(pub, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(cipher) == 0 {
		t.Fatal("cipher is empty")
	}

	decoded, err := Decrypt(priv, cipher)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decoded) != string(plaintext) {
		t.Errorf("decoded = %q, want %q", decoded, plaintext)
	}
}

func TestEncryptDecrypt_OAEP_DifferentKeyFails(t *testing.T) {
	_, pub1, _ := GenerateKeyPair()
	priv2, _, _ := GenerateKeyPair()

	cipher, err := Encrypt(pub1, []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(priv2, cipher); err == nil {
		t.Fatal("expected decryption failure with wrong key")
	}
}

func TestEncryptDecrypt_NilKey(t *testing.T) {
	if _, err := Encrypt(nil, []byte("x")); err == nil {
		t.Error("Encrypt with nil key should error")
	}
	if _, err := Decrypt(nil, []byte("x")); err == nil {
		t.Error("Decrypt with nil key should error")
	}
}

func TestBase64Helpers(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	pubPEM, err := PublicKeyToString(pub)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := PrivateKeyToString(priv)

	plaintext := "halo dunia"

	cipherB64, err := EncryptBase64(pubPEM, plaintext)
	if err != nil {
		t.Fatalf("EncryptBase64: %v", err)
	}
	if cipherB64 == "" {
		t.Fatal("empty cipher base64")
	}

	got, err := DecryptBase64(privPEM, cipherB64)
	if err != nil {
		t.Fatalf("DecryptBase64: %v", err)
	}
	if got != plaintext {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestDecryptBase64_BadInput(t *testing.T) {
	priv, _, _ := GenerateKeyPair()
	privPEM := PrivateKeyToString(priv)

	if _, err := DecryptBase64(privPEM, "$$$not-base64$$$"); err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestPKCS1v15_RoundTrip(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("legacy payload")

	cipher, err := EncryptPKCS1v15(pub, plaintext)
	if err != nil {
		t.Fatalf("EncryptPKCS1v15: %v", err)
	}
	got, err := DecryptPKCS1v15(priv, cipher)
	if err != nil {
		t.Fatalf("DecryptPKCS1v15: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Errorf("got %q, want %q", got, plaintext)
	}
}

func TestOAEPDecrypt_RejectsPKCS1v15Cipher(t *testing.T) {
	priv, pub, _ := GenerateKeyPair()
	cipher, err := EncryptPKCS1v15(pub, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(priv, cipher); err == nil {
		t.Error("expected OAEP Decrypt to reject PKCS1v15 ciphertext")
	}
}
