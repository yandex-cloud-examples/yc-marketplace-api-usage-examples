package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// RndString returns a random string of a given length using the characters in
// the given string. It splits the string on runes to support UTF-8
// characters.
func RndString(length int, chars string) (string, error) {
	result := make([]rune, length)
	runes := []rune(chars)
	x := int64(len(runes))
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(x))
		if err != nil {
			return "", errors.Join(err, errors.New("error creating random number"))
		}
		result[i] = runes[num.Int64()]
	}
	return string(result), nil
}

func fakeId() string {
	// generate random alphanumeric string 17 characters long

	const alphabet = "1234567890abcdefghijklmnopqrstuvwxyz"

	rnd, err := RndString(17, alphabet)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("mkt%s", rnd)
}
