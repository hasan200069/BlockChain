package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func HashEverything(input interface{}) string {
	inputString := fmt.Sprintf("%v", input)
	hash1 := sha256.New()
	hash1.Write([]byte(inputString))
	hash := hash1.Sum(nil)
	return hex.EncodeToString(hash)
}
