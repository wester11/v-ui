package jwtauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// SecretBox — AES-256-GCM шифрование.
type SecretBox struct {
	gcm cipher.AEAD
}

func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, errors.New("secretbox: key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{gcm: gcm}, nil
}

func (b *SecretBox) Seal(plain []byte) ([]byte, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return b.gcm.Seal(nonce, nonce, plain, nil), nil
}

func (b *SecretBox) Open(ct []byte) ([]byte, error) {
	ns := b.gcm.NonceSize()
	if len(ct) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, payload := ct[:ns], ct[ns:]
	return b.gcm.Open(nil, nonce, payload, nil)
}
