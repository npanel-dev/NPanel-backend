package data

import (
	"strings"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/npanel-dev/NPanel-backend/internal/database"
)

func renderPredicate(driver database.Driver, predicate *entsql.Predicate) (string, []any) {
	table := entsql.Table("nodes")
	selector := entsql.Dialect(driver.String()).Select().From(table).Where(predicate)
	return selector.Query()
}

func TestJSONContainsPredicateByDialect(t *testing.T) {
	mysqlQuery, _ := renderPredicate(database.MySQL, jsonArrayContainsInt64(database.MySQL, "node_group_ids", 7))
	if !strings.Contains(mysqlQuery, "JSON_CONTAINS") || !strings.Contains(mysqlQuery, "?") {
		t.Fatalf("unexpected MySQL predicate: %s", mysqlQuery)
	}

	postgresQuery, args := renderPredicate(database.Postgres, jsonArrayContainsInt64(database.Postgres, "node_group_ids", 7))
	if !strings.Contains(postgresQuery, "@> $1::jsonb") {
		t.Fatalf("unexpected PostgreSQL predicate: %s", postgresQuery)
	}
	if len(args) != 1 || args[0] != "[7]" {
		t.Fatalf("unexpected PostgreSQL args: %#v", args)
	}
}

func TestCSVContainsPredicateByDialect(t *testing.T) {
	mysqlQuery, _ := renderPredicate(database.MySQL, csvContains(database.MySQL, "tags", "news"))
	if !strings.Contains(mysqlQuery, "FIND_IN_SET") {
		t.Fatalf("unexpected MySQL predicate: %s", mysqlQuery)
	}

	postgresQuery, _ := renderPredicate(database.Postgres, csvContains(database.Postgres, "tags", "news"))
	if !strings.Contains(postgresQuery, "ANY(string_to_array") {
		t.Fatalf("unexpected PostgreSQL predicate: %s", postgresQuery)
	}
}

func TestNumericColumnLikeByDialect(t *testing.T) {
	mysqlQuery, _ := renderPredicate(database.MySQL, numericColumnLike(database.MySQL, "port", "%443%"))
	if !strings.Contains(mysqlQuery, "AS CHAR") {
		t.Fatalf("unexpected MySQL predicate: %s", mysqlQuery)
	}

	postgresQuery, _ := renderPredicate(database.Postgres, numericColumnLike(database.Postgres, "port", "%443%"))
	if !strings.Contains(postgresQuery, "AS TEXT") {
		t.Fatalf("unexpected PostgreSQL predicate: %s", postgresQuery)
	}
}
