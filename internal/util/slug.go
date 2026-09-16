package util

import (
	"crypto/rand"
	"math/big"
)

const slugChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateSlug(length int) string {
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(slugChars))))
		b[i] = slugChars[n.Int64()]
	}
	return string(b)
}
