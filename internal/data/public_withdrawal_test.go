package data

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent/enttest"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserwithdrawal"

	_ "github.com/mattn/go-sqlite3"
)

func TestProcessCommissionWithdrawDoesNotOverdrawDuplicateRequests(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:public_withdrawal_no_overdraw?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	createWithdrawalUser(t, client, 200, 100)

	repo := NewWithdrawalRepo(&Data{db: client}, log.NewStdLogger(io.Discard))
	if _, err := repo.ProcessCommissionWithdraw(ctx, 200, 70, "usdt", "wallet"); err != nil {
		t.Fatalf("first ProcessCommissionWithdraw() error = %v", err)
	}
	if _, err := repo.ProcessCommissionWithdraw(ctx, 200, 70, "usdt", "wallet"); err == nil {
		t.Fatal("second ProcessCommissionWithdraw() error = nil, want insufficient commission")
	}

	user := client.ProxyUser.GetX(ctx, 200)
	if user.Commission == nil || *user.Commission != 30 {
		t.Fatalf("commission = %v, want 30", user.Commission)
	}
	count := client.ProxyUserWithdrawal.Query().
		Where(proxyuserwithdrawal.UserID(200)).
		CountX(ctx)
	if count != 1 {
		t.Fatalf("withdrawal count = %d, want 1", count)
	}
}
