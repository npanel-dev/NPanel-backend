package migrate

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
)

//go:embed legacy_sql/*.sql
var legacySQLFS embed.FS

type legacySQLMigration struct {
	version int64
	path    string
}

var legacySQLMigrations = []legacySQLMigration{
	{version: 2, path: "legacy_sql/00002_init_basic_data.up.sql"},
	{version: 2101, path: "legacy_sql/02101_subscribe_application.up.sql"},
	{version: 2102, path: "legacy_sql/02102_subscribe_config.up.sql"},
	{version: 2107, path: "legacy_sql/02107_log_setting.up.sql"},
	{version: 2114, path: "legacy_sql/02114_node_config.up.sql"},
	{version: 2117, path: "legacy_sql/02117_site_custom_data.up.sql"},
	{version: 2122, path: "legacy_sql/02122_user_withdrawal.up.sql"},
	{version: 2128, path: "legacy_sql/02128_device_limit.up.sql"},
	{version: 2131, path: "legacy_sql/02131_add_groups.up.sql"},
	{version: 2132, path: "legacy_sql/02132_update_verify_config.up.sql"},
	{version: 2133, path: "legacy_sql/02133_add_expired_node_group.up.sql"},
	{version: 2134, path: "legacy_sql/02134_subscribe_traffic_limit.up.sql"},
	{version: 2135, path: "legacy_sql/02135_add_node_group_type.up.sql"},
	{version: 2136, path: "legacy_sql/02136_add_node_type_is_hidden.up.sql"},
	{version: 2137, path: "legacy_sql/02137_payment_sort.up.sql"},
	{version: 2138, path: "legacy_sql/02138_add_simnet_subscribe_application.up.sql"},
	{version: 2139, path: "legacy_sql/02139_device_admission_control.up.sql"},
	{version: 2140, path: "legacy_sql/02140_update_simnet_subscribe_application_format.up.sql"},
	{version: 2141, path: "legacy_sql/02141_subscribe_category.up.sql"},
	{version: 2142, path: "legacy_sql/02142_subscribe_price_option.up.sql"},
	{version: 2143, path: "legacy_sql/02143_subscribe_defaults_and_language_normalization.up.sql"},
	{version: 2144, path: "legacy_sql/02144_routing_tables.up.sql"},
	{version: 2145, path: "legacy_sql/02145_routing_health_report.up.sql"},
	{version: 2146, path: "legacy_sql/02146_routing_route_event.up.sql"},
	{version: 2147, path: "legacy_sql/02147_routing_gray_release.up.sql"},
	{version: 2148, path: "legacy_sql/02148_subscribe_description_content.up.sql"},
	{version: 2149, path: "legacy_sql/02149_subscribe_price_option_code_type.up.sql"},
	{version: 2150, path: "legacy_sql/02150_subscribe_price_option_version.up.sql"},
	{version: 2151, path: "legacy_sql/02151_archive_duplicate_duration_price_options.up.sql"},
}

func (m *Migrator) initLegacyDefaultData(ctx context.Context) error {
	if m.dbDriver == "" || m.dbSource == "" {
		return fmt.Errorf("database connection info is empty")
	}

	db, err := sql.Open(m.dbDriver, m.dbSource)
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}
	defer db.Close()

	currentVersion, err := getLegacyMigrationVersion(ctx, db)
	if err != nil {
		return err
	}

	for _, migration := range legacySQLMigrations {
		if migration.version <= currentVersion {
			continue
		}
		if err := m.executeLegacySQLMigration(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) executeLegacySQLMigration(ctx context.Context, db *sql.DB, migration legacySQLMigration) (err error) {
	return m.executeLegacySQLMigrationWithVersion(ctx, db, migration, true)
}

func (m *Migrator) executeLegacySQLMigrationWithVersion(ctx context.Context, db *sql.DB, migration legacySQLMigration, recordVersion bool) (err error) {
	content, err := legacySQLFS.ReadFile(migration.path)
	if err != nil {
		return fmt.Errorf("read legacy sql %s: %w", migration.path, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy sql tx %s: %w", migration.path, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	statements := splitSQLStatements(stripLegacyCommentLines(string(content)))
	for _, stmt := range statements {
		stmt = normalizeLegacyStatement(migration.path, stmt)
		if !shouldExecuteLegacyStatement(stmt) {
			continue
		}
		if _, err = tx.ExecContext(ctx, stmt); err != nil {
			if shouldIgnoreLegacySQLError(migration.path, stmt, err) {
				m.logger.Warnf("ignore legacy sql error: path=%s err=%v stmt=%s", migration.path, err, stmt)
				continue
			}
			return fmt.Errorf("exec legacy sql %s failed: %w", migration.path, err)
		}
	}

	if recordVersion {
		if err = setLegacyMigrationVersion(ctx, tx, migration.version); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy sql %s failed: %w", migration.path, err)
	}

	if recordVersion {
		m.logger.Infof("legacy sql migration applied: version=%d path=%s", migration.version, migration.path)
	} else {
		m.logger.Infof("legacy compatibility schema ensured: version=%d path=%s", migration.version, migration.path)
	}
	return nil
}

// EnsureLegacyCompatibilitySchema applies idempotent schema patches needed by
// new code paths when an imported legacy database has no schema_migrations table.
func (m *Migrator) EnsureLegacyCompatibilitySchema(ctx context.Context) error {
	if m.dbDriver == "" || m.dbSource == "" {
		return fmt.Errorf("database connection info is empty")
	}

	db, err := sql.Open(m.dbDriver, m.dbSource)
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}
	defer db.Close()

	for _, migration := range legacySQLMigrations {
		if migration.version != 2141 && migration.version != 2142 && migration.version != 2143 && migration.version != 2144 && migration.version != 2145 && migration.version != 2146 && migration.version != 2147 && migration.version != 2148 && migration.version != 2149 && migration.version != 2150 && migration.version != 2151 {
			continue
		}
		if err := m.executeLegacySQLMigrationWithVersion(ctx, db, migration, false); err != nil {
			return err
		}
	}

	return nil
}

func getLegacyMigrationVersion(ctx context.Context, db *sql.DB) (int64, error) {
	row := db.QueryRowContext(ctx, "SELECT `version`, `dirty` FROM `schema_migrations` ORDER BY `version` DESC LIMIT 1")

	var version int64
	var dirty bool
	err := row.Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query schema_migrations failed: %w", err)
	}
	if dirty {
		return 0, fmt.Errorf("schema_migrations is dirty at version %d", version)
	}
	return version, nil
}

func setLegacyMigrationVersion(ctx context.Context, tx *sql.Tx, version int64) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM `schema_migrations`"); err != nil {
		return fmt.Errorf("cleanup schema_migrations failed: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO `schema_migrations` (`version`, `dirty`) VALUES (?, ?)", version, false); err != nil {
		return fmt.Errorf("insert schema_migrations version %d failed: %w", version, err)
	}
	return nil
}

