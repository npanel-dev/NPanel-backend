package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyrechargepromotion"
	"github.com/npanel-dev/NPanel-backend/ent/proxyrechargepromotiontier"
	"github.com/redis/go-redis/v9"
)

// 本测试为 e2e 验证：连接本地 Docker MySQL 或 PostgreSQL，直接测试充值赠送的核心逻辑：
//   1. public data 层 ListActive 能否正确查出有效活动
//   2. handler.resolveRechargeBonus 的命中算法（档位匹配 / 自定义比例 / 互斥）
//
// 运行方式（需先启动后端完成建表）：
//
//	NPANEL_E2E_DATABASE_DRIVER=mysql \
//	NPANEL_E2E_DSN="npanel:npanel_password@tcp(127.0.0.1:13306)/npanel?parseTime=True&loc=Local" \
//	  go test ./internal/queue/handler/ -run TestE2ERechargePromotion -v -count=1
//
//	NPANEL_E2E_DATABASE_DRIVER=postgres \
//	NPANEL_E2E_DSN="postgres://npanel:npanel_password@127.0.0.1:15432/npanel?sslmode=disable" \
//	  go test ./internal/queue/handler/ -run TestE2ERechargePromotion -v -count=1

func newRechargePromoClient(t *testing.T) *ent.Client {
	return newE2EClient(t)
}

func cleanRechargePromoFixtures(t *testing.T, ctx context.Context, client *ent.Client) {
	t.Helper()
	promotions, err := client.ProxyRechargePromotion.Query().
		Where(proxyrechargepromotion.DescriptionEQ("e2e test")).All(ctx)
	if err != nil {
		t.Errorf("query recharge promotion fixtures: %v", err)
		return
	}
	ids := make([]int64, 0, len(promotions))
	for _, promotion := range promotions {
		ids = append(ids, promotion.ID)
	}
	if len(ids) == 0 {
		return
	}
	_, err = client.ProxyRechargePromotionTier.Delete().
		Where(proxyrechargepromotiontier.PromotionIDIn(ids...)).Exec(ctx)
	cleanupE2E(t, "recharge promotion tiers", err)
	_, err = client.ProxyRechargePromotion.Delete().
		Where(proxyrechargepromotion.IDIn(ids...)).Exec(ctx)
	cleanupE2E(t, "recharge promotions", err)
}

// seedPromotion 创建一个活动（含档位），返回活动 ID
func seedPromotion(t *testing.T, ctx context.Context, client *ent.Client,
	name string, priority int32, stackMode int8, customEnabled bool, customPercent int32,
	startMs, endMs int64, tiers []struct {
		Amount int64
		Bonus  int64
		Label  string
	}) int64 {
	t.Helper()
	p, err := client.ProxyRechargePromotion.Create().
		SetName(name).
		SetDescription("e2e test").
		SetStartTime(startMs).
		SetEndTime(endMs).
		SetStatus(true).
		SetPriority(priority).
		SetStackMode(stackMode).
		SetCustomBonusEnabled(customEnabled).
		SetCustomBonusPercent(customPercent).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed promotion %s: %v", name, err)
	}
	for _, tier := range tiers {
		if _, err := client.ProxyRechargePromotionTier.Create().
			SetPromotionID(p.ID).
			SetAmount(tier.Amount).
			SetBonus(tier.Bonus).
			SetLabel(tier.Label).
			SetEnable(true).
			Save(ctx); err != nil {
			t.Fatalf("seed tier: %v", err)
		}
	}
	return p.ID
}

