package accession

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
)

const chars = "abcdefghijklmnopqrstuvxyz23456789"
const length = 6

type ObjectType string

const (
	Dataset = ObjectType("Dataset")
	File    = ObjectType("File")
)

func GenerateAccession(objectType ObjectType) string {
	genPart := func() string {
		result := make([]byte, length)
		for i := range result {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
			if err != nil {
				slog.Error("error generating accession", "error", err)
				result[i] = chars[0]

				continue
			}
			result[i] = chars[n.Int64()]
		}

		return string(result)
	}

	return fmt.Sprintf("aa-%s-%s-%s", objectType, genPart(), genPart())
}
