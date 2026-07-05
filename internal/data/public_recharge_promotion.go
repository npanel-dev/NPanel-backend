package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyrechargepromotion"
	"github.com/npanel-dev/NPanel-backend/ent/proxyrechargepromotiontier"
	publicpromotionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/recharge_promotion"
)

type publicRechargePromotionRepo struct {
	data *Data
	log  *log.Helper
}

// NewPublicRechargePromotionRepo 创建用户端充值促销仓库
func NewPublicRechargePromotionRepo(data *Data, logger log.Logger) publicpromotionbiz.RechargePromotionPublicRepo {
	return &publicRechargePromotionRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ListActive 列出当前生效的活动（status=true 且在时间窗口内），含启用的档位
func (r *publicRechargePromotionRepo) ListActive(ctx context.Context) ([]*publicpromotionbiz.ActivePromotion, error) {
	now := time.Now().UnixMilli()

	// 查询有效活动：status=true && (start_time <= now) && (end_time=0 或 end_time >= now)
	// 按 priority DESC 排序
	promotions, err := r.data.db.ProxyRechargePromotion.Query().
		Where(
			proxyrechargepromotion.StatusEQ(true),
			proxyrechargepromotion.StartTimeLTE(now),
			proxyrechargepromotion.Or(
				proxyrechargepromotion.EndTimeEQ(0),
				proxyrechargepromotion.EndTimeGTE(now),
			),
		).
		WithTiers(func(q *ent.ProxyRechargePromotionTierQuery) {
			q.Where(proxyrechargepromotiontier.EnableEQ(true)).
				Order(ent.Asc(proxyrechargepromotiontier.FieldSort))
		}).
		Order(ent.Desc(proxyrechargepromotion.FieldPriority), ent.Asc(proxyrechargepromotion.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*publicpromotionbiz.ActivePromotion, 0, len(promotions))
	for _, p := range promotions {
		if p == nil {
			continue
		}
		ap := &publicpromotionbiz.ActivePromotion{
			ID:                 p.ID,
			Name:               p.Name,
			Description:        p.Description,
			EndTime:            p.EndTime,
			StackMode:          p.StackMode,
			CustomBonusEnabled: p.CustomBonusEnabled,
			CustomBonusPercent: p.CustomBonusPercent,
		}
		if p.Edges.Tiers != nil {
			ap.Tiers = make([]*publicpromotionbiz.ActiveTier, 0, len(p.Edges.Tiers))
			for _, t := range p.Edges.Tiers {
				if t == nil {
					continue
				}
				ap.Tiers = append(ap.Tiers, &publicpromotionbiz.ActiveTier{
					Amount: t.Amount,
					Bonus:  t.Bonus,
					Label:  t.Label,
					Sort:   t.Sort,
				})
			}
		}
		out = append(out, ap)
	}
	return out, nil
}
