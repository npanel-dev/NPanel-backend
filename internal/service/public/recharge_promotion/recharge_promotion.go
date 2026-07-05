package recharge_promotion

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/npanel-dev/NPanel-backend/api/public/promotion/v1"
	publicpromotionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/recharge_promotion"
)

// RechargePromotionPublicService 充值促销用户端服务
type RechargePromotionPublicService struct {
	v1.UnimplementedRechargePromotionPublicServiceServer

	uc  *publicpromotionbiz.RechargePromotionPublicUsecase
	log *log.Helper
}

// NewRechargePromotionPublicService 创建用户端充值促销服务
func NewRechargePromotionPublicService(uc *publicpromotionbiz.RechargePromotionPublicUsecase, logger log.Logger) *RechargePromotionPublicService {
	return &RechargePromotionPublicService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/public/recharge_promotion")),
	}
}

// ListActiveRechargePromotion 列出当前生效的充值促销活动
func (s *RechargePromotionPublicService) ListActiveRechargePromotion(ctx context.Context, _ *v1.ListActiveRechargePromotionRequest) (*v1.ListActiveRechargePromotionReply, error) {
	list, err := s.uc.ListActive(ctx)
	if err != nil {
		s.log.Errorw("msg", "list active recharge promotion failed", "error", err)
		return nil, err
	}

	out := make([]*v1.ActiveRechargePromotion, 0, len(list))
	for _, p := range list {
		ap := &v1.ActiveRechargePromotion{
			Id:                 p.ID,
			Name:               p.Name,
			Description:        p.Description,
			EndTime:            p.EndTime,
			StackMode:          int32(p.StackMode),
			CustomBonusEnabled: p.CustomBonusEnabled,
			CustomBonusPercent: p.CustomBonusPercent,
		}
		if p.Tiers != nil {
			ap.Tiers = make([]*v1.ActiveRechargePromotionTier, 0, len(p.Tiers))
			for _, t := range p.Tiers {
				ap.Tiers = append(ap.Tiers, &v1.ActiveRechargePromotionTier{
					Amount: t.Amount,
					Bonus:  t.Bonus,
					Label:  t.Label,
					Sort:   t.Sort,
				})
			}
		}
		out = append(out, ap)
	}

	return &v1.ListActiveRechargePromotionReply{
		Total: int32(len(out)),
		List:  out,
	}, nil
}
