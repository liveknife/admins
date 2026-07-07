package database

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Type        DBType
	DSN         string
	PGHost      string
	PGPort      string
	PGUser      string
	PGPassword  string
	PGDBName    string
	PGSSLMode   string
	MyHost      string
	MyPort      string
	MyUser      string
	MyPassword  string
	MyDBName    string
	MaxOpenConn int
	MaxIdleConn int
}

func DefaultConfig() Config {
	dbType, _ := ParseDBType(os.Getenv("DATABASE_TYPE"))
	dsn := os.Getenv("DATABASE_DSN")
	cfg := Config{Type: dbType, DSN: dsn, MaxOpenConn: 25, MaxIdleConn: 5}

	cfg.PGHost = envOr("PG_HOST", "localhost")
	cfg.PGPort = envOr("PG_PORT", "5432")
	cfg.PGUser = envOr("PG_USER", "postgres")
	cfg.PGPassword = os.Getenv("PG_PASSWORD")
	cfg.PGDBName = envOr("PG_DB_NAME", "admins")
	cfg.PGSSLMode = envOr("PG_SSL_MODE", "disable")

	cfg.MyHost = envOr("MYSQL_HOST", "localhost")
	cfg.MyPort = envOr("MYSQL_PORT", "3306")
	cfg.MyUser = envOr("MYSQL_USER", "root")
	cfg.MyPassword = os.Getenv("MYSQL_PASSWORD")
	cfg.MyDBName = envOr("MYSQL_DB_NAME", "admins")

	if dsn == "" {
		switch dbType {
		case DBTypePostgres:
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
				cfg.PGUser, cfg.PGPassword, cfg.PGHost, cfg.PGPort, cfg.PGDBName, cfg.PGSSLMode)
		case DBTypeMySQL:
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
				cfg.MyUser, cfg.MyPassword, cfg.MyHost, cfg.MyPort, cfg.MyDBName)
		default:
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
				cfg.PGUser, cfg.PGPassword, cfg.PGHost, cfg.PGPort, cfg.PGDBName, cfg.PGSSLMode)
		}
	}
	cfg.DSN = dsn

	if v := os.Getenv("DB_MAX_OPEN_CONN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxOpenConn = n
		}
	}
	if v := os.Getenv("DB_MAX_IDLE_CONN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxIdleConn = n
		}
	}
	return cfg
}

func Connect() (*DB, error) {
	cfg := DefaultConfig()
	CurrentDialect = &Dialect{Type: cfg.Type, DriverName: driverName(cfg.Type)}
	db, err := sql.Open(CurrentDialect.DriverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open %s failed: %w", cfg.Type, err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConn)
	db.SetMaxIdleConns(cfg.MaxIdleConn)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping %s failed: %w", cfg.Type, err)
	}
	fmt.Printf("[%s] connected successfully\n", strings.ToUpper(cfg.Type.String()))
	if err := migrate(db, CurrentDialect); err != nil {
		return nil, fmt.Errorf("migrate failed: %w", err)
	}
	return &DB{db}, nil
}

func driverName(t DBType) string {
	switch t {
	case DBTypePostgres:
		return "pgx"
	case DBTypeMySQL:
		return "mysql"
	default:
		return "pgx"
	}
}
