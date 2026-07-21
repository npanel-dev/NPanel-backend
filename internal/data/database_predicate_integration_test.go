package data

import (
	"context"
	"os"
	"strconv"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/npanel-dev/NPanel-backend/ent/proxycoupon"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/internal/database"
)

func TestDatabasePredicatesAgainstDatabase(t *testing.T) {
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
			if err := client.Schema.Create(ctx); err != nil {
				t.Fatalf("create schema: %v", err)
			}

			node, err := client.ProxyNode.Create().
				SetName("predicate-test-node").
				SetNodeGroupIds([]int64{42, 99}).
				Save(ctx)
			if err != nil {
				t.Fatalf("create node: %v", err)
			}
			t.Cleanup(func() { _ = client.ProxyNode.DeleteOneID(node.ID).Exec(context.Background()) })

			coupon, err := client.ProxyCoupon.Create().
				SetName("predicate-test-coupon").
				SetCode("predicate-test-" + tt.name).
				SetSubscribe("1,42,100").
				Save(ctx)
			if err != nil {
				t.Fatalf("create coupon: %v", err)
			}
			t.Cleanup(func() { _ = client.ProxyCoupon.DeleteOneID(coupon.ID).Exec(context.Background()) })

			jsonMatch, err := client.ProxyNode.Query().
				Where(func(selector *entsql.Selector) {
					selector.Where(jsonArrayContainsInt64(tt.driver, proxynode.FieldNodeGroupIds, 42))
				}).
				Exist(ctx)
			if err != nil || !jsonMatch {
				t.Fatalf("JSON array predicate: match=%v err=%v", jsonMatch, err)
			}

			csvMatch, err := client.ProxyCoupon.Query().
				Where(func(selector *entsql.Selector) {
					selector.Where(csvContains(tt.driver, proxycoupon.FieldSubscribe, "42"))
				}).
				Exist(ctx)
			if err != nil || !csvMatch {
				t.Fatalf("CSV predicate: match=%v err=%v", csvMatch, err)
			}

			caseFoldMatch, err := client.ProxyCoupon.Query().
				Where(proxycoupon.NameContainsFold("PREDICATE-TEST")).
				Exist(ctx)
			if err != nil || !caseFoldMatch {
				t.Fatalf("case-insensitive text predicate: match=%v err=%v", caseFoldMatch, err)
			}

			idMatch, err := client.ProxyNode.Query().
				Where(func(selector *entsql.Selector) {
					selector.Where(numericColumnLike(tt.driver, proxynode.FieldID, "%"+strconv.FormatInt(node.ID, 10)+"%"))
				}).
				Exist(ctx)
			if err != nil || !idMatch {
				t.Fatalf("numeric LIKE predicate: match=%v err=%v", idMatch, err)
			}
		})
	}
}
