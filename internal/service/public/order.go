package public

import (
	"context"

	pb "github.com/npanel-dev/NPanel-backend/api/public/order/v1"
	publicBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public"
	"github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PublicOrderService struct {
	pb.UnimplementedPublicOrderServer

	uc *publicBiz.OrderUsecase
}

func NewPublicOrderService(uc *publicBiz.OrderUsecase) *PublicOrderService {
	return &PublicOrderService{uc: uc}
}

func (s *PublicOrderService) CloseOrder(ctx context.Context, req *pb.CloseOrderRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	if err := s.uc.CloseOrder(ctx, int(userID), req.OrderNo); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *PublicOrderService) QueryOrderDetail(ctx context.Context, req *pb.QueryOrderDetailRequest) (*pb.OrderDetail, error) {
	userID := middleware.GetUserID(ctx)
	order, err := s.uc.QueryOrderDetail(ctx, int(userID), req.OrderNo)
	if err != nil {
		return nil, err
	}
	return s.convertToProtoOrderDetail(order), nil
}

func (s *PublicOrderService) QueryOrderList(ctx context.Context, req *pb.QueryOrderListRequest) (*pb.QueryOrderListReply, error) {
	userID := middleware.GetUserID(ctx)
	orders, total, err := s.uc.QueryOrderList(ctx, int(userID), int(req.Page), int(req.Size), 0, 0)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.OrderDetail, 0, len(orders))
	for _, order := range orders {
		list = append(list, s.convertToProtoOrderDetail(order))
	}

	return &pb.QueryOrderListReply{
		Total: total,
		List:  list,
	}, nil
}

func (s *PublicOrderService) PreCreateOrder(ctx context.Context, req *pb.PreCreateOrderRequest) (*pb.PreCreateOrderReply, error) {
	userID := middleware.GetUserID(ctx)
	result, err := s.uc.PreCreateOrder(ctx, &publicBiz.PreCreateOrderParams{
		UserID:        userID,
		Type:          req.Type,
		SubscribeID:   req.SubscribeId,
		PriceOptionID: req.PriceOptionId,
		Quantity:      req.Quantity,
		Coupon:        req.Coupon,
		Payment:       req.Payment,
	})
	if err != nil {
		return nil, err
	}

	return &pb.PreCreateOrderReply{
		Price:          result.Price,
		Amount:         result.Amount,
		Discount:       result.Discount,
		GiftAmount:     result.GiftAmount,
		Coupon:         req.Coupon,
		CouponDiscount: result.CouponDiscount,
		FeeAmount:      result.FeeAmount,
	}, nil
}

func (s *PublicOrderService) Purchase(ctx context.Context, req *pb.PurchaseRequest) (*pb.PurchaseReply, error) {
	userID := middleware.GetUserID(ctx)
	result, err := s.uc.Purchase(ctx, &publicBiz.PurchaseParams{
		UserID:        userID,
		SubscribeID:   req.SubscribeId,
		PriceOptionID: req.PriceOptionId,
		Quantity:      req.Quantity,
		Coupon:        req.Coupon,
		Payment:       req.Payment,
	})
	if err != nil {
		return nil, err
	}
	return &pb.PurchaseReply{OrderNo: result.OrderNo}, nil
}

func (s *PublicOrderService) Recharge(ctx context.Context, req *pb.RechargeRequest) (*pb.RechargeReply, error) {
	userID := middleware.GetUserID(ctx)
	result, err := s.uc.Recharge(ctx, &publicBiz.RechargeParams{
		UserID:  userID,
		Amount:  req.Amount,
		Payment: req.Payment,
	})
	if err != nil {
		return nil, err
	}
	return &pb.RechargeReply{OrderNo: result.OrderNo}, nil
}

func (s *PublicOrderService) Renewal(ctx context.Context, req *pb.RenewalRequest) (*pb.RenewalReply, error) {
	userID := middleware.GetUserID(ctx)
	result, err := s.uc.Renewal(ctx, &publicBiz.RenewalParams{
		UserID:          userID,
		UserSubscribeID: req.UserSubscribeId,
		PriceOptionID:   req.PriceOptionId,
		Quantity:        req.Quantity,
		Coupon:          req.Coupon,
		Payment:         req.Payment,
	})
	if err != nil {
		return nil, err
	}
	return &pb.RenewalReply{OrderNo: result.OrderNo}, nil
}

