package utils

import (
	"fmt"
	"os"
	"strconv"
)

// GetEnv try to get an environment variable from process envs and if not found use a fallback value
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvBool(key string) bool {
	val := GetEnv(key, "")
	ret, err := strconv.ParseBool(val)
	if err != nil {
		panic(fmt.Sprintf("some error"))
	}
	return ret
}
