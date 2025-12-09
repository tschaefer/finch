/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package aes

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_EncryptReturnsError_InvalidKeySize(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("short key"))

	_, err := Encrypt(key, "plaintext")
	wanted := "AES key must be 32 bytes"
	assert.EqualError(t, err, wanted, "encrypt short key")
}

func Test_EncryptReturnsError_InvalidKeyEncoding(t *testing.T) {
	key := "invalid_base64_key"
	_, err := Encrypt(key, "plaintext")
	wanted := "illegal base64 data at input byte 7"
	assert.EqualError(t, err, wanted, "encrypt invalid base64 key")
}

func Test_DecryptReturnsError_InvalidKeySize(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("short key"))

	_, err := Decrypt(key, "ciphertext")
	wanted := "AES key must be 32 bytes"
	assert.EqualError(t, err, wanted, "decrypt short key")
}

func Test_DecryptReturnsError_InvalidKeyEncoding(t *testing.T) {
	key := "invalid_base64_key"
	_, err := Decrypt(key, "ciphertext")
	wanted := "illegal base64 data at input byte 7"
	assert.EqualError(t, err, wanted, "decrypt invalid base64 key")
}

func Test_EncryptReturnsCiphertext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	ciphertext, err := Encrypt(key, "plaintext")
	assert.NoError(t, err, "encrypt ciphertext")
	assert.NotEmpty(t, ciphertext, "ciphertext")
	assert.NotEqual(t, "plaintext", ciphertext, "plaintext")
}

func Test_DecryptReturnsPlaintext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	ciphertext, err := Encrypt(key, "plaintext")
	assert.NoError(t, err, "encrypt ciphertext")

	plaintext, err := Decrypt(key, ciphertext)
	assert.NoError(t, err, "decrypt plaintext")
	assert.Equal(t, "plaintext", plaintext, "plaintext")
}
