package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

// deriveKey turns any-length master key material into a fixed 32-byte AES key.
func deriveKey(master string) []byte {
	sum := sha256.Sum256([]byte(master))
	return sum[:]
}

// Encrypt seals plaintext with AES-256-GCM using a key derived from master.
// Output layout: nonce || ciphertext. Used for SSH private keys at rest.
func Encrypt(master string, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(master))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt.
func Decrypt(master string, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(master))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}
