package data

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/enttest"
	systemlog "github.com/npanel-dev/NPanel-backend/internal/model/log"

	_ "github.com/mattn/go-sqlite3"
)

func TestQueryUserAffiliateTotalsOnlyEarnedCommission(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:public_user_affiliate_total?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	createAffiliateUser(t, client, 100, nil)
	refererID := int64(100)
	createAffiliateUser(t, client, 101, &refererID)
	createCommissionLog(t, client, 1, 100, systemlog.CommissionTypePurchase, 120)
	createCommissionLog(t, client, 2, 100, systemlog.CommissionTypeRenewal, 80)
	createCommissionLog(t, client, 3, 100, systemlog.CommissionTypeWithdraw, 50)
	createCommissionLog(t, client, 4, 100, systemlog.CommissionTypeConvertBalance, 30)
	createCommissionLog(t, client, 5, 100, systemlog.CommissionTypeRefund, 10)

	repo := NewPublicUserRepo(&Data{db: client}, log.NewStdLogger(io.Discard))
	registers, total, err := repo.QueryUserAffiliate(ctx, 100)
	if err != nil {
		t.Fatalf("QueryUserAffiliate() error = %v", err)
	}
	if registers != 1 {
		t.Fatalf("registers = %d, want 1", registers)
	}
	if total != 200 {
		t.Fatalf("total commission = %d, want 200", total)
	}
}

func createAffiliateUser(t *testing.T, client *ent.Client, id int64, refererID *int64) {
	t.Helper()
	builder := client.ProxyUser.Create().
		SetID(id).
		SetPassword("password")
	if refererID != nil {
		builder.SetRefererID(*refererID)
	}
	builder.SaveX(context.Background())
}

func createCommissionLog(t *testing.T, client *ent.Client, id int64, userID int64, typ uint16, amount int64) {
	t.Helper()
	payload, err := (&systemlog.Commission{
		Type:      typ,
		Amount:    amount,
		Timestamp: time.Now().UnixMilli(),
	}).Marshal()
	if err != nil {
		t.Fatalf("marshal commission log: %v", err)
	}
	client.ProxySystemLog.Create().
		SetID(id).
		SetType(int8(systemlog.TypeCommission)).
		SetDate(time.Now().Format(time.DateOnly)).
		SetObjectID(userID).
		SetContent(string(payload)).
		SaveX(context.Background())
}
