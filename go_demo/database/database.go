package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-demo/utils"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

// ──────────────────────────────────────────────
// 数据库类型
// ──────────────────────────────────────────────

type DBType int

const (
	DBTypePostgres DBType = iota
	DBTypeMySQL
)

func (t DBType) String() string {
	switch t {
	case DBTypePostgres:
		return "postgres"
	case DBTypeMySQL:
		return "mysql"
	default:
		return "unknown"
	}
}

func ParseDBType(s string) (DBType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "postgres", "postgresql", "pg":
		return DBTypePostgres, nil
	case "mysql", "maria", "mariadb":
		return DBTypeMySQL, nil
	default:
		return 0, fmt.Errorf("unsupported database type: %q (supported: postgres, mysql)", s)
	}
}

// ──────────────────────────────────────────────
// 方言（Dialect）
// ──────────────────────────────────────────────

var placeholderRe = regexp.MustCompile(`\$(\d+)`)

type Dialect struct {
	Type       DBType
	DriverName string
}

func (d *Dialect) Placeholder(n int) string {
	if d.Type == DBTypePostgres {
		return "$" + strconv.Itoa(n)
	}
	return "?"
}

func (d *Dialect) RewriteSQL(sql string) string {
	if d.Type == DBTypePostgres {
		return sql
	}
	return placeholderRe.ReplaceAllStringFunc(sql, func(match string) string { return "?" })
}

func (d *Dialect) Now() string {
	if d.Type == DBTypePostgres {
		return "NOW()"
	}
	return "CURRENT_TIMESTAMP"
}

func (d *Dialect) AutoIncrement() string {
	switch d.Type {
	case DBTypePostgres:
		return "BIGSERIAL PRIMARY KEY"
	default:
		return "BIGINT AUTO_INCREMENT PRIMARY KEY"
	}
}

func (d *Dialect) Timestamp() string {
	if d.Type == DBTypePostgres {
		return "TIMESTAMPTZ"
	}
	return "DATETIME"
}

func (d *Dialect) UpsertClause(onConflict ...string) string {
	if len(onConflict) == 0 || onConflict[0] == "" {
		onConflict = []string{"(id)"}
	}
	switch d.Type {
	case DBTypePostgres:
		return fmt.Sprintf("ON CONFLICT %s DO NOTHING", onConflict[0])
	case DBTypeMySQL:
		return "ON DUPLICATE KEY UPDATE id=id"
	default:
		return "ON CONFLICT DO NOTHING"
	}
}

func (d *Dialect) StringAgg(distinct bool, expr, delimiter string) string {
	dist := ""
	if distinct {
		dist = "DISTINCT "
	}
	switch d.Type {
	case DBTypePostgres:
		return fmt.Sprintf("string_agg(%s%s, '%s')", dist, expr, delimiter)
	default:
		return fmt.Sprintf("GROUP_CONCAT(%s%s SEPARATOR '%s')", dist, expr, delimiter)
	}
}

func (d *Dialect) SupportsReturning() bool { return d.Type == DBTypePostgres }
func (d *Dialect) ColumnExistsSQL() string {
	return `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)`
}

var CurrentDialect *Dialect

// ──────────────────────────────────────────────
// 配置
// ──────────────────────────────────────────────

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

// ──────────────────────────────────────────────
// Redis
// ──────────────────────────────────────────────

