/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package aes

import (
	"encoding/base64"
	"testing"
)

func Test_EncryptReturnsErrorOnInvalidKeySize(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("short key"))

	_, err := Encrypt(key, "plaintext")
	if err == nil {
		t.Error("Expected error for invalid key size, got nil")
	}
}

func Test_EncryptReturnsErrorOnInvalidBase64Key(t *testing.T) {
	key := "invalid_base64_key"
	_, err := Encrypt(key, "plaintext")
	if err == nil {
		t.Error("Expected error for invalid base64 key, got nil")
	}
}

func Test_DecryptReturnsErrorOnInvalidKeySize(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("short key"))

	_, err := Decrypt(key, "ciphertext")
	if err == nil {
		t.Error("Expected error for invalid key size, got nil")
	}
}

func Test_DecryptReturnsErrorOnInvalidBase64Key(t *testing.T) {
	key := "invalid_base64_key"
	_, err := Decrypt(key, "ciphertext")
	if err == nil {
		t.Error("Expected error for invalid base64 key, got nil")
	}
}

func Test_EncryptReturnsCiphertext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	ciphertext, err := Encrypt(key, "plaintext")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if ciphertext == "" {
		t.Error("Expected non-empty ciphertext, got empty string")
	}
	if ciphertext == "plaintext" {
		t.Error("Expected ciphertext to be different from plaintext")
	}
}

func Test_DecryptReturnsPlaintext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	ciphertext, err := Encrypt(key, "plaintext")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	plaintext, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if plaintext != "plaintext" {
		t.Errorf("Expected plaintext to be 'plaintext', got %s", plaintext)
	}
}
