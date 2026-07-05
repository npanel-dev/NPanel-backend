package recharge_promotion

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/promotion/v1"
	promotionbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/recharge_promotion"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// RechargePromotionService 充值促销管理服务
type RechargePromotionService struct {
	v1.UnimplementedRechargePromotionAdminServiceServer

	uc  *promotionbiz.RechargePromotionUsecase
	log *log.Helper
}

// NewRechargePromotionService 创建充值促销管理服务
func NewRechargePromotionService(uc *promotionbiz.RechargePromotionUsecase, logger log.Logger) *RechargePromotionService {
	return &RechargePromotionService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/admin/recharge_promotion")),
	}
}

// ============================================================================
// proto ↔ biz 模型转换
// ============================================================================

// tierProtoToBiz 把 proto 档位转成 biz 档位
func tierProtoToBiz(t *v1.RechargePromotionTier) *promotionbiz.Tier {
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

// tierBizToProto 把 biz 档位转成 proto 档位
func tierBizToProto(t *promotionbiz.Tier) *v1.RechargePromotionTier {
	if t == nil {
		return nil
	}
	return &v1.RechargePromotionTier{
		Amount: t.Amount,
		Bonus:  t.Bonus,
		Label:  t.Label,
		Sort:   t.Sort,
		Enable: t.Enable,
	}
}

// promotionBizToProto 把 biz 活动转成 proto 活动
func promotionBizToProto(p *promotionbiz.Promotion) *v1.RechargePromotion {
	if p == nil {
		return nil
	}
	out := &v1.RechargePromotion{
		Id:                 p.ID,
		Name:               p.Name,
		Description:        p.Description,
		StartTime:          p.StartTime,
		EndTime:            p.EndTime,
		Status:             p.Status,
		Priority:           p.Priority,
		StackMode:          int32(p.StackMode),
		CustomBonusEnabled: p.CustomBonusEnabled,
		CustomBonusPercent: p.CustomBonusPercent,
		Sort:               p.Sort,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
	if p.Tiers != nil {
		out.Tiers = make([]*v1.RechargePromotionTier, 0, len(p.Tiers))
		for _, t := range p.Tiers {
			out.Tiers = append(out.Tiers, tierBizToProto(t))
		}
	}
	return out
}

// ============================================================================
// RPC 方法
// ============================================================================

// CreateRechargePromotion 创建充值促销活动
func (s *RechargePromotionService) CreateRechargePromotion(ctx context.Context, req *v1.CreateRechargePromotionRequest) (*v1.CreateRechargePromotionReply, error) {
	tiers := make([]*promotionbiz.Tier, 0, len(req.Tiers))
	for _, t := range req.Tiers {
		tiers = append(tiers, tierProtoToBiz(t))
	}
	p := &promotionbiz.Promotion{
		Name:               req.Name,
		Description:        req.Description,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		Priority:           req.Priority,
		StackMode:          int8(req.StackMode),
		CustomBonusEnabled: req.CustomBonusEnabled,
		CustomBonusPercent: req.CustomBonusPercent,
		Sort:               req.Sort,
		Tiers:              tiers,
	}

	created, err := s.uc.Create(ctx, p)
	if err != nil {
		s.log.Errorw("msg", "create recharge promotion failed", "error", err)
		return nil, responsecode.ErrInvalidParam()
	}

	return &v1.CreateRechargePromotionReply{
		Code:    int32(responsecode.AdminCreateRechargePromotionSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminCreateRechargePromotionSuccess],
		Data:    &v1.CreateRechargePromotionData{Promotion: promotionBizToProto(created)},
	}, nil
}

// UpdateRechargePromotion 更新充值促销活动（tiers 全量替换）
func (s *RechargePromotionService) UpdateRechargePromotion(ctx context.Context, req *v1.UpdateRechargePromotionRequest) (*v1.UpdateRechargePromotionReply, error) {
	tiers := make([]*promotionbiz.Tier, 0, len(req.Tiers))
	for _, t := range req.Tiers {
		tiers = append(tiers, tierProtoToBiz(t))
	}
	p := &promotionbiz.Promotion{
		ID:                 req.Id,
		Name:               req.Name,
		Description:        req.Description,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		Priority:           req.Priority,
		StackMode:          int8(req.StackMode),
		CustomBonusEnabled: req.CustomBonusEnabled,
		CustomBonusPercent: req.CustomBonusPercent,
		Sort:               req.Sort,
		Tiers:              tiers,
	}

	updated, err := s.uc.Update(ctx, p)
	if err != nil {
		s.log.Errorw("msg", "update recharge promotion failed", "error", err)
		return nil, responsecode.ErrInvalidParam()
	}

	return &v1.UpdateRechargePromotionReply{
		Code:    int32(responsecode.AdminUpdateRechargePromotionSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminUpdateRechargePromotionSuccess],
		Data:    &v1.UpdateRechargePromotionData{Promotion: promotionBizToProto(updated)},
	}, nil
}

// DeleteRechargePromotion 删除充值促销活动
func (s *RechargePromotionService) DeleteRechargePromotion(ctx context.Context, req *v1.DeleteRechargePromotionRequest) (*v1.DeleteRechargePromotionReply, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		s.log.Errorw("msg", "delete recharge promotion failed", "error", err)
		return nil, responsecode.ErrInvalidParam()
	}
	return &v1.DeleteRechargePromotionReply{
		Code:    int32(responsecode.AdminDeleteRechargePromotionSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminDeleteRechargePromotionSuccess],
	}, nil
}

// GetRechargePromotionList 获取充值促销活动列表
func (s *RechargePromotionService) GetRechargePromotionList(ctx context.Context, req *v1.GetRechargePromotionListRequest) (*v1.GetRechargePromotionListReply, error) {
	list, total, err := s.uc.List(ctx, req.Page, req.Size, req.Status, req.Search)
	if err != nil {
		s.log.Errorw("msg", "get recharge promotion list failed", "error", err)
		return nil, responsecode.ErrInvalidParam()
	}

	out := make([]*v1.RechargePromotion, 0, len(list))
	for _, p := range list {
		out = append(out, promotionBizToProto(p))
	}

	return &v1.GetRechargePromotionListReply{
		Code:    int32(responsecode.AdminGetRechargePromotionListSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminGetRechargePromotionListSuccess],
		Data: &v1.GetRechargePromotionListData{
			Total: total,
			List:  out,
		},
	}, nil
}

// UpdateRechargePromotionStatus 快速启停活动
func (s *RechargePromotionService) UpdateRechargePromotionStatus(ctx context.Context, req *v1.UpdateRechargePromotionStatusRequest) (*v1.UpdateRechargePromotionStatusReply, error) {
	if err := s.uc.UpdateStatus(ctx, req.Id, req.Status); err != nil {
		s.log.Errorw("msg", "update recharge promotion status failed", "error", err)
		return nil, responsecode.ErrInvalidParam()
	}
	return &v1.UpdateRechargePromotionStatusReply{
		Code:    int32(responsecode.AdminUpdateRechargePromotionStatusSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminUpdateRechargePromotionStatusSuccess],
	}, nil
}