func (s *PublicOrderService) ResetTraffic(ctx context.Context, req *pb.ResetTrafficRequest) (*pb.ResetTrafficReply, error) {
	userID := middleware.GetUserID(ctx)
	result, err := s.uc.ResetTraffic(ctx, &publicBiz.ResetTrafficParams{
		UserID:          userID,
		UserSubscribeID: req.UserSubscribeId,
		Payment:         req.Payment,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ResetTrafficReply{OrderNo: result.OrderNo}, nil
}

func (s *PublicOrderService) convertToProtoOrderDetail(order *publicBiz.OrderDetail) *pb.OrderDetail {
	if order == nil {
		return &pb.OrderDetail{}
	}

	var payment *pb.PaymentMethod
	if order.Payment != nil {
		payment = &pb.PaymentMethod{
			Id:          order.Payment.ID,
			Name:        order.Payment.Name,
			Platform:    order.Payment.Platform,
			Description: order.Payment.Description,
			Icon:        order.Payment.Icon,
			FeeMode:     uint32(order.Payment.FeeMode),
			FeePercent:  order.Payment.FeePercent,
			FeeAmount:   order.Payment.FeeAmount,
		}
	}

	var subscribe *pb.Subscribe
	if order.Subscribe != nil {
		discounts := make([]*pb.SubscribeDiscount, 0, len(order.Subscribe.Discount))
		for _, item := range order.Subscribe.Discount {
			discounts = append(discounts, &pb.SubscribeDiscount{
				Quantity: item.Quantity,
				Discount: item.Discount,
			})
		}

		subscribe = &pb.Subscribe{
			Id:                order.Subscribe.ID,
			Name:              order.Subscribe.Name,
			Language:          order.Subscribe.Language,
			Description:       order.Subscribe.Description,
			UnitPrice:         order.Subscribe.UnitPrice,
			UnitTime:          order.Subscribe.UnitTime,
			Discount:          discounts,
			Replacement:       order.Subscribe.Replacement,
			Inventory:         int32(order.Subscribe.Inventory),
			Traffic:           order.Subscribe.Traffic,
			SpeedLimit:        int32(order.Subscribe.SpeedLimit),
			DeviceLimit:       int32(order.Subscribe.DeviceLimit),
			Quota:             int32(order.Subscribe.Quota),
			CategoryId:        order.Subscribe.CategoryID,
			CategoryName:      order.Subscribe.CategoryName,
			Nodes:             convertIntSliceToInt64Slice(order.Subscribe.Nodes),
			NodeTags:          order.Subscribe.NodeTags,
			NodeGroupIds:      convertStringSliceToInt64Slice(order.Subscribe.NodeGroupIds),
			NodeGroupId:       parseStringInt64(order.Subscribe.NodeGroupId),
			TrafficLimit:      []*pb.TrafficLimit{},
			Show:              order.Subscribe.Show,
			Sell:              order.Subscribe.Sell,
			Sort:              int32(order.Subscribe.Sort),
			DeductionRatio:    int32(order.Subscribe.DeductionRatio),
			AllowDeduction:    order.Subscribe.AllowDeduction,
			ResetCycle:        int32(order.Subscribe.ResetCycle),
			RenewalReset:      order.Subscribe.RenewalReset,
			ShowOriginalPrice: order.Subscribe.ShowOriginalPrice,
			PriceOptions:      convertOrderPriceOptions(order.Subscribe.PriceOptions),
			CreatedAt:         order.Subscribe.CreatedAt,
			UpdatedAt:         order.Subscribe.UpdatedAt,
		}
	}

	return &pb.OrderDetail{
		Id:              order.ID,
		UserId:          order.UserID,
		OrderNo:         order.OrderNo,
		Type:            order.Type,
		Quantity:        order.Quantity,
		Price:           order.Price,
		Amount:          order.Amount,
		GiftAmount:      order.GiftAmount,
		Discount:        order.Discount,
		Coupon:          order.Coupon,
		CouponDiscount:  order.CouponDiscount,
		Commission:      0,
		Payment:         payment,
		Method:          order.Method,
		FeeAmount:       order.FeeAmount,
		TradeNo:         order.TradeNo,
		Status:          order.Status,
		SubscribeId:     order.SubscribeID,
		PriceOptionId:   order.PriceOptionID,
		PriceOptionName: order.PriceOptionName,
		DurationUnit:    order.DurationUnit,
		DurationValue:   order.DurationValue,
		OptionPrice:     order.OptionPrice,
		Subscribe:       subscribe,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}

func convertOrderPriceOptions(items []publicBiz.SubscribePriceOption) []*pb.SubscribePriceOption {
	if len(items) == 0 {
		return []*pb.SubscribePriceOption{}
	}
	result := make([]*pb.SubscribePriceOption, 0, len(items))
	for _, item := range items {
		result = append(result, &pb.SubscribePriceOption{
			Id:            item.ID,
			SubscribeId:   item.SubscribeID,
			Name:          item.Name,
			DurationUnit:  item.DurationUnit,
			DurationValue: item.DurationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     int32(item.Inventory),
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          int32(item.Sort),
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return result
}

func convertIntSliceToInt64Slice(input []int) []int64 {
	if len(input) == 0 {
		return []int64{}
	}
	result := make([]int64, 0, len(input))
	for _, item := range input {
		result = append(result, int64(item))
	}
	return result
}

func convertStringSliceToInt64Slice(input []string) []int64 {
	if len(input) == 0 {
		return []int64{}
	}
	result := make([]int64, 0, len(input))
	for _, item := range input {
		result = append(result, parseStringInt64(item))
	}
	return result
}

func parseStringInt64(value string) int64 {
	if value == "" {
		return 0
	}
	var result int64
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0
		}
		result = result*10 + int64(ch-'0')
	}
	return result
}
