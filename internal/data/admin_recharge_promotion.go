package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyrechargepromotion"
	"github.com/npanel-dev/NPanel-backend/ent/proxyrechargepromotiontier"
	promotionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/recharge_promotion"
)

type adminRechargePromotionRepo struct {
	data *Data
	log  *log.Helper
}

// NewAdminRechargePromotionRepo 创建充值促销管理仓库
// 返回 biz 接口类型（非 struct），Wire 靠此自动绑定，无需 wire.Bind
func NewAdminRechargePromotionRepo(data *Data, logger log.Logger) promotionbiz.RechargePromotionRepo {
	return &adminRechargePromotionRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ============================================================================
// ent ↔ biz 模型转换
// ============================================================================

// tierEntToBiz 把 ent 档位转成 biz 档位
func tierEntToBiz(t *ent.ProxyRechargePromotionTier) *promotionbiz.Tier {
	if t == nil {
		return nil
	}
	return &promotionbiz.Tier{
		Amount: t.Amount,
		Bonus:  t.Bonus,
		Label:  t.Label,
		Sort:   t.Sort,
		Enable: t.Enable,
	}
}

// promotionEntToBiz 把 ent 活动转成 biz 活动（含 tiers）
func promotionEntToBiz(p *ent.ProxyRechargePromotion) *promotionbiz.Promotion {
	if p == nil {
		return nil
	}
	out := &promotionbiz.Promotion{
		ID:                 p.ID,
		Name:               p.Name,
		Description:        p.Description,
		StartTime:          p.StartTime,
		EndTime:            p.EndTime,
		Status:             p.Status,
		Priority:           p.Priority,
		StackMode:          p.StackMode,
		CustomBonusEnabled: p.CustomBonusEnabled,
		CustomBonusPercent: p.CustomBonusPercent,
		Sort:               p.Sort,
	}
	if !p.CreatedAt.IsZero() {
		out.CreatedAt = p.CreatedAt.UnixMilli()
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = p.UpdatedAt.UnixMilli()
	}
	// tiers 通过边查询获取
	if p.Edges.Tiers != nil {
		out.Tiers = make([]*promotionbiz.Tier, 0, len(p.Edges.Tiers))
		for _, t := range p.Edges.Tiers {
			if t != nil {
				out.Tiers = append(out.Tiers, tierEntToBiz(t))
			}
		}
	}
	return out
}

// ============================================================================
// CRUD
// ============================================================================

// Create 创建活动（含档位）
func (r *adminRechargePromotionRepo) Create(ctx context.Context, p *promotionbiz.Promotion) (*promotionbiz.Promotion, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	created, err := tx.ProxyRechargePromotion.Create().
		SetName(p.Name).
		SetDescription(p.Description).
		SetStartTime(p.StartTime).
		SetEndTime(p.EndTime).
		SetStatus(p.Status).
		SetPriority(p.Priority).
		SetStackMode(p.StackMode).
		SetCustomBonusEnabled(p.CustomBonusEnabled).
		SetCustomBonusPercent(p.CustomBonusPercent).
		SetSort(p.Sort).
		Save(ctx)
	if err != nil {
		return nil, rollback(tx, err)
	}

	// 批量插入档位
	if err := r.createTiersInTx(ctx, tx, created.ID, p.Tiers); err != nil {
		return nil, rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, rollback(tx, err)
	}

	// 重新查询带 tiers 的完整活动返回
	return r.getByIDWithTiers(ctx, created.ID)
}

// Update 更新活动（tiers 全量替换：先删后插）
func (r *adminRechargePromotionRepo) Update(ctx context.Context, p *promotionbiz.Promotion) (*promotionbiz.Promotion, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	if err := tx.ProxyRechargePromotion.UpdateOneID(p.ID).
		SetName(p.Name).
		SetDescription(p.Description).
		SetStartTime(p.StartTime).
		SetEndTime(p.EndTime).
		SetPriority(p.Priority).
		SetStackMode(p.StackMode).
		SetCustomBonusEnabled(p.CustomBonusEnabled).
		SetCustomBonusPercent(p.CustomBonusPercent).
		SetSort(p.Sort).
		Exec(ctx); err != nil {
		return nil, rollback(tx, err)
	}

	// tiers 全量替换：先删后插
	if _, err := tx.ProxyRechargePromotionTier.Delete().
		Where(proxyrechargepromotiontier.PromotionIDEQ(p.ID)).
		Exec(ctx); err != nil {
		return nil, rollback(tx, err)
	}
	if err := r.createTiersInTx(ctx, tx, p.ID, p.Tiers); err != nil {
		return nil, rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, rollback(tx, err)
	}

	return r.getByIDWithTiers(ctx, p.ID)
}

// Delete 删除活动（连同档位）
func (r *adminRechargePromotionRepo) Delete(ctx context.Context, id int64) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	// 先删子表
	if _, err := tx.ProxyRechargePromotionTier.Delete().
		Where(proxyrechargepromotiontier.PromotionIDEQ(id)).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	// 再删主表
	if _, err := tx.ProxyRechargePromotion.Delete().
		Where(proxyrechargepromotion.IDEQ(id)).
		Exec(ctx); err != nil {
		return rollback(tx, err)
	}
	return tx.Commit()
}

