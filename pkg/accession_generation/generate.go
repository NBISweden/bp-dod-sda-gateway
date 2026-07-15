package accession_generation

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const chars = "abcdefghijklmnopqrstuvxyz23456789"
const length = 6

type ObjectType string

const (
	Dataset = ObjectType("Dataset")
	File    = ObjectType("File")
)

func GenerateAccessionID(objectType ObjectType) string {
	genPart := func() string {
		result := make([]byte, length)
		for i := range length {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			result[i] = chars[n.Int64()]
		}
		return string(result)
	}
	return fmt.Sprintf("aa-%s-%s-%s", objectType, genPart(), genPart())
}
