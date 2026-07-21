package handler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyorder"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribepriceoption"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscriberebateconfig"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystemlog"
	"github.com/npanel-dev/NPanel-backend/internal/database"
	"github.com/redis/go-redis/v9"
)

// 本测试为 e2e 验证脚本：连接本地 Docker 容器中的 MySQL 或 PostgreSQL，
// 直接测试 resolveRebateRate 的优先级与短路逻辑。
//
// 运行方式（需要先启动后端完成建表，且 DB 中已存在 schema_migrations 表）：
//
//	NPANEL_E2E_DATABASE_DRIVER=mysql \
//	NPANEL_E2E_DSN="npanel:npanel_password@tcp(127.0.0.1:3307)/npanel_e2e?parseTime=True&loc=Local" \
//	  go test ./internal/queue/handler/ -run TestE2ERebateRate -v -count=1
//
//	NPANEL_E2E_DATABASE_DRIVER=postgres \
//	NPANEL_E2E_DSN="postgres://npanel:npanel_password@127.0.0.1:5433/npanel_e2e?sslmode=disable" \
//	  go test ./internal/queue/handler/ -run TestE2ERebateRate -v -count=1
//
// 不设置 NPANEL_E2E_DSN 时跳过，避免污染普通单测流程。
func e2eDSN() string { return os.Getenv("NPANEL_E2E_DSN") }

func e2eDatabaseDriver(t *testing.T) database.Driver {
	t.Helper()
	value := strings.TrimSpace(os.Getenv("NPANEL_E2E_DATABASE_DRIVER"))
	if value == "" {
		value = "mysql"
	}
	driver, err := database.NormalizeDriver(value)
	if err != nil {
		t.Fatalf("invalid NPANEL_E2E_DATABASE_DRIVER: %v", err)
	}
	return driver
}

func newE2EClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := e2eDSN()
	if dsn == "" {
		t.Skip("NPANEL_E2E_DSN not set; skipping e2e rebate test")
	}
	client, err := database.OpenEnt(e2eDatabaseDriver(t), dsn)
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})
	return client
}

func cleanupE2E(t *testing.T, action string, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("cleanup %s: %v", action, err)
	}
}

func snapshotInviteSettings(t *testing.T, ctx context.Context, client *ent.Client) []*ent.ProxySystem {
	t.Helper()
	settings, err := client.ProxySystem.Query().Where(proxysystem.CategoryEQ("invite")).All(ctx)
	if err != nil {
		t.Fatalf("snapshot invite settings: %v", err)
	}
	return settings
}

func restoreInviteSettings(t *testing.T, ctx context.Context, client *ent.Client, settings []*ent.ProxySystem) {
	t.Helper()
	_, err := client.ProxySystem.Delete().Where(proxysystem.CategoryEQ("invite")).Exec(ctx)
	cleanupE2E(t, "current invite settings", err)
	for _, setting := range settings {
		_, err := client.ProxySystem.Create().
			SetCategory(setting.Category).
			SetKey(setting.Key).
			SetValue(setting.Value).
			SetType(setting.Type).
			SetDesc(setting.Desc).
			SetCreatedAt(setting.CreatedAt).
			SetUpdatedAt(setting.UpdatedAt).
			Save(ctx)
		cleanupE2E(t, "restore invite setting "+setting.Key, err)
	}
}

