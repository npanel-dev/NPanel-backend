package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/npanel-dev/NPanel-backend/ent"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Driver string

const (
	MySQL    Driver = "mysql"
	Postgres Driver = "postgres"
)

func NormalizeDriver(value string) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(MySQL):
		return MySQL, nil
	case string(Postgres), "postgresql":
		return Postgres, nil
	default:
		return "", fmt.Errorf("unsupported database driver %q: expected mysql or postgres", value)
	}
}

func (d Driver) String() string {
	return string(d)
}

func (d Driver) sqlDriver() string {
	if d == Postgres {
		return "pgx"
	}
	return string(d)
}

func OpenSQL(driver Driver, source string) (*sql.DB, error) {
	if _, err := NormalizeDriver(driver.String()); err != nil {
		return nil, err
	}
	db, err := sql.Open(driver.sqlDriver(), source)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", driver, err)
	}
	return db, nil
}

func OpenEnt(driver Driver, source string) (*ent.Client, error) {
	db, err := OpenSQL(driver, source)
	if err != nil {
		return nil, err
	}
	drv := entsql.OpenDB(driver.String(), db)
	return ent.NewClient(ent.Driver(drv)), nil
}

func TableExists(ctx context.Context, db *sql.DB, driver Driver, tableName string) (bool, error) {
	var exists bool
	switch driver {
	case MySQL:
		var count int
		err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`,
			tableName,
		).Scan(&count)
		return count > 0, err
	case Postgres:
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = ANY(current_schemas(false)) AND table_name = $1)`,
			tableName,
		).Scan(&exists)
		return exists, err
	default:
		return false, fmt.Errorf("unsupported database driver %q", driver)
	}
}

func ColumnExists(ctx context.Context, db *sql.DB, driver Driver, tableName, columnName string) (bool, error) {
	var exists bool
	switch driver {
	case MySQL:
		var count int
		err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			tableName,
			columnName,
		).Scan(&count)
		return count > 0, err
	case Postgres:
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = ANY(current_schemas(false)) AND table_name = $1 AND column_name = $2)`,
			tableName,
			columnName,
		).Scan(&exists)
		return exists, err
	default:
		return false, fmt.Errorf("unsupported database driver %q", driver)
	}
}
