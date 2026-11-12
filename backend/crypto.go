package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

// encrypt, AES-256-GCM ile şifreler; çıktı base64(nonce || şifreli metin).
func encrypt(key []byte, plain string) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func decrypt(key []byte, encoded string) (string, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", errors.New("geçersiz şifreli veri")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", errors.New("şifre çözülemedi (anahtar yanlış veya veri bozuk)")
	}
	return string(plain), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
