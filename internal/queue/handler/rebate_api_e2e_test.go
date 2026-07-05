package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscriberebateconfig"
	"github.com/redis/go-redis/v9"
)

// TestE2ERebateRateViaAPI 验证端到端联通：
// 通过管理员 HTTP API 写入 rebate config → resolveRebateRate（返佣引擎）正确读取。
//
// 前置：后端已在 :8081 运行（NPANEL_E2E_API 可覆盖），且 npanel_e2e 库已建表。
// 复用 TestE2ERebateRate 留下的 subscribe=1（price option 1/2/3）。
func TestE2ERebateRateViaAPI(t *testing.T) {
	if e2eDSN() == "" {
		t.Skip("NPANEL_E2E_DSN not set; skipping api e2e test")
	}
	ctx := context.Background()
	client, err := ent.Open("mysql", e2eDSN())
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	defer client.Close()

	// 确认 subscribe=1 + price options 存在（由 TestE2ERebateRate fixtures 创建）
	sub, err := client.ProxySubscribe.Get(ctx, 1)
	if err != nil {
		t.Skipf("subscribe id=1 not found; run TestE2ERebateRate first: %v", err)
	}
	subscribeID := sub.ID

	// 临时 referer + buyer
	referer, err := client.ProxyUser.Create().SetPassword("x").SetReferCode("APIREF2").Save(ctx)
	if err != nil {
		t.Fatalf("create referer: %v", err)
	}
	buyer, err := client.ProxyUser.Create().SetPassword("x").SetRefererID(referer.ID).Save(ctx)
	if err != nil {
		t.Fatalf("create buyer: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.ProxyOrder.Delete().Exec(ctx)
		_ = client.ProxyUser.DeleteOneID(buyer.ID).Exec(ctx)
		_ = client.ProxyUser.DeleteOneID(referer.ID).Exec(ctx)
		_, _ = client.ProxySubscribeRebateConfig.Delete().
			Where(proxysubscriberebateconfig.SubscribeIDEQ(subscribeID)).Exec(ctx)
		// 清掉全局 invite 配置，避免污染其他测试
		_, _ = client.ProxySystem.Delete().Exec(ctx)
	})

	apiBase := os.Getenv("NPANEL_E2E_API")
	if apiBase == "" {
		apiBase = "http://localhost:8081"
	}

	// 1) 管理员登录拿 token
	loginBody, _ := json.Marshal(map[string]any{
		"email": "admin@npanel.dev", "password": "123456",
	})
	loginResp, err := http.Post(apiBase+"/v1/auth/admin/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("admin login: %v", err)
	}
	var loginOut struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginOut); err != nil {
		t.Fatalf("decode login resp: %v", err)
	}
	loginResp.Body.Close()
	if loginOut.Data.Token == "" {
		t.Fatalf("admin login: no token")
	}
	token := loginOut.Data.Token

	// 2) PUT rebate config：月卡=10%, 季卡=15%, 年卡关闭；不设套餐默认
	putBody, _ := json.Marshal(map[string]any{
		"subscribe_id": subscribeID,
		"items": []map[string]any{
			{"price_option_id": 1, "referral_percentage": 10, "enabled": true},
			{"price_option_id": 2, "referral_percentage": 15, "enabled": true},
			{"price_option_id": 3, "referral_percentage": 99, "enabled": false},
		},
	})
	req, _ := http.NewRequest("PUT", apiBase+"/v1/admin/subscribe/rebate_config", bytes.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	putResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT rebate_config: %v", err)
	}
	putOut, _ := io.ReadAll(putResp.Body)
	putResp.Body.Close()
	var putParsed map[string]any
	_ = json.Unmarshal(putOut, &putParsed)
	if code, _ := putParsed["code"].(float64); code != 200 {
		t.Fatalf("PUT rebate_config failed: %s", string(putOut))
	}

	// 3) 用 resolveRebateRate 验证（清掉全局 invite 配置 → 强制命中档位配置）
	_, _ = client.ProxySystem.Delete().Exec(ctx)
	refererEnt, _ := client.ProxyUser.Get(ctx, referer.ID)
	h := &ActivateOrderHandler{
		db:     client,
		rdb:    redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"}),
		logger: log.NewHelper(log.DefaultLogger),
	}

	// 月卡 (option 1) → 10%
	oMonthly := makeOrder(t, ctx, client, buyer.ID, subscribeID, 1, true)
	if pct, ok := h.resolveRebateRate(ctx, refererEnt, oMonthly); !ok || pct != 10 {
		t.Fatalf("[API 月卡] 期望 (10,true), 得到 (%d,%v)", pct, ok)
	}
	// 季卡 (option 2) → 15%
	oQuarterly := makeOrder(t, ctx, client, buyer.ID, subscribeID, 2, true)
	if pct, ok := h.resolveRebateRate(ctx, refererEnt, oQuarterly); !ok || pct != 15 {
		t.Fatalf("[API 季卡] 期望 (15,true), 得到 (%d,%v)", pct, ok)
	}
	// 年卡 (option 3) enabled=false → 短路
	oYearly := makeOrder(t, ctx, client, buyer.ID, subscribeID, 3, true)
	if pct, ok := h.resolveRebateRate(ctx, refererEnt, oYearly); ok {
		t.Fatalf("[API 年卡关闭] 期望 (0,false) 短路, 得到 (%d,%v)", pct, ok)
	}

	t.Logf("[API↔引擎 联通] OK: HTTP API 写入的配置被 resolveRebateRate 正确读取（月卡=10, 季卡=15, 年卡短路）")
}
