package migrate

import (
	"context"
	"database/sql"
	"io"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent/proxyschemamigrations"
	"github.com/npanel-dev/NPanel-backend/internal/database"
)

func TestPortableBaselineAgainstDatabase(t *testing.T) {
	tests := []struct {
		name   string
		driver database.Driver
		env    string
	}{
		{name: "mysql", driver: database.MySQL, env: "NPANEL_TEST_MYSQL_DSN"},
		{name: "postgres", driver: database.Postgres, env: "NPANEL_TEST_POSTGRES_DSN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := os.Getenv(tt.env)
			if dsn == "" {
				t.Skipf("%s is not set", tt.env)
			}
			client, err := database.OpenEnt(tt.driver, dsn)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = client.Close() })

			ctx := context.Background()
			migrator := NewMigrator(client, log.NewStdLogger(io.Discard), nil, tt.driver.String(), dsn)
			for run := 1; run <= 2; run++ {
				if err := migrator.AutoMigrateWithData(ctx); err != nil {
					t.Fatalf("initialization run %d: %v", run, err)
				}
			}

			assertCount(t, "auth methods", client.ProxyAuthMethod.Query().CountX(ctx), 8)
			assertCount(t, "payments", client.ProxyPayment.Query().CountX(ctx), 1)
			assertCount(t, "system settings", client.ProxySystem.Query().CountX(ctx), 61)
			assertCount(t, "applications", client.ProxySubscribeApplication.Query().CountX(ctx), 6)
			assertCount(t, "users", client.ProxyUser.Query().CountX(ctx), 1)

			if !client.ProxySchemaMigrations.Query().
				Where(proxyschemamigrations.IDEQ(currentSchemaVersion)).
				ExistX(ctx) {
				t.Fatalf("schema version %d was not recorded", currentSchemaVersion)
			}

			sqlDB, err := database.OpenSQL(tt.driver, dsn)
			if err != nil {
				t.Fatalf("open database for index verification: %v", err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })
			assertIndexExists(t, ctx, sqlDB, tt.driver, "proxyredemptionrecord_user_id")
			assertIndexExists(t, ctx, sqlDB, tt.driver, "proxyredemptionrecord_redemption_code_id")
			assertIndexExists(t, ctx, sqlDB, tt.driver, "proxyuserwithdrawal_user_id")

			application, err := client.ProxySubscribeApplication.Create().
				SetName("Sequence Test").
				SetScheme("sequence-test").
				SetUserAgent("sequence-test").
				SetOutputFormat("yaml").
				SetDownloadLink("").
				Save(ctx)
			if err != nil {
				t.Fatalf("create application with generated id: %v", err)
			}
			t.Cleanup(func() {
				_ = client.ProxySubscribeApplication.DeleteOneID(application.ID).Exec(context.Background())
			})
			if application.ID <= 1001 {
				t.Fatalf("generated application id = %d, want > 1001", application.ID)
			}
		})
	}
}

func assertIndexExists(t *testing.T, ctx context.Context, db *sql.DB, driver database.Driver, indexName string) {
	t.Helper()
	var exists bool
	var err error
	if driver == database.Postgres {
		err = db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = ANY(current_schemas(false)) AND indexname = $1)`,
			indexName,
		).Scan(&exists)
	} else {
		var count int
		err = db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND index_name = ?`,
			indexName,
		).Scan(&count)
		exists = count > 0
	}
	if err != nil {
		t.Fatalf("query index %s: %v", indexName, err)
	}
	if !exists {
		t.Fatalf("index %s does not exist", indexName)
	}
}

func assertCount(t *testing.T, label string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s count = %d, want %d", label, got, want)
	}
}
