package cryptoblock

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

const fixture = "enc:v1:AAECAwQFBgcICQoLDA0ODw:EBESExQVFhcYGRobHB0eHyAhIiMkJSYn:kjoOIIaoKTbcrT789T4USc1ZP4jD-siNCzoQtC-I0x8dWFRt2OBpKg"

func TestImmutableRustFixture(t *testing.T) {
	salt, nonce := make([]byte, 16), make([]byte, 24)
	for i := range salt {
		salt[i] = byte(i)
	}
	for i := range nonce {
		nonce[i] = byte(i + 16)
	}
	got, err := encryptV1([]byte("# Legacy secret\nold note"), "archive-pass", salt, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if got != fixture {
		t.Fatalf("wire format differs from Rust: %s", got)
	}
	plain, err := Decrypt(fixture, "archive-pass")
	if err != nil || plain != "# Legacy secret\nold note" {
		t.Fatalf("fixture: %q %v", plain, err)
	}
}
func TestRoundTripAuthenticationAndVersions(t *testing.T) {
	p, err := Encrypt("# Секрет\nкириллица ☀", "пароль")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(p, "пароль")
	if err != nil || plain != "# Секрет\nкириллица ☀" {
		t.Fatalf("round trip: %q %v", plain, err)
	}
	if _, err = Decrypt(p, "wrong"); !errors.Is(err, ErrAuthentication) {
		t.Fatal(err)
	}
	parts := strings.Split(p, ":")
	data, _ := base64.RawURLEncoding.DecodeString(parts[4])
	data[0] ^= 1
	parts[4] = base64.RawURLEncoding.EncodeToString(data)
	if _, err = Decrypt(strings.Join(parts, ":"), "пароль"); !errors.Is(err, ErrAuthentication) {
		t.Fatal(err)
	}
	if _, err = Decrypt(strings.Replace(p, "enc:v1:", "enc:v2:", 1), "пароль"); !errors.Is(err, ErrFormat) {
		t.Fatal(err)
	}
	if _, err = Decrypt("enc:v1:bad:bad:bad", "x"); !errors.Is(err, ErrDamaged) {
		t.Fatal(err)
	}
	if _, err = Encrypt("", "p"); !errors.Is(err, ErrEmpty) {
		t.Fatal(err)
	}
	if _, err = Decrypt(p, ""); !errors.Is(err, ErrPassword) {
		t.Fatal(err)
	}
}
