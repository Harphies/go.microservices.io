package utils

import "os"

// GetEnv try to get an environment variable from process envs and if not found use a fallback value
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