// UpdateStatus 快速启停（不需要事务）
func (r *adminRechargePromotionRepo) UpdateStatus(ctx context.Context, id int64, status bool) error {
	return r.data.db.ProxyRechargePromotion.UpdateOneID(id).
		SetStatus(status).
		Exec(ctx)
}

// List 列表查询（含 tiers），按 priority DESC, id DESC 排序
func (r *adminRechargePromotionRepo) List(ctx context.Context, page, size int32, status *bool, search *string) ([]*promotionbiz.Promotion, int32, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	query := r.data.db.ProxyRechargePromotion.Query().WithTiers()
	if status != nil {
		query = query.Where(proxyrechargepromotion.StatusEQ(*status))
	}
	if search != nil && *search != "" {
		query = query.Where(proxyrechargepromotion.NameContainsFold(*search))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	list, err := query.
		Order(ent.Desc(proxyrechargepromotion.FieldPriority), ent.Desc(proxyrechargepromotion.FieldID)).
		Offset(int(offset)).
		Limit(int(size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*promotionbiz.Promotion, 0, len(list))
	for _, p := range list {
		out = append(out, promotionEntToBiz(p))
	}
	return out, int32(total), nil
}

// ============================================================================
// helpers
// ============================================================================

// createTiersInTx 在事务内批量创建档位
func (r *adminRechargePromotionRepo) createTiersInTx(ctx context.Context, tx *ent.Tx, promotionID int64, tiers []*promotionbiz.Tier) error {
	if len(tiers) == 0 {
		return nil
	}
	builders := make([]*ent.ProxyRechargePromotionTierCreate, 0, len(tiers))
	for _, t := range tiers {
		if t == nil || t.Amount <= 0 {
			continue
		}
		builders = append(builders, tx.ProxyRechargePromotionTier.Create().
			SetPromotionID(promotionID).
			SetAmount(t.Amount).
			SetBonus(t.Bonus).
			SetLabel(t.Label).
			SetSort(t.Sort).
			SetEnable(t.Enable))
	}
	if len(builders) > 0 {
		if _, err := tx.ProxyRechargePromotionTier.CreateBulk(builders...).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

// getByIDWithTiers 查询单个活动（含 tiers）
func (r *adminRechargePromotionRepo) getByIDWithTiers(ctx context.Context, id int64) (*promotionbiz.Promotion, error) {
	p, err := r.data.db.ProxyRechargePromotion.Query().
		Where(proxyrechargepromotion.IDEQ(id)).
		WithTiers().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return promotionEntToBiz(p), nil
}