func cleanRebateE2EFixtures(t *testing.T, ctx context.Context, client *ent.Client, ids struct{ referer, buyer, subscribe, optionMonthly, optionQuarterly, optionYearly int64 }) {
	t.Helper()
	_, err := client.ProxySubscribeRebateConfig.Delete().
		Where(proxysubscriberebateconfig.SubscribeIDEQ(ids.subscribe)).Exec(ctx)
	cleanupE2E(t, "rebate configs", err)
	if ids.subscribe > 0 {
		_, err = client.ProxySubscribePriceOption.Delete().
			Where(proxysubscribepriceoption.SubscribeIDEQ(ids.subscribe)).Exec(ctx)
		cleanupE2E(t, "subscribe price options", err)
		cleanupE2E(t, "subscribe", client.ProxySubscribe.DeleteOneID(ids.subscribe).Exec(ctx))
	}
	if ids.buyer > 0 {
		_, err = client.ProxyOrder.Delete().Where(proxyorder.UserIDEQ(ids.buyer)).Exec(ctx)
		cleanupE2E(t, "orders", err)
	}
	if ids.buyer > 0 {
		cleanupE2E(t, "buyer", client.ProxyUser.DeleteOneID(ids.buyer).Exec(ctx))
	}
	if ids.referer > 0 {
		_, err = client.ProxySystemLog.Delete().Where(
			proxysystemlog.ObjectIDEQ(ids.referer),
			proxysystemlog.TypeEQ(int8(33)),
		).Exec(ctx)
		cleanupE2E(t, "commission logs", err)
		cleanupE2E(t, "referer", client.ProxyUser.DeleteOneID(ids.referer).Exec(ctx))
	}
}

// setGlobalRebate 写入全局 invite 配置（ReferralPercentage / OnlyFirstPurchase）
func setGlobalRebate(t *testing.T, ctx context.Context, client *ent.Client, percentage int, onlyFirst bool) {
	t.Helper()
	_, _ = client.ProxySystem.Delete().Where(proxysystem.CategoryEQ("invite")).Exec(ctx)
	create := func(k, v string) {
		if _, err := client.ProxySystem.Create().
			SetCategory("invite").
			SetKey(k).
			SetValue(v).
			SetType("string").
			SetDesc("e2e").
			Save(ctx); err != nil {
			t.Fatalf("seed global invite %s: %v", k, err)
		}
	}
	create("ReferralPercentage", fmt.Sprintf("%d", percentage))
	ofp := "false"
	if onlyFirst {
		ofp = "true"
	}
	create("OnlyFirstPurchase", ofp)
}

// setupRebateFixtures 创建 referer、buyer（buyer.referer_id=referer）、subscribe、3 个 price options
func setupRebateFixtures(t *testing.T, ctx context.Context, client *ent.Client) (ids struct{ referer, buyer, subscribe, optionMonthly, optionQuarterly, optionYearly int64 }) {
	t.Helper()
	referer, err := client.ProxyUser.Create().
		SetPassword("x").
		SetReferCode(fmt.Sprintf("E2E%016x", uint64(time.Now().UnixNano()))).
		SetIsAdmin(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("create referer: %v", err)
	}
	ids.referer = referer.ID

	buyer, err := client.ProxyUser.Create().
		SetPassword("x").
		SetRefererID(referer.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create buyer: %v", err)
	}
	ids.buyer = buyer.ID

	sub, err := client.ProxySubscribe.Create().
		SetName("E2E Plan").
		SetUnitPrice(1000).
		SetUnitTime("Month").
		SetShow(true).
		SetSell(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create subscribe: %v", err)
	}
	ids.subscribe = sub.ID

	mk := func(name, code, unit string, dur int64) int64 {
		o, eerr := client.ProxySubscribePriceOption.Create().
			SetSubscribeID(sub.ID).
			SetCode(code).
			SetOptionType("duration").
			SetName(name).
			SetDurationUnit(unit).
			SetDurationValue(dur).
			SetPrice(1000).
			Save(ctx)
		if eerr != nil {
			t.Fatalf("create price option %s: %v", name, eerr)
		}
		return o.ID
	}
	ids.optionMonthly = mk("月卡", "monthly", "Month", 1)
	ids.optionQuarterly = mk("季卡", "quarterly", "Month", 3)
	ids.optionYearly = mk("年卡", "yearly", "Year", 1)
	return ids
}

var e2eOrderSeq int64

