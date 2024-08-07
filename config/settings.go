package config

import (
	"os"
)

var (
	TODO_DBFILE = getEnv("TODO_DBFILE", "scheduler.db")
	TODO_PORT   = getEnv("TODO_PORT", "7540")
	TODO_PASS   = getEnv("TODO_PASSWORD", "")
)

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
