package data

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/enttest"
	withdrawalbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/withdrawal"

	_ "github.com/mattn/go-sqlite3"
)

func TestRejectWithdrawalRefundsCommissionOnce(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:admin_withdrawal_reject_once?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	createWithdrawalUser(t, client, 100, 200)
	client.ProxyUserWithdrawal.Create().
		SetID(1).
		SetUserID(100).
		SetAmount(50).
		SetMethod("usdt").
		SetContent("wallet").
		SetStatus(withdrawalbiz.StatusPending).
		SaveX(ctx)

	repo := NewAdminWithdrawalRepo(&Data{db: client}, log.NewStdLogger(io.Discard))
	if err := repo.RejectWithdrawal(ctx, 1, "invalid account"); err != nil {
		t.Fatalf("first RejectWithdrawal() error = %v", err)
	}
	if err := repo.RejectWithdrawal(ctx, 1, "again"); err == nil {
		t.Fatal("second RejectWithdrawal() error = nil, want error")
	}

	user := client.ProxyUser.GetX(ctx, 100)
	if user.Commission == nil || *user.Commission != 250 {
		t.Fatalf("commission = %v, want 250", user.Commission)
	}

	record := client.ProxyUserWithdrawal.GetX(ctx, 1)
	if record.Status != withdrawalbiz.StatusRejected {
		t.Fatalf("status = %d, want %d", record.Status, withdrawalbiz.StatusRejected)
	}
}

func createWithdrawalUser(t *testing.T, client *ent.Client, id int64, commission int64) {
	t.Helper()
	client.ProxyUser.Create().
		SetID(id).
		SetPassword("password").
		SetCommission(commission).
		SaveX(context.Background())
}
