package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

var ErrPasswordUnavailable = errors.New("password is unavailable")

// EncryptPassword 使用 AES-256-GCM 加密明文密码（保险箱功能）
func EncryptPassword(plain string) (string, error) {
	block, err := aes.NewCipher(vaultKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	payload := append(nonce, gcm.Seal(nil, nonce, []byte(plain), nil)...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecryptPassword 解密保险箱中的密文密码
func DecryptPassword(secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", ErrPasswordUnavailable
	}
	payload, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(vaultKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", ErrPasswordUnavailable
	}
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// vaultKey 从环境变量派生加密密钥
func vaultKey() []byte {
	secret := strings.TrimSpace(os.Getenv("PASSWORD_VAULT_KEY"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	if secret == "" {
		secret = "change-me-in-production-password-vault"
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