// TestE2ERechargePromotion 端到端验证充值赠送逻辑
func TestE2ERechargePromotion(t *testing.T) {
	ctx := context.Background()
	client := newRechargePromoClient(t)

	// 清理旧的 e2e 数据
	cleanRechargePromoFixtures(t, ctx, client)
	t.Cleanup(func() { cleanRechargePromoFixtures(t, ctx, client) })

	nowMs := time.Now().UnixMilli()
	futureMs := nowMs + 86400000 // +1 天
	pastMs := nowMs - 86400000   // -1 天

	// 准备测试数据
	// 活动 A：日常充值，priority=10，互斥，开启自定义 5%，3 个档位
	seedPromotion(t, ctx, client, "日常充值赠送", 10, 0, true, 500, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{5000, 500, ""},
		{10000, 1200, "热门"},
		{50000, 10000, "推荐"},
	})

	// 活动 B：已过期（end_time 在过去），不应命中
	seedPromotion(t, ctx, client, "已过期活动", 20, 0, false, 0, pastMs-1000, pastMs, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 99999, ""},
	})

	// 活动 C：未开始（start_time 在未来），不应命中
	seedPromotion(t, ctx, client, "未开始活动", 30, 0, false, 0, futureMs, futureMs+1000, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 88888, ""},
	})

	// 活动 D：停用（status=false），不应命中
	pausedID := seedPromotion(t, ctx, client, "停用活动", 40, 0, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 77777, ""},
	})
	if err := client.ProxyRechargePromotion.UpdateOneID(pausedID).SetStatus(false).Exec(ctx); err != nil {
		t.Fatalf("pause promo: %v", err)
	}

	// 构造 handler（只需要 db，bonus 计算只读）
	h := &ActivateOrderHandler{
		db:     client,
		rdb:    redis.NewClient(&redis.Options{Addr: "127.0.0.1:16379"}),
		logger: log.NewHelper(log.DefaultLogger),
	}

	// ===== 测试 1：档位精确命中（充 100 元 = 10000 分，应送 1200） =====
	t.Run("档位精确命中_充100送12", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 10000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 1200 {
			t.Errorf("充100元 bonus = %d, 期望 1200", r.Bonus)
		}
		if len(r.Matched) != 1 || r.Matched[0].Name != "日常充值赠送" {
			t.Errorf("命中的活动 = %+v, 期望 日常充值赠送", r.Matched)
		}
		t.Logf("✅ 充100元 → bonus=%d（命中活动: %s）", r.Bonus, r.Matched[0].Name)
	})

	// ===== 测试 2：自定义金额按比例（充 80 元 = 8000 分，5% = 400） =====
	t.Run("自定义金额_充80按5%送4", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 8000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		expected := (8000 * 500) / 10000 // 400
		if r.Bonus != int64(expected) {
			t.Errorf("充80元 bonus = %d, 期望 %d", r.Bonus, expected)
		}
		t.Logf("✅ 充80元 → bonus=%d（5%% 比例）", r.Bonus)
	})

	// ===== 测试 3：档位优先于比例（充 50 元命中档位送 5，不走 5%） =====
	t.Run("档位优先_充50命中档位送5", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 5000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 500 {
			t.Errorf("充50元 bonus = %d, 期望 500（档位）", r.Bonus)
		}
		t.Logf("✅ 充50元 → bonus=%d（命中档位，不走比例）", r.Bonus)
	})

	// ===== 测试 4：过期的活动不命中 =====
	t.Run("过期活动不命中", func(t *testing.T) {
		// 充 10000，过期活动 B 有档位 10000 送 99999，但不应命中；命中活动 A 送 1200
		r, err := h.resolveRechargeBonus(ctx, 10000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 1200 {
			t.Errorf("充100元 bonus = %d, 期望 1200（过期活动不应命中送 99999）", r.Bonus)
		}
		t.Logf("✅ 过期活动被正确过滤，bonus=%d", r.Bonus)
	})

	// ===== 测试 5：DB 查询验证（ListActive 的 SQL 正确性） =====
	t.Run("DB查询_有效活动数", func(t *testing.T) {
		now := time.Now().UnixMilli()
		active, err := client.ProxyRechargePromotion.Query().
			Where(
				proxyrechargepromotion.StatusEQ(true),
				proxyrechargepromotion.StartTimeLTE(now),
				proxyrechargepromotion.Or(
					proxyrechargepromotion.EndTimeEQ(0),
					proxyrechargepromotion.EndTimeGTE(now),
				),
			).
			All(ctx)
		if err != nil {
			t.Fatalf("query active: %v", err)
		}
		// 应只有活动 A 命中（B 过期、C 未开始、D 停用）
		if len(active) != 1 {
			names := make([]string, 0, len(active))
			for _, p := range active {
				names = append(names, fmt.Sprintf("%s(id=%d,status=%v)", p.Name, p.ID, p.Status))
			}
			t.Errorf("有效活动数 = %d [%v], 期望 1（仅活动A）", len(active), names)
		} else {
			t.Logf("✅ 有效活动数 = 1（%s）", active[0].Name)
		}
	})

	// 清理
	cleanRechargePromoFixtures(t, ctx, client)
}

