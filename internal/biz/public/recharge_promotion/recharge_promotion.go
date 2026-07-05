package recharge_promotion

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// ActiveTier 用户端档位（脱敏，仅展示字段）
type ActiveTier struct {
	Amount int64  // 充值档位金额 分
	Bonus  int64  // 赠送金额 分
	Label  string // 档位标签
	Sort   int32  // 档位排序 ASC
}

// ActivePromotion 用户端活动（脱敏，仅展示字段）
type ActivePromotion struct {
	ID                 int64
	Name               string
	Description        string
	EndTime            int64 // Unix 毫秒，0=长期（用户端只关心截止时间）
	StackMode          int8  // 叠加策略
	CustomBonusEnabled bool
	CustomBonusPercent int32 // 万分比
	Tiers              []*ActiveTier
}

// RechargePromotionPublicRepo 用户端充值促销仓库接口
type RechargePromotionPublicRepo interface {
	// ListActive 列出当前生效的活动（status=true 且在时间窗口内），含启用的档位
	ListActive(ctx context.Context) ([]*ActivePromotion, error)
}

// RechargePromotionPublicUsecase 用户端充值促销用例
type RechargePromotionPublicUsecase struct {
	repo RechargePromotionPublicRepo
	log  *log.Helper
}

// NewRechargePromotionPublicUsecase 创建用户端充值促销用例
func NewRechargePromotionPublicUsecase(repo RechargePromotionPublicRepo, logger log.Logger) *RechargePromotionPublicUsecase {
	return &RechargePromotionPublicUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// ListActive 列出当前生效的活动
func (uc *RechargePromotionPublicUsecase) ListActive(ctx context.Context) ([]*ActivePromotion, error) {
	return uc.repo.ListActive(ctx)
}
