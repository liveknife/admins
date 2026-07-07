package database

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

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
