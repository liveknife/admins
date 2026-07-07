package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func ValidateProductionConfig() {
	if err := ProductionConfigError(); err != nil {
		log.Fatal(err)
	}
}

func ProductionConfigError() error {
	if !IsProduction() {
		return nil
	}

	if err := requireNonEmpty("JWT_SECRET"); err != nil {
		return err
	}
	if strings.TrimSpace(os.Getenv("JWT_SECRET")) == "please-change-this-secret-in-production" {
		return fmt.Errorf("[FATAL] production mode requires JWT_SECRET to be changed from the default value")
	}

	if strings.TrimSpace(os.Getenv("DATABASE_DSN")) != "" {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_TYPE"))) {
	case "mysql", "maria", "mariadb":
		return requireNonEmpty("MYSQL_PASSWORD")
	default:
		return requireNonEmpty("PG_PASSWORD")
	}
}

func requireNonEmpty(key string) error {
	if strings.TrimSpace(os.Getenv(key)) == "" {
		return fmt.Errorf("[FATAL] production mode requires %s to be set", key)
	}
	return nil
}
