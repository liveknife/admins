package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Port() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func PrintDevConfig() {
	if IsProduction() {
		return
	}
	keys := []struct{ key, label string }{
		{"APP_ENV", "Mode"},
		{"PORT", "Port"},
		{"DATABASE_TYPE", "DB Type"},
		{"PG_HOST", "PG Host"},
		{"PG_PORT", "PG Port"},
		{"PG_USER", "PG User"},
		{"PG_DB_NAME", "PG Database"},
		{"MYSQL_HOST", "MySQL Host"},
		{"MYSQL_PORT", "MySQL Port"},
		{"MYSQL_USER", "MySQL User"},
		{"MYSQL_DB_NAME", "MySQL Database"},
		{"REDIS_HOST", "Redis Host"},
		{"DB_MAX_OPEN_CONN", "MaxConn"},
		{"DEFAULT_ADMIN_EMAIL", "Admin Email"},
		{"AI_PROVIDER", "AI Provider"},
		{"AI_BASE_URL", "AI Base URL"},
		{"AI_CHAT_MODEL", "AI Chat Model"},
		{"AI_EMBEDDING_MODEL", "AI Embedding Model"},
	}
	maxLen := 0
	for _, k := range keys {
		if l := len(k.label); l > maxLen {
			maxLen = l
		}
	}
	for _, k := range keys {
		v := os.Getenv(k.key)
		if v == "" {
			v = "(not set)"
		}
		fmt.Printf("  %-*s: %s\n", maxLen, k.label, v)
	}
	if dbMaxOpen, _ := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONN")); dbMaxOpen > 0 {
		fmt.Printf("  %-*s: %d\n", maxLen, "Max Open Conn", dbMaxOpen)
	}
}

func EnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func NormalizePhone(phone string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(phone)), "")
}
