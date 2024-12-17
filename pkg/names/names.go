package names

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// NewRandomK8 returns a random k8s compatible name with the given prefix.
func NewRandomK8(prefix string) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", prefix, uuid.String()[:8]), nil
}

func HashWithLength(input string, length int) string {
	hash := sha1.Sum([]byte(input))
	return hex.EncodeToString(hash[:])[:length]
}