type RedisConfig struct {
	URL      string
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

func DefaultRedisConfig() RedisConfig {
	cfg := RedisConfig{
		URL: os.Getenv("REDIS_URL"), Host: envOr("REDIS_HOST", "localhost"),
		Port: envOr("REDIS_PORT", "6379"), Password: os.Getenv("REDIS_PASSWORD"),
		DB: 0, PoolSize: 10,
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, _ := strconv.Atoi(v); n >= 0 {
			cfg.DB = n
		}
	}
	if v := os.Getenv("REDIS_POOL_SIZE"); v != "" {
		if n, _ := strconv.Atoi(v); n > 0 {
			cfg.PoolSize = n
		}
	}
	return cfg
}

var RedisClient *redis.Client

func ConnectRedis() (*redis.Client, error) {
	cfg := DefaultRedisConfig()
	var client *redis.Client
	if cfg.URL != "" {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_URL failed: %w", err)
		}
		client = redis.NewClient(opt)
	} else {
		client = redis.NewClient(&redis.Options{Addr: cfg.Host + ":" + cfg.Port, Password: cfg.Password, DB: cfg.DB, PoolSize: cfg.PoolSize})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	fmt.Printf("[REDIS] connected successfully (%s:%s/%d)\n", cfg.Host, cfg.Port, cfg.DB)
	RedisClient = client
	return client, nil
}

// ──────────────────────────────────────────────
// DB 包装器（自动 SQL 重写）
// ──────────────────────────────────────────────

type DB struct{ *sql.DB }

func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.DB.Exec(CurrentDialect.RewriteSQL(query), args...)
}
func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(CurrentDialect.RewriteSQL(query), args...)
}
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(CurrentDialect.RewriteSQL(query), args...)
}

// ──────────────────────────────────────────────
// 包级辅助函数
// ──────────────────────────────────────────────

func Exec(db *sql.DB, query string, args ...any) (sql.Result, error) {
	return db.Exec(RewriteSQL(query), args...)
}
func Query(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	return db.Query(RewriteSQL(query), args...)
}
func QueryRow(db *sql.DB, query string, args ...any) *sql.Row {
	return db.QueryRow(RewriteSQL(query), args...)
}
func ExecTx(tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	return tx.Exec(RewriteSQL(query), args...)
}
func QueryTx(tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	return tx.Query(RewriteSQL(query), args...)
}
func QueryRowTx(tx *sql.Tx, query string, args ...any) *sql.Row {
	return tx.QueryRow(RewriteSQL(query), args...)
}
func ExecCtx(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	return db.ExecContext(ctx, RewriteSQL(query), args...)
}
func QueryCtx(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	return db.QueryContext(ctx, RewriteSQL(query), args...)
}
func QueryRowCtx(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	return db.QueryRowContext(ctx, RewriteSQL(query), args...)
}
func ExecTxCtx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(ctx, RewriteSQL(query), args...)
}
func QueryTxCtx(ctx context.Context, tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	return tx.QueryContext(ctx, RewriteSQL(query), args...)
}
func QueryRowTxCtx(ctx context.Context, tx *sql.Tx, query string, args ...any) *sql.Row {
	return tx.QueryRowContext(ctx, RewriteSQL(query), args...)
}