// TestE2ERechargePromotionStackMode 验证叠加/互斥模式下的多活动命中
func TestE2ERechargePromotionStackMode(t *testing.T) {
	ctx := context.Background()
	client := newRechargePromoClient(t)
	cleanRechargePromoFixtures(t, ctx, client)
	t.Cleanup(func() { cleanRechargePromoFixtures(t, ctx, client) })

	nowMs := time.Now().UnixMilli()

	// 活动 A：priority=10，互斥（stack_mode=0），档位 10000 送 1200
	seedPromotion(t, ctx, client, "日常互斥", 10, 0, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 1200, ""},
	})

	// 活动 B：priority=20，叠加（stack_mode=1），档位 10000 送 1200
	seedPromotion(t, ctx, client, "春节叠加", 20, 1, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 1200, ""},
	})

	h := &ActivateOrderHandler{
		db:     client,
		rdb:    redis.NewClient(&redis.Options{Addr: "127.0.0.1:16379"}),
		logger: log.NewHelper(log.DefaultLogger),
	}

	// 充 100 元：两个活动都有 10000 档位
	// priority DESC → 春节(20) 在前，stack_mode=1 叠加 → 继续累加日常(10) → 总 2400
	t.Run("叠加模式_两活动都命中求和2400", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 10000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 2400 {
			t.Errorf("bonus = %d, 期望 2400（1200+1200 叠加）", r.Bonus)
		}
		if len(r.Matched) != 2 {
			t.Errorf("命中活动数 = %d, 期望 2", len(r.Matched))
		}
		t.Logf("✅ 叠加模式 bonus=%d，命中 %d 个活动", r.Bonus, len(r.Matched))
	})

	cleanRechargePromoFixtures(t, ctx, client)

	// 互斥场景：两个活动都是 stack_mode=0，priority 不同
	seedPromotion(t, ctx, client, "低优先互斥", 10, 0, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 500, ""},
	})
	seedPromotion(t, ctx, client, "高优先互斥", 20, 0, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 9999, ""},
	})

	// 充 100 元：互斥模式只取 priority 最高的（高优先=20 送 9999）
	t.Run("互斥模式_只取最高优先级", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 10000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 9999 {
			t.Errorf("bonus = %d, 期望 9999（高优先级活动）", r.Bonus)
		}
		if len(r.Matched) != 1 {
			t.Errorf("命中活动数 = %d, 期望 1（互斥）", len(r.Matched))
		}
		t.Logf("✅ 互斥模式 bonus=%d（取高优先级）", r.Bonus)
	})

}