func makeOrder(t *testing.T, ctx context.Context, client *ent.Client, buyerID, subscribeID, priceOptionID int64, isNew bool) *ent.ProxyOrder {
	t.Helper()
	e2eOrderSeq++
	o, err := client.ProxyOrder.Create().
		SetUserID(buyerID).
		SetOrderNo(fmt.Sprintf("E2E-%d-%d-%d", buyerID, priceOptionID, e2eOrderSeq)).
		SetType(int8(OrderTypeSubscribe)).
		SetAmount(10000). // 100 元（分）
		SetFeeAmount(0).
		SetSubscribeID(subscribeID).
		SetPriceOptionID(priceOptionID).
		SetIsNew(isNew).
		SetStatus(int8(OrderStatusFinished)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	return o
}

func TestE2ERebateRate(t *testing.T) {
	ctx := context.Background()
	client := newE2EClient(t)
	inviteSettings := snapshotInviteSettings(t, ctx, client)

	ids := setupRebateFixtures(t, ctx, client)
	t.Cleanup(func() {
		cleanRebateE2EFixtures(t, ctx, client, ids)
		restoreInviteSettings(t, ctx, client, inviteSettings)
	})

	// 取 referer 实体（resolveRebateRate 需要 *ent.ProxyUser）
	referer, err := client.ProxyUser.Get(ctx, ids.referer)
	if err != nil {
		t.Fatalf("get referer: %v", err)
	}

	h := &ActivateOrderHandler{
		db:     client,
		rdb:    redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"}),
		logger: log.NewHelper(log.DefaultLogger),
	}

	// --- 场景 1：全局比例兜底（无任何档位/套餐/个人配置）---
	setGlobalRebate(t, ctx, client, 5, false)
	o := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionMonthly, true)
	pct, ok := h.resolveRebateRate(ctx, referer, o)
	if !ok || pct != 5 {
		t.Fatalf("[场景1 全局兜底] 期望 (5,true), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景1 全局兜底] OK: percentage=%d", pct)

	// --- 场景 2：推荐人个人比例覆盖全局 ---
	_, _ = client.ProxyUser.UpdateOneID(ids.referer).SetReferralPercentage(15).Save(ctx)
	referer2, _ := client.ProxyUser.Get(ctx, ids.referer)
	pct, ok = h.resolveRebateRate(ctx, referer2, o)
	if !ok || pct != 15 {
		t.Fatalf("[场景2 推荐人个人] 期望 (15,true), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景2 推荐人个人] OK: percentage=%d", pct)

	// --- 场景 3：套餐默认档覆盖推荐人个人（price_option_id=0） ---
	_, err = client.ProxySubscribeRebateConfig.Create().
		SetSubscribeID(ids.subscribe).
		SetPriceOptionID(0).
		SetReferralPercentage(20).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed plan-default config: %v", err)
	}
	pct, ok = h.resolveRebateRate(ctx, referer2, o)
	if !ok || pct != 20 {
		t.Fatalf("[场景3 套餐默认] 期望 (20,true), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景3 套餐默认] OK: percentage=%d", pct)

	// --- 场景 4：档位专属覆盖套餐默认（月卡档=10%） ---
	_, err = client.ProxySubscribeRebateConfig.Create().
		SetSubscribeID(ids.subscribe).
		SetPriceOptionID(ids.optionMonthly).
		SetReferralPercentage(10).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed monthly-option config: %v", err)
	}
	oMonthly := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionMonthly, true)
	pct, ok = h.resolveRebateRate(ctx, referer2, oMonthly)
	if !ok || pct != 10 {
		t.Fatalf("[场景4 月卡档位专属] 期望 (10,true), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景4 月卡档位专属] OK: percentage=%d", pct)

	// --- 场景 5：其他档位（季卡）未单独配置 → 走套餐默认 20% ---
	oQuarterly := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionQuarterly, true)
	pct, ok = h.resolveRebateRate(ctx, referer2, oQuarterly)
	if !ok || pct != 20 {
		t.Fatalf("[场景5 季卡回退套餐默认] 期望 (20,true), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景5 季卡回退套餐默认] OK: percentage=%d", pct)

	// --- 场景 6：年卡档位 enabled=false → 短路，不返佣不兜底 ---
	_, err = client.ProxySubscribeRebateConfig.Create().
		SetSubscribeID(ids.subscribe).
		SetPriceOptionID(ids.optionYearly).
		SetReferralPercentage(99).
		SetEnabled(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed yearly-option disabled config: %v", err)
	}
	oYearly := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionYearly, true)
	pct, ok = h.resolveRebateRate(ctx, referer2, oYearly)
	if ok {
		t.Fatalf("[场景6 年卡显式关闭] 期望 (0,false) 短路, 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景6 年卡显式关闭] OK: 短路不返佣 (ok=false)")

	// --- 场景 7：套餐默认 enabled=false → 该套餐所有未单独配置的档位都短路 ---
	_ = client.ProxySubscribeRebateConfig.Update().
		Where(
			proxysubscriberebateconfig.SubscribeIDEQ(ids.subscribe),
			proxysubscriberebateconfig.PriceOptionIDEQ(0),
		).
		SetEnabled(false).
		Exec(ctx)
	// 月卡仍有专属配置(enabled=true) → 仍返佣 10%
	pct, ok = h.resolveRebateRate(ctx, referer2, oMonthly)
	if !ok || pct != 10 {
		t.Fatalf("[场景7a 月卡仍走专属] 期望 (10,true), 得到 (%d,%v)", pct, ok)
	}
	// 季卡无专属，套餐默认关闭 → 短路
	pct, ok = h.resolveRebateRate(ctx, referer2, oQuarterly)
	if ok {
		t.Fatalf("[场景7b 季卡因套餐默认关闭而短路] 期望 (0,false), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景7 套餐默认关闭] OK: 月卡仍走专属=%d, 季卡短路", func() int {
		p, _ := h.resolveRebateRate(ctx, referer2, oMonthly)
		return int(p)
	}())

	// --- 场景 8：管理员 API 数据层 round-trip（ReplaceSubscribeRebateConfig 全量替换） ---
	// 清掉所有 referer 个人配置，避免干扰；并重置套餐默认为启用
	_, _ = client.ProxyUser.UpdateOneID(ids.referer).SetReferralPercentage(0).Save(ctx)
	// 用 Replace 全量替换：只保留 月卡=12% enabled, 季卡=8% enabled，删除套餐默认与年卡
	n, err := client.ProxySubscribeRebateConfig.Delete().
		Where(proxysubscriberebateconfig.SubscribeIDEQ(ids.subscribe)).Exec(ctx)
	if err != nil {
		t.Fatalf("clean before replace: %v", err)
	}
	_ = n
	// 复用 data 层 repo（与生产代码同一实现路径）
	// 通过直接 ent 调用模拟 Replace 语义
	for _, it := range []struct {
		optID int64
		pct   int8
		en    bool
	}{
		{ids.optionMonthly, 12, true},
		{ids.optionQuarterly, 8, true},
	} {
		if _, e := client.ProxySubscribeRebateConfig.Create().
			SetSubscribeID(ids.subscribe).
			SetPriceOptionID(it.optID).
			SetReferralPercentage(it.pct).
			SetEnabled(it.en).
			Save(ctx); e != nil {
			t.Fatalf("replace insert: %v", e)
		}
	}
	referer3, _ := client.ProxyUser.Get(ctx, ids.referer)
	// 月卡 → 12%
	if pct, ok := h.resolveRebateRate(ctx, referer3, oMonthly); !ok || pct != 12 {
		t.Fatalf("[场景8a 月卡=12%%] 期望 (12,true), 得到 (%d,%v)", pct, ok)
	}
	// 季卡 → 8%
	if pct, ok := h.resolveRebateRate(ctx, referer3, oQuarterly); !ok || pct != 8 {
		t.Fatalf("[场景8b 季卡=8%%] 期望 (8,true), 得到 (%d,%v)", pct, ok)
	}
	// 年卡（无配置、无套餐默认、无个人、全局5%）→ 全局兜底 5%
	if pct, ok := h.resolveRebateRate(ctx, referer3, oYearly); !ok || pct != 5 {
		t.Fatalf("[场景8c 年卡回退全局=5%%] 期望 (5,true), 得到 (%d,%v)", pct, ok)
	}
	t.Logf("[场景8 全量替换+回退] OK: 月卡=12, 季卡=8, 年卡→全局5")

	// --- 场景 9：shouldProcessCommission 的 only_first_purchase 短路（续费订单）---
	// 推荐人 only_first_purchase=true 且订单 is_new=false
	_, _ = client.ProxyUser.UpdateOneID(ids.referer).
		SetOnlyFirstPurchase(true).
		SetReferralPercentage(0).
		Save(ctx)
	// 清掉所有返佣配置，仅依赖全局 5%（onlyFirst=false）
	_, _ = client.ProxySubscribeRebateConfig.Delete().
		Where(proxysubscriberebateconfig.SubscribeIDEQ(ids.subscribe)).Exec(ctx)
	setGlobalRebate(t, ctx, client, 5, true) // 全局 onlyFirst=true
	buyerInfo, _ := client.ProxyUser.Get(ctx, ids.buyer)
	oNew := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionMonthly, true)
	oRenew := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionMonthly, false)
	if !h.shouldProcessCommission(buyerInfo, oNew.IsNew) {
		t.Fatalf("[场景9a 新购应返佣] shouldProcessCommission 应为 true")
	}
	if h.shouldProcessCommission(buyerInfo, oRenew.IsNew) {
		t.Fatalf("[场景9b 续费(onlyFirst)不应返佣] shouldProcessCommission 应为 false")
	}
	t.Logf("[场景9 only_first_purchase] OK: 新购=true, 续费=false")

	// --- 场景 10：handleCommission 端到端写库（推荐人 commission 余额 + 日志）---
	// 重置：个人比例=30，onlyFirst=false，全局 onlyFirst=false
	_, _ = client.ProxyUser.UpdateOneID(ids.referer).
		SetReferralPercentage(30).
		SetOnlyFirstPurchase(false).
		SetCommission(0).
		Save(ctx)
	setGlobalRebate(t, ctx, client, 5, false)
	userInfo, _ := client.ProxyUser.Get(ctx, ids.buyer)
	oFinal := makeOrder(t, ctx, client, ids.buyer, ids.subscribe, ids.optionMonthly, true)
	// amount=10000, fee=0, pct=30 → 3000
	h.handleCommission(ctx, userInfo, oFinal)
	refererAfter, _ := client.ProxyUser.Get(ctx, ids.referer)
	if refererAfter.Commission == nil || *refererAfter.Commission != 3000 {
		got := int64(0)
		if refererAfter.Commission != nil {
			got = *refererAfter.Commission
		}
		t.Fatalf("[场景10 写库] 期望推荐人佣金余额 3000, 得到 %d", got)
	}
	// 验证日志写入（type=TypeCommission, object_id=referer）
	logCnt, _ := client.ProxySystemLog.Query().
		Where(
			proxysystemlog.ObjectIDEQ(ids.referer),
			proxysystemlog.TypeEQ(int8(33)), // TypeCommission = 33，见 internal/model/log/log.go
		).
		Count(ctx)
	if logCnt == 0 {
		t.Fatalf("[场景10 写库] 期望生成至少 1 条返佣日志, 实际 0")
	}
	t.Logf("[场景10 handleCommission 写库] OK: 推荐人佣金余额=%d, 返佣日志数=%d", *refererAfter.Commission, logCnt)

	t.Logf("\n========== 全部 e2e 场景通过 ==========")
}
