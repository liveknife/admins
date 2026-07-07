package config

import (
	"strings"
	"testing"
)

func TestProductionConfigRequiresJWTSecret(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP_ENV", AppEnvProd)
	t.Setenv("PG_PASSWORD", "secret")

	err := ProductionConfigError()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET error, got %v", err)
	}
}

func TestProductionConfigAllowsDSNWithoutDatabasePassword(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP_ENV", AppEnvProd)
	t.Setenv("JWT_SECRET", "a-strong-production-secret")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost:5432/admins?sslmode=disable")

	if err := ProductionConfigError(); err != nil {
		t.Fatalf("expected DATABASE_DSN to satisfy database config, got %v", err)
	}
}

func TestProductionConfigRequiresMySQLPassword(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP_ENV", AppEnvProd)
	t.Setenv("DATABASE_TYPE", "mysql")
	t.Setenv("JWT_SECRET", "a-strong-production-secret")

	err := ProductionConfigError()
	if err == nil || !strings.Contains(err.Error(), "MYSQL_PASSWORD") {
		t.Fatalf("expected MYSQL_PASSWORD error, got %v", err)
	}
}

func resetConfigEnv(t *testing.T) {
	t.Helper()
	currentEnv = ""
	t.Cleanup(func() { currentEnv = "" })
}
