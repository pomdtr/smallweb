package utils

import (
	"crypto/rand"
	"math/big"
)

const base62Charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// Base62 encoding function to generate random Base62 strings
func GenerateBase62String(length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(base62Charset)))

	for i := range result {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = base62Charset[num.Int64()]
	}

	return string(result), nil
}