func stripLegacyCommentLines(content string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func splitSQLStatements(content string) []string {
	statements := make([]string, 0)
	var builder strings.Builder
	inSingleQuote := false
	escaped := false

	for _, r := range content {
		builder.WriteRune(r)
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '\'':
			inSingleQuote = !inSingleQuote
		case r == ';' && !inSingleQuote:
			stmt := strings.TrimSpace(builder.String())
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			builder.Reset()
		}
	}

	if tail := strings.TrimSpace(builder.String()); tail != "" {
		statements = append(statements, tail)
	}
	return statements
}

func normalizeLegacyStatement(path, stmt string) string {
	stmt = strings.TrimSpace(stmt)
	if path == "legacy_sql/00002_init_basic_data.up.sql" && strings.Contains(stmt, "`subscribe_type`") {
		// subscribe_type belonged to the legacy project, but the current schema
		// replaced it with subscribe_application and no longer creates this table.
		// Keep the embedded SQL as a reference snapshot while skipping the obsolete seed.
		return ""
	}
	if path == "legacy_sql/02101_subscribe_application.up.sql" {
		stmt = strings.Replace(stmt, "INSERT INTO `subscribe_application`", "INSERT IGNORE INTO `subscribe_application`", 1)
	}
	return stmt
}

func shouldExecuteLegacyStatement(stmt string) bool {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	return strings.HasPrefix(upper, "SET ") ||
		strings.HasPrefix(upper, "ALTER ") ||
		strings.HasPrefix(upper, "CREATE ") ||
		strings.HasPrefix(upper, "DROP ") ||
		strings.HasPrefix(upper, "INSERT ") ||
		strings.HasPrefix(upper, "DELETE ") ||
		strings.HasPrefix(upper, "UPDATE ") ||
		strings.HasPrefix(upper, "PREPARE ") ||
		strings.HasPrefix(upper, "EXECUTE ") ||
		strings.HasPrefix(upper, "DEALLOCATE ")
}

func shouldIgnoreLegacySQLError(path, stmt string, err error) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	switch mysqlErr.Number {
	case 1060:
		return strings.HasPrefix(path, "legacy_sql/02133_") ||
			path == "legacy_sql/02135_add_node_group_type.up.sql" ||
			path == "legacy_sql/02136_add_node_type_is_hidden.up.sql" ||
			path == "legacy_sql/02137_payment_sort.up.sql" ||
			path == "legacy_sql/02143_subscribe_defaults_and_language_normalization.up.sql"
	case 1061:
		return path == "legacy_sql/02133_add_expired_node_group.up.sql" ||
			path == "legacy_sql/02143_subscribe_defaults_and_language_normalization.up.sql"
	case 1091:
		return path == "legacy_sql/02137_payment_sort.up.sql"
	}

	upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
	if strings.HasPrefix(upperStmt, "ALTER TABLE") && strings.Contains(strings.ToLower(mysqlErr.Message), "duplicate") {
		return strings.HasPrefix(path, "legacy_sql/02133_") ||
			path == "legacy_sql/02135_add_node_group_type.up.sql" ||
			path == "legacy_sql/02136_add_node_type_is_hidden.up.sql" ||
			path == "legacy_sql/02137_payment_sort.up.sql"
	}

	return false
}
