package randomid

import (
	"crypto/rand"
	"fmt"
)

const alphaNumChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// AlphaNum returns a cryptographically random alphanumeric string of length n.
func AlphaNum(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("invalid length: %d", n)
	}

	b := make([]byte, n)
	for i := range b {
		idx, err := randomAlphaNumIndex(len(alphaNumChars))
		if err != nil {
			return "", err
		}
		b[i] = alphaNumChars[idx]
	}
	return string(b), nil
}

func randomAlphaNumIndex(max int) (int, error) {
	if max <= 0 || max > 256 {
		return 0, fmt.Errorf("invalid max: %d", max)
	}

	limit := byte(256 - (256 % max))
	for {
		var one [1]byte
		if _, err := rand.Read(one[:]); err != nil {
			return 0, err
		}
		if one[0] < limit {
			return int(one[0]) % max, nil
		}
	}
}
