package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

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