func InsertID(tx *sql.Tx, query string, args ...any) (int64, error) {
	if CurrentDialect == nil || CurrentDialect.SupportsReturning() {
		var id int64
		err := tx.QueryRowContext(context.Background(), RewriteSQL(query), args...).Scan(&id)
		return id, err
	}
	result, err := tx.Exec(RewriteSQL(strings.Replace(query, " RETURNING id", "", -1)), args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func Now() string {
	if CurrentDialect != nil {
		return CurrentDialect.Now()
	}
	return "NOW()"
}
func StringAgg(distinct bool, expr, delimiter string) string {
	if CurrentDialect != nil {
		return CurrentDialect.StringAgg(distinct, expr, delimiter)
	}
	return fmt.Sprintf("GROUP_CONCAT(%s%s SEPARATOR '%s')", map[bool]string{false: "", true: "DISTINCT "}[distinct], expr, delimiter)
}
func RewriteSQL(sql string) string {
	if CurrentDialect != nil {
		return CurrentDialect.RewriteSQL(sql)
	}
	return sql
}

// ──────────────────────────────────────────────
// Connect — 连接数据库并迁移
// ──────────────────────────────────────────────

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

// ──────────────────────────────────────────────
// 迁移
// ──────────────────────────────────────────────

func migrate(db *sql.DB, d *Dialect) error {
	for i, q := range buildCreateTables(d) {
		if _, err := db.Exec(q); err != nil {
			hint := q
			if len(hint) > 100 {
				hint = hint[:100] + "..."
			}
			return fmt.Errorf("create table[%d] error: %w\nSQL: %s", i, err, hint)
		}
	}
	if err := ensureColumns(db, d); err != nil {
		return err
	}
	if err := ensureIndexes(db, d); err != nil {
		return err
	}
	if err := seedRBAC(db, d); err != nil {
		return err
	}
	return nil
}

func buildCreateTables(d *Dialect) []string {
	pk, ts, now := d.AutoIncrement(), d.Timestamp(), d.Now()
	return []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users(id %s,username VARCHAR(100) NOT NULL UNIQUE,email VARCHAR(255) NOT NULL UNIQUE,phone VARCHAR(20) NOT NULL DEFAULT '',password_hash TEXT NOT NULL,password_secret TEXT NOT NULL DEFAULT '',deleted_at %s,created_at %s NOT NULL DEFAULT %s,updated_at %s NOT NULL DEFAULT %s)`, pk, ts, ts, now, ts, now),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS password_reset_tokens(id %s,user_id BIGINT NOT NULL,token VARCHAR(255) NOT NULL UNIQUE,expires_at %s NOT NULL,used_at %s,created_at %s NOT NULL DEFAULT %s,CONSTRAINT fk_reset_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE)`, pk, ts, ts, ts, now),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS refresh_tokens(id %s,user_id BIGINT NOT NULL,token_hash VARCHAR(512) NOT NULL UNIQUE,expires_at %s NOT NULL,revoked_at %s,created_at %s NOT NULL DEFAULT %s,CONSTRAINT fk_refresh_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE)`, pk, ts, ts, ts, now),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS roles(id %s,name VARCHAR(50) NOT NULL UNIQUE,description TEXT NOT NULL DEFAULT '',created_at %s NOT NULL DEFAULT %s,updated_at %s NOT NULL DEFAULT %s)`, pk, ts, now, ts, now),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS permissions(id %s,code VARCHAR(100) NOT NULL UNIQUE,description TEXT NOT NULL DEFAULT '',created_at %s NOT NULL DEFAULT %s)`, pk, ts, now),
		`CREATE TABLE IF NOT EXISTS role_permissions(role_id BIGINT NOT NULL,permission_id BIGINT NOT NULL,PRIMARY KEY(role_id,permission_id),CONSTRAINT fk_rp_role FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE,CONSTRAINT fk_rp_permission FOREIGN KEY(permission_id) REFERENCES permissions(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS user_roles(user_id BIGINT NOT NULL,role_id BIGINT NOT NULL,PRIMARY KEY(user_id,role_id),CONSTRAINT fk_ur_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,CONSTRAINT fk_ur_role FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS chat_messages(id %s,from_user_id BIGINT NOT NULL,to_user_id BIGINT NOT NULL,message_type VARCHAR(20) NOT NULL DEFAULT 'text',content TEXT NOT NULL,media_url VARCHAR(1024) NOT NULL DEFAULT '',file_name VARCHAR(256) NOT NULL DEFAULT '',mime_type VARCHAR(120) NOT NULL DEFAULT '',file_size BIGINT NOT NULL DEFAULT 0,transcript TEXT NOT NULL DEFAULT '',translation TEXT NOT NULL DEFAULT '',created_at %s NOT NULL DEFAULT %s,CONSTRAINT fk_msg_from FOREIGN KEY(from_user_id) REFERENCES users(id),CONSTRAINT fk_msg_to FOREIGN KEY(to_user_id) REFERENCES users(id))`, pk, ts, now),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS operation_logs(id %s,user_id BIGINT NOT NULL DEFAULT 0,username VARCHAR(100) NOT NULL DEFAULT '',action VARCHAR(80) NOT NULL,resource VARCHAR(120) NOT NULL DEFAULT '',detail TEXT NOT NULL DEFAULT '',ip VARCHAR(80) NOT NULL DEFAULT '',user_agent TEXT NOT NULL DEFAULT '',created_at %s NOT NULL DEFAULT %s)`, pk, ts, now),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS notifications(id %s,user_id BIGINT,title VARCHAR(160) NOT NULL,content TEXT NOT NULL DEFAULT '',type VARCHAR(40) NOT NULL DEFAULT 'info',is_read BOOLEAN NOT NULL DEFAULT FALSE,created_at %s NOT NULL DEFAULT %s,read_at %s,CONSTRAINT fk_notice_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE)`, pk, ts, now, ts),
	}
}

func columnExists(db *sql.DB, d *Dialect, table, col string) bool {
	var exists bool
	sqlStr := d.RewriteSQL(d.ColumnExistsSQL())
	err := db.QueryRow(sqlStr, table, col).Scan(&exists)
	return err == nil && exists
}

func ensureColumns(db *sql.DB, d *Dialect) error {
	userCols := map[string]string{
		"phone":           `ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20) NOT NULL DEFAULT ''`,
		"deleted_at":      fmt.Sprintf("ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at %s", d.Timestamp()),
		"password_secret": `ALTER TABLE users ADD COLUMN IF NOT EXISTS password_secret TEXT NOT NULL DEFAULT ''`,
	}
	for col, stmt := range userCols {
		if !columnExists(db, d, "users", col) {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("add user column %s: %w", col, err)
			}
		}
	}
	chatCols := map[string]string{
		"message_type": `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS message_type VARCHAR(20) NOT NULL DEFAULT 'text'`,
		"media_url":    `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS media_url VARCHAR(1024) NOT NULL DEFAULT ''`,
		"file_name":    `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS file_name VARCHAR(256) NOT NULL DEFAULT ''`,
		"mime_type":    `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS mime_type VARCHAR(120) NOT NULL DEFAULT ''`,
		"file_size":    `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS file_size BIGINT NOT NULL DEFAULT 0`,
		"transcript":   `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS transcript TEXT NOT NULL DEFAULT ''`,
		"translation":  `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS translation TEXT NOT NULL DEFAULT ''`,
		"is_read":      `ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS is_read BOOLEAN NOT NULL DEFAULT FALSE`,
	}
	for col, stmt := range chatCols {
		if !columnExists(db, d, "chat_messages", col) {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("add chat column %s: %w", col, err)
			}
		}
	}
	return nil
}

func ensureIndexes(db *sql.DB, d *Dialect) error {
	for _, idx := range buildIndexes(d) {
		if _, err := db.Exec(idx); err != nil {
			errText := strings.ToLower(err.Error())
			if d.Type == DBTypeMySQL && (strings.Contains(errText, "duplicate") || strings.Contains(errText, "already exists")) {
				continue
			}
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
}

func buildIndexes(d *Dialect) []string {
	pgIdx := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users(phone) WHERE phone <> '' AND phone IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_expires ON refresh_tokens(user_id, expires_at) WHERE revoked_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_conv ON chat_messages(LEAST(from_user_id,to_user_id), GREATEST(from_user_id,to_user_id), id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_from ON chat_messages(from_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_to ON chat_messages(to_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_operation_logs_created ON operation_logs(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read, created_at DESC)`,
	}
	mysqlIdx := []string{
		`CREATE INDEX idx_users_phone ON users(phone)`,
		`CREATE INDEX idx_refresh_tokens_user_expires ON refresh_tokens(user_id, expires_at)`,
		`CREATE INDEX idx_chat_messages_from ON chat_messages(from_user_id, created_at DESC)`,
		`CREATE INDEX idx_chat_messages_to ON chat_messages(to_user_id, created_at DESC)`,
		`CREATE INDEX idx_operation_logs_created ON operation_logs(created_at DESC)`,
		`CREATE INDEX idx_notifications_user_read ON notifications(user_id, is_read, created_at DESC)`,
	}
	switch d.Type {
	case DBTypePostgres:
		return pgIdx
	case DBTypeMySQL:
		return mysqlIdx
	default:
		return nil
	}
}

