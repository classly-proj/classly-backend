package db

import (
	"crypto/sha256"
)

func HashPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return string(hash.Sum(nil))
}
