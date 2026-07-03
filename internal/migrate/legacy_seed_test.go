package migrate

import (
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

func TestNormalizeLegacyStatementSkipsObsoleteSubscribeTypeSeed(t *testing.T) {
	stmt := "INSERT IGNORE INTO `subscribe_type` (`id`, `name`) VALUES (1, 'Clash')"
	if got := normalizeLegacyStatement("legacy_sql/00002_init_basic_data.up.sql", stmt); got != "" {
		t.Fatalf("expected obsolete subscribe_type seed to be skipped, got %q", got)
	}
}

func TestNormalizeLegacyStatementUsesInsertIgnoreForSubscribeApplication(t *testing.T) {
	stmt := "INSERT INTO `subscribe_application` (`id`, `name`) VALUES (1, 'Default')"
	want := "INSERT IGNORE INTO `subscribe_application` (`id`, `name`) VALUES (1, 'Default')"
	if got := normalizeLegacyStatement("legacy_sql/02101_subscribe_application.up.sql", stmt); got != want {
		t.Fatalf("unexpected normalized statement: got %q want %q", got, want)
	}
}

func TestLegacySQLMigrationsIncludeSyncedLatestVersions(t *testing.T) {
	want := []int64{2133, 2134, 2135, 2136, 2137, 2138, 2139, 2140, 2141, 2142, 2143, 2144, 2145, 2146, 2147, 2148, 2149, 2150, 2151, 2152, 2153, 2154}
	got := make([]int64, 0, len(want))
	for _, migration := range legacySQLMigrations {
		if migration.version >= 2133 {
			got = append(got, migration.version)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected synced migration count: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected synced migration versions: got %v want %v", got, want)
		}
	}
}

func TestLegacyCompatibilityMigrationVersionsAvoidDestructiveGroupMigration(t *testing.T) {
	if _, ok := legacyCompatibilityMigrationVersions[2131]; ok {
		t.Fatal("02131_add_groups must not run as a repeatable compatibility migration because it drops group tables")
	}
	if _, ok := legacyCompatibilityMigrationVersions[2141]; !ok {
		t.Fatal("expected subscribe category migration to remain in compatibility migration set")
	}
}

func TestLegacyRequiredSchemaPatchesCoverServerAndNodeListColumns(t *testing.T) {
	want := map[string]bool{
		"servers.last_reported_at": false,
		"servers.longitude":        false,
		"servers.latitude":         false,
		"servers.longitude_center": false,
		"servers.latitude_center":  false,
		"nodes.node_group_ids":     false,
		"nodes.node_type":          false,
		"nodes.is_hidden":          false,
		"nodes.target_node_id":     false,
	}

	for _, patch := range legacyRequiredSchemaPatches {
		key := patch.table + "." + patch.column
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}

	for key, found := range want {
		if !found {
			t.Fatalf("missing legacy required schema patch for %s", key)
		}
	}
}

func TestShouldIgnoreLegacySQLErrorAllowsRepeatableSubscribeDefaultConstraint(t *testing.T) {
	path := "legacy_sql/02143_subscribe_defaults_and_language_normalization.up.sql"
	if !shouldIgnoreLegacySQLError(path, "ALTER TABLE `subscribe_application` ADD COLUMN `default_unique_key` TINYINT", &mysql.MySQLError{Number: 1060}) {
		t.Fatal("expected duplicate generated column error to be ignored for repeatable compatibility migration")
	}
	if !shouldIgnoreLegacySQLError(path, "ALTER TABLE `subscribe_application` ADD UNIQUE INDEX `uniq_subscribe_application_default` (`default_unique_key`)", &mysql.MySQLError{Number: 1061}) {
		t.Fatal("expected duplicate default unique index error to be ignored for repeatable compatibility migration")
	}
}
