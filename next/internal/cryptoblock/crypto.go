// Package cryptoblock implements the immutable Plainmark enc:v1 wire format.
package cryptoblock

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const aadV1 = "plainmark:encrypted-block:v1"

var (
	ErrPassword       = errors.New("Password cannot be empty.")
	ErrEmpty          = errors.New("Select some text to encrypt.")
	ErrFormat         = errors.New("Unsupported encrypted block format.")
	ErrDamaged        = errors.New("This encrypted block is damaged.")
	ErrAuthentication = errors.New("Incorrect password or the encrypted block is damaged.")
)

// Password strings, as all Go strings, may have copies managed by the GC.
// Clear owned key buffers, but do not claim Rust zeroize-equivalent guarantees.
func keyV1(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 65536, 4, 32)
}

func Encrypt(plaintext, password string) (string, error) {
	if plaintext == "" {
		return "", ErrEmpty
	}
	if password == "" {
		return "", ErrPassword
	}
	salt, nonce := make([]byte, 16), make([]byte, 24)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return encryptV1([]byte(plaintext), password, salt, nonce)
}
func encryptV1(plaintext []byte, password string, salt, nonce []byte) (string, error) {
	if len(salt) != 16 || len(nonce) != 24 {
		return "", ErrDamaged
	}
	key := keyV1(password, salt)
	defer clear(key)
	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}
	encrypted := cipher.Seal(nil, nonce, plaintext, []byte(aadV1))
	b64 := base64.RawURLEncoding
	return "enc:v1:" + b64.EncodeToString(salt) + ":" + b64.EncodeToString(nonce) + ":" + b64.EncodeToString(encrypted), nil
}
func Decrypt(payload, password string) (string, error) {
	if password == "" {
		return "", ErrPassword
	}
	parts := strings.Split(strings.TrimSpace(payload), ":")
	if len(parts) != 5 || parts[0] != "enc" || parts[1] != "v1" {
		return "", ErrFormat
	}
	salt, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || len(salt) != 16 {
		return "", ErrDamaged
	}
	nonce, err := base64.RawURLEncoding.Strict().DecodeString(parts[3])
	if err != nil || len(nonce) != 24 {
		return "", ErrDamaged
	}
	encrypted, err := base64.RawURLEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return "", ErrDamaged
	}
	key := keyV1(password, salt)
	defer clear(key)
	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}
	plain, err := cipher.Open(nil, nonce, encrypted, []byte(aadV1))
	if err != nil {
		return "", ErrAuthentication
	}
	defer clear(plain)
	if !utf8.Valid(plain) {
		return "", ErrDamaged
	}
	return string(plain), nil
}
