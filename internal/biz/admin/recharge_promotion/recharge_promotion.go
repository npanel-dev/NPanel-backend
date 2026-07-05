package recharge_promotion

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// ============================================================================
// biz 层模型（与 proto 解耦，service 层做 proto ↔ biz 转换）
// ============================================================================

// Tier 充值促销档位
type Tier struct {
	Amount int64  // 充值档位金额 分
	Bonus  int64  // 赠送金额 分
	Label  string // 档位标签
	Sort   int32  // 档位排序 ASC
	Enable bool   // 单档启停
}

// Promotion 充值促销活动
type Promotion struct {
	ID                 int64
	Name               string
	Description        string
	StartTime          int64  // Unix 毫秒，0=不限
	EndTime            int64  // Unix 毫秒，0=长期
	Status             bool
	Priority           int32
	StackMode          int8 // 0=互斥 1=叠加
	CustomBonusEnabled bool
	CustomBonusPercent int32 // 万分比 0-10000
	Sort               int32
	Tiers              []*Tier
	CreatedAt          int64 // Unix 毫秒
	UpdatedAt          int64 // Unix 毫秒
}

// ============================================================================
// Repo 接口（data 层实现）
// ============================================================================

// RechargePromotionRepo 充值促销仓库接口
type RechargePromotionRepo interface {
	Create(ctx context.Context, p *Promotion) (*Promotion, error)
	Update(ctx context.Context, p *Promotion) (*Promotion, error)
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status bool) error
	List(ctx context.Context, page, size int32, status *bool, search *string) ([]*Promotion, int32, error)
}

// ============================================================================
// Usecase
// ============================================================================

// RechargePromotionUsecase 充值促销管理用例
type RechargePromotionUsecase struct {
	repo RechargePromotionRepo
	log  *log.Helper
}

// NewRechargePromotionUsecase 创建充值促销管理用例
func NewRechargePromotionUsecase(repo RechargePromotionRepo, logger log.Logger) *RechargePromotionUsecase {
	return &RechargePromotionUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// ============================================================================
// 校验
// ============================================================================

// validate 校验活动字段（创建/更新共用）
func (p *Promotion) validate() error {
	if p.Name == "" {
		return errors.New("活动名称不能为空")
	}
	// end_time=0 表示长期，否则 start_time 必须 < end_time
	if p.EndTime != 0 && p.StartTime >= p.EndTime {
		return errors.New("结束时间必须大于开始时间")
	}
	if p.StackMode != 0 && p.StackMode != 1 {
		return fmt.Errorf("无效的叠加策略: %d（应为 0=互斥 或 1=叠加）", p.StackMode)
	}
	if p.CustomBonusPercent < 0 || p.CustomBonusPercent > 10000 {
		return fmt.Errorf("自定义金额赠送比例 %d 超出范围（0-10000）", p.CustomBonusPercent)
	}
	// 校验档位
	seen := make(map[int64]struct{}, len(p.Tiers))
	for i, t := range p.Tiers {
		if t == nil {
			return fmt.Errorf("第 %d 个档位为空", i+1)
		}
		if t.Amount <= 0 {
			return fmt.Errorf("第 %d 个档位金额必须大于 0", i+1)
		}
		if t.Bonus < 0 {
			return fmt.Errorf("第 %d 个档位赠送金额不能为负", i+1)
		}
		if _, exists := seen[t.Amount]; exists {
			return fmt.Errorf("档位金额 %d 重复", t.Amount)
		}
		seen[t.Amount] = struct{}{}
	}
	return nil
}

// ============================================================================
// Usecase 方法
// ============================================================================

// Create 创建活动
func (uc *RechargePromotionUsecase) Create(ctx context.Context, p *Promotion) (*Promotion, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	return uc.repo.Create(ctx, p)
}

// Update 更新活动（tiers 全量替换）
func (uc *RechargePromotionUsecase) Update(ctx context.Context, p *Promotion) (*Promotion, error) {
	if p.ID == 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, p)
}

// Delete 删除活动（连同档位）
func (uc *RechargePromotionUsecase) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("活动ID不能为空")
	}
	return uc.repo.Delete(ctx, id)
}

// UpdateStatus 快速启停
func (uc *RechargePromotionUsecase) UpdateStatus(ctx context.Context, id int64, status bool) error {
	if id == 0 {
		return errors.New("活动ID不能为空")
	}
	return uc.repo.UpdateStatus(ctx, id, status)
}

// List 列表查询
func (uc *RechargePromotionUsecase) List(ctx context.Context, page, size int32, status *bool, search *string) ([]*Promotion, int32, error) {
	return uc.repo.List(ctx, page, size, status, search)
}
