package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

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
	dist := ""
	if distinct {
		dist = "DISTINCT "
	}
	return fmt.Sprintf("GROUP_CONCAT(%s%s SEPARATOR '%s')", dist, expr, delimiter)
}

func RewriteSQL(sql string) string {
	if CurrentDialect != nil {
		return CurrentDialect.RewriteSQL(sql)
	}
	return sql
}
