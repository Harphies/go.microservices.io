package utils

import (
	"github.com/oklog/ulid"
	"math/rand"
	"time"
)

func GenerateID() (string, error) {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)
	return id.String(), err
}

/*
Example usage
	id, err := debug-container.GenerateID()
	if err != nil {
		stream.logger.Error("failed generating uuid")
		return "", err
	}
	streamId := strings.ToLower(id)
*/