// ──────────────────────────────────────────────
// 种子数据
// ──────────────────────────────────────────────

func seedRBAC(db *sql.DB, d *Dialect) error {
	upsertRole := d.UpsertClause("(name)")
	upsertPerm := d.UpsertClause("(code)")
	upsertRP := d.UpsertClause("(role_id, permission_id)")

	for _, role := range []struct{ n, d string }{{"admin", "System administrator"}, {"user", "Default user"}} {
		sqlStr := d.RewriteSQL(fmt.Sprintf(`INSERT INTO roles(name,description) VALUES($1,$2) %s`, upsertRole))
		if _, err := db.Exec(sqlStr, role.n, role.d); err != nil {
			return fmt.Errorf("insert role %s: %w", role.n, err)
		}
	}
	for _, p := range []struct{ c, d string }{
		{"admin:access", "Access admin APIs"}, {"users:read", "View users"}, {"users:write", "Manage user roles and reset passwords"},
		{"users:password:read", "View decrypted user passwords"}, {"messages:chat", "Use realtime chat"},
		{"roles:read", "View roles and permissions"}, {"roles:write", "Manage roles and role permissions"}, {"permissions:read", "View permissions"},
		{"dashboard:read", "View dashboard data"}, {"logs:read", "View operation logs"},
		{"notifications:read", "View notifications"}, {"notifications:write", "Manage notifications"},
		{"ai:assistant", "Use admin AI assistant"}, {"health:read", "View system health"},
	} {
		sqlStr := d.RewriteSQL(fmt.Sprintf(`INSERT INTO permissions(code,description) VALUES($1,$2) %s`, upsertPerm))
		if _, err := db.Exec(sqlStr, p.c, p.d); err != nil {
			return fmt.Errorf("insert permission %s: %w", p.c, err)
		}
	}
	sqlStr := d.RewriteSQL(fmt.Sprintf(`INSERT INTO role_permissions(role_id,permission_id) SELECT r.id,p.id FROM roles r CROSS JOIN permissions p WHERE r.name='admin' %s`, upsertRP))
	if _, err := db.Exec(sqlStr); err != nil {
		return fmt.Errorf("insert admin-role-permissions: %w", err)
	}
	sqlStr = d.RewriteSQL(fmt.Sprintf(`INSERT INTO role_permissions(role_id,permission_id) SELECT r.id,p.id FROM roles r JOIN permissions p ON p.code='messages:chat' WHERE r.name='user' %s`, upsertRP))
	if _, err := db.Exec(sqlStr); err != nil {
		return fmt.Errorf("insert user-role-permissions: %w", err)
	}
	if err := seedDefaultAdmin(db, d); err != nil {
		return err
	}

	var urSQL string
	if d.Type == DBTypePostgres {
		urSQL = `INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u JOIN roles r ON r.name=CASE WHEN u.id=(SELECT MIN(id) FROM users) THEN 'admin' ELSE 'user' END WHERE NOT EXISTS(SELECT 1 FROM user_roles ur WHERE ur.user_id=u.id) ON CONFLICT DO NOTHING`
	} else {
		urSQL = `INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u,roles r WHERE r.name='admin' AND u.id=(SELECT MIN(id) FROM users) AND NOT EXISTS(SELECT 1 FROM user_roles ur WHERE ur.user_id=u.id)`
	}
	urSQL = d.RewriteSQL(urSQL)
	if _, err := db.Exec(urSQL); err != nil {
		return fmt.Errorf("assign first-user-admin: %w", err)
	}

	urSQL2 := d.RewriteSQL(fmt.Sprintf(`INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u,roles r WHERE r.name='user' AND u.id<>(SELECT MIN(id) FROM users) AND NOT EXISTS(SELECT 1 FROM user_roles ur WHERE ur.user_id=u.id) %s`, d.UpsertClause("(user_id,role_id)")))
	if _, err := db.Exec(urSQL2); err != nil {
		fmt.Printf("[WARN] assign-remaining-users: %v\n", err)
	}
	if err := seedNotifications(db, d); err != nil {
		return err
	}
	return nil
}

