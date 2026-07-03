package tool

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestEncodePassWord(t *testing.T) {
	encoded := EncodePassWord("123456")
	if !VerifyPassWord("123456", encoded) {
		t.Fatalf("VerifyPassWord failed for encoded password")
	}
	if VerifyPassWord("wrong-password", encoded) {
		t.Fatalf("VerifyPassWord accepted an invalid password")
	}
}

func TestMultiPasswordVerify(t *testing.T) {
	password := "secret123"
	salt := "legacy-salt"
	md5Hash := md5.Sum([]byte(password))
	sha256Hash := sha256.Sum256([]byte(password))
	md5SaltHash := md5.Sum([]byte(password + salt))
	sha256SaltHash := sha256.Sum256([]byte(password + salt))
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}
	defaultHash := EncodePassWord(password)

	tests := []struct {
		name string
		algo string
		salt string
		hash string
	}{
		{name: "md5", algo: "md5", hash: hex.EncodeToString(md5Hash[:])},
		{name: "sha256", algo: "sha256", hash: hex.EncodeToString(sha256Hash[:])},
		{name: "md5salt", algo: "md5salt", salt: salt, hash: hex.EncodeToString(md5SaltHash[:])},
		{name: "sha256salt", algo: "sha256salt", salt: salt, hash: hex.EncodeToString(sha256SaltHash[:])},
		{name: "bcrypt", algo: "bcrypt", hash: string(bcryptHash)},
		{name: "default", algo: "default", hash: defaultHash},
		{name: "case and space tolerant", algo: " SHA256SALT ", salt: salt, hash: hex.EncodeToString(sha256SaltHash[:])},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !MultiPasswordVerify(tt.algo, tt.salt, password, tt.hash) {
				t.Fatalf("MultiPasswordVerify(%q) rejected a valid password", tt.algo)
			}
			if MultiPasswordVerify(tt.algo, tt.salt, "wrong-password", tt.hash) {
				t.Fatalf("MultiPasswordVerify(%q) accepted an invalid password", tt.algo)
			}
		})
	}
}
