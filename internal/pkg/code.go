package pkg

import (
	cryptoRand "crypto/rand"
	"math/big"
	"strings"
)

func RandDigits(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		x, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + x.Int64()))
	}
	return b.String(), nil
}
