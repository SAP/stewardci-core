package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
)

// RandomAlphaNumString generates a random string value consisting of [0-9a-z] with a length
// as configured.
func RandomAlphaNumString(length int64) (string, error) {
	if length <= 0 {
		return "", nil
	}

	const base = 36 // number of symbols to be used [0-9a-z]
	maxRandom := new(big.Int).Exp(big.NewInt(base), big.NewInt(int64(length)), nil)
	randomInt, err := rand.Int(rand.Reader, maxRandom)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0"+strconv.Itoa(int(length))+"s", randomInt.Text(base)), nil
}
