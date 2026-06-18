package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// ──────────────────────────────────────────────
// 环境模式: development | production
// ──────────────────────────────────────────────

const (
	AppEnvDev  = "development"
	AppEnvProd = "production"
)

var currentEnv string

func Init() {
	loadEnvFile()
	currentEnv = GetAppEnv()
}

func GetAppEnv() string {
	if currentEnv != "" {
		return currentEnv
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	if env := os.Getenv("GOENV"); env != "" {
		return env
	}
	return AppEnvDev
}

func IsProduction() bool {
	return GetAppEnv() == AppEnvProd
}

func loadEnvFile() {
	env := GetAppEnv()
	var files []string

	switch env {
	case AppEnvProd:
		files = []string{".env.production", ".env"}
	default:
		files = []string{".env.development", ".env"}
	}

	for _, f := range files {
		if err := godotenv.Load(f); err == nil {
			log.Printf("[CONFIG] loaded %s (env=%s)", f, env)
			return
		}
	}
	log.Printf("[CONFIG] no .env file found, using system environment variables")
}

func Port() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}

func ValidateProductionConfig() {
	if !IsProduction() {
		return
	}
	requiredVars := []string{"PG_PASSWORD", "MYSQL_PASSWORD", "JWT_SECRET"}
	for _, v := range requiredVars {
		val := os.Getenv(v)
		if val == "" && v != "MYSQL_PASSWORD" {
			continue
		}
		if val == "" && os.Getenv("DATABASE_DSN") == "" {
			log.Fatalf("[FATAL] production mode requires %s to be set", v)
		}
	}
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

// EnvOrDefault 获取环境变量，为空时返回默认值
func EnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// NormalizePhone 清理手机号格式
func NormalizePhone(phone string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(phone)), "")
}