func seedNotifications(db *sql.DB, d *Dialect) error {
	var count int
	if err := db.QueryRow(d.RewriteSQL(`SELECT COUNT(*) FROM notifications`)).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	items := []struct{ title, content, typ string }{
		{"\u6b22\u8fce\u4f7f\u7528\u540e\u53f0\u7cfb\u7edf", "\u4eea\u8868\u76d8\u3001\u64cd\u4f5c\u65e5\u5fd7\u548c\u901a\u77e5\u4e2d\u5fc3\u5df2\u51c6\u5907\u5c31\u7eea\u3002", "success"},
		{"\u5b89\u5168\u63d0\u9192", "\u8bf7\u5b9a\u671f\u68c0\u67e5\u7528\u6237\u6743\u9650\u548c\u89d2\u8272\u914d\u7f6e\uff0c\u4fdd\u6301\u8d26\u53f7\u5b89\u5168\u3002", "warning"},
		{"\u6d88\u606f\u4e2d\u5fc3\u4e0a\u7ebf", "\u7cfb\u7edf\u901a\u77e5\u4f1a\u5728\u53f3\u4e0a\u89d2\u94c3\u94db\u548c\u901a\u77e5\u4e2d\u5fc3\u9875\u9762\u540c\u6b65\u5c55\u793a\u3002", "info"},
	}
	for _, item := range items {
		if _, err := db.Exec(d.RewriteSQL(`INSERT INTO notifications(title,content,type) VALUES($1,$2,$3)`), item.title, item.content, item.typ); err != nil {
			return err
		}
	}
	return nil
}