// TestE2ERechargePromotionCustomRatio 验证自定义金额按比例赠送的边界
func TestE2ERechargePromotionCustomRatio(t *testing.T) {
	ctx := context.Background()
	client := newRechargePromoClient(t)
	cleanRechargePromoFixtures(t, ctx, client)
	t.Cleanup(func() { cleanRechargePromoFixtures(t, ctx, client) })

	nowMs := time.Now().UnixMilli()
	// 活动：无档位，仅自定义金额按 5%（500‱）赠送
	seedPromotion(t, ctx, client, "纯比例活动", 10, 0, true, 500, nowMs, 0, nil)

	h := &ActivateOrderHandler{
		db:     client,
		rdb:    redis.NewClient(&redis.Options{Addr: "127.0.0.1:16379"}),
		logger: log.NewHelper(log.DefaultLogger),
	}

	// 充 33.33 元 = 3333 分，5% = 166.65 → 整数除法 166
	t.Run("比例计算_3333分按5%送166", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 3333)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		expected := (3333 * 500) / 10000 // 166
		if r.Bonus != int64(expected) {
			t.Errorf("bonus = %d, 期望 %d", r.Bonus, expected)
		}
		t.Logf("✅ 3333分按5%% → bonus=%d（整数除法）", r.Bonus)
	})

	// 充 0 或负数 → bonus=0
	t.Run("零金额_返回0", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 0)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 0 {
			t.Errorf("bonus = %d, 期望 0", r.Bonus)
		}
		t.Logf("✅ 零金额 bonus=0")
	})

	t.Run("负金额_返回0", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, -1000)
		if err != nil {
			t.Fatalf("resolveRechargeBounds: %v", err)
		}
		if r.Bonus != 0 {
			t.Errorf("bonus = %d, 期望 0（负金额）", r.Bonus)
		}
		t.Logf("✅ 负金额 bonus=0")
	})

}

// TestE2ERechargePromotionDisabledAndTime 验证停用/未开始活动不命中
func TestE2ERechargePromotionDisabledAndTime(t *testing.T) {
	ctx := context.Background()
	client := newRechargePromoClient(t)
	cleanRechargePromoFixtures(t, ctx, client)
	t.Cleanup(func() { cleanRechargePromoFixtures(t, ctx, client) })

	nowMs := time.Now().UnixMilli()
	futureMs := nowMs + 86400000 // +1天

	// 一个正常活动（控制组）
	normalID := seedPromotion(t, ctx, client, "正常活动", 10, 0, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 1000, ""},
	})

	// 一个停用活动（status=false）
	disabledID := seedPromotion(t, ctx, client, "停用活动", 20, 0, false, 0, nowMs, 0, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 88888, ""},
	})
	if err := client.ProxyRechargePromotion.UpdateOneID(disabledID).SetStatus(false).Exec(ctx); err != nil {
		t.Fatalf("disable: %v", err)
	}

	// 一个未开始活动（start_time 在未来）
	seedPromotion(t, ctx, client, "未开始活动", 30, 0, false, 0, futureMs, futureMs+1000, []struct {
		Amount int64
		Bonus  int64
		Label  string
	}{
		{10000, 77777, ""},
	})

	h := &ActivateOrderHandler{
		db:     client,
		rdb:    redis.NewClient(&redis.Options{Addr: "127.0.0.1:16379"}),
		logger: log.NewHelper(log.DefaultLogger),
	}

	// 充 100：应只命中正常活动（送 1000），停用和未开始都被过滤
	t.Run("停用和未开始都不命中_只命中正常活动", func(t *testing.T) {
		r, err := h.resolveRechargeBonus(ctx, 10000)
		if err != nil {
			t.Fatalf("resolveRechargeBonus: %v", err)
		}
		if r.Bonus != 1000 {
			t.Errorf("bonus = %d, 期望 1000（停用 88888 和未开始 77777 应被过滤）", r.Bonus)
		}
		if len(r.Matched) != 1 || r.Matched[0].Name != "正常活动" {
			t.Errorf("命中 = %+v, 期望只有 正常活动", r.Matched)
		}
		t.Logf("✅ 停用/未开始被过滤，bonus=%d", r.Bonus)
	})

	_ = normalID // 避免 unused 警告
}