func seedDefaultAdmin(db *sql.DB, d *Dialect) error {
	username := strings.TrimSpace(envOr("DEFAULT_ADMIN_USERNAME", "admin"))
	email := strings.ToLower(envOr("DEFAULT_ADMIN_EMAIL", "admin@example.com"))
	phone := normalizePhone(envOr("DEFAULT_ADMIN_PHONE", "13800000000"))
	password := envOr("DEFAULT_ADMIN_PASSWORD", "admin123")
	if username == "" || email == "" || password == "" {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	checkSQL := d.RewriteSQL(`SELECT id FROM users WHERE email = $1`)
	if err := tx.QueryRow(checkSQL, email).Scan(&userID); err == sql.ErrNoRows {
		checkSQL2 := d.RewriteSQL(`SELECT id FROM users WHERE username = $1`)
		if err := tx.QueryRow(checkSQL2, username).Scan(&userID); err != sql.ErrNoRows {
			return nil
		} // already exists
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		secret, _ := utils.EncryptPassword(password)
		insertSQL := d.RewriteSQL(`INSERT INTO users(username,email,phone,password_hash,password_secret) VALUES($1,$2,$3,$4,$5)`)
		if d.SupportsReturning() {
			err = tx.QueryRow(insertSQL+" RETURNING id", username, email, phone, string(hash), secret).Scan(&userID)
		} else {
			result, _ := tx.Exec(insertSQL, username, email, phone, string(hash), secret)
			userID, _ = result.LastInsertId()
		}
	} else if err != nil {
		return fmt.Errorf("find admin: %w", err)
	}

	var roleID int64
	if err := tx.QueryRow(d.RewriteSQL(`SELECT id FROM roles WHERE name=$1`), "admin").Scan(&roleID); err != nil {
		return err
	}
	if _, err := tx.Exec(d.RewriteSQL(`INSERT INTO user_roles(user_id,role_id) VALUES($1,$2) ON CONFLICT DO NOTHING`), userID, roleID); err != nil {
		return err
	}
	return tx.Commit()
}

// ──────────────────────────────────────────────
// 工具函数
// ──────────────────────────────────────────────

func envOr(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
func normalizePhone(p string) string { return strings.Join(strings.Fields(strings.TrimSpace(p)), "") }
