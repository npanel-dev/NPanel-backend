package order

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/order/v1"
	"github.com/npanel-dev/NPanel-backend/internal/biz/admin/order"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

const (
	maxOrderQuantity = 1000
	maxLargeAmount   = 2000000000
	maxOrderAmount   = 2147483647
)

func validateCreateOrderRequest(req *v1.CreateOrderRequest) error {
	if req.UserId <= 0 || req.Type == 0 || req.PaymentId <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.Quantity < 0 || req.Quantity > maxOrderQuantity {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.Price < 0 || req.Price > maxLargeAmount {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.Amount < 0 || req.Amount > maxOrderAmount {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.Discount < 0 || req.Discount > maxLargeAmount {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.CouponDiscount < 0 || req.CouponDiscount > maxLargeAmount {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.Commission < 0 || req.Commission > maxLargeAmount {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.FeeAmount < 0 || req.FeeAmount > maxLargeAmount {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	return nil
}

type OrderService struct {
	v1.UnimplementedOrderServiceServer

	uc  *order.OrderUseCase
	log *log.Helper
}

func NewOrderService(uc *order.OrderUseCase, logger log.Logger) *OrderService {
	return &OrderService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/admin/order")),
	}
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(ctx context.Context, req *v1.CreateOrderRequest) (*v1.CreateOrderReply, error) {
	if err := validateCreateOrderRequest(req); err != nil {
		return nil, err
	}

	userId := int(req.UserId)
	subscribeId := int(req.SubscribeId)
	paymentId := int(req.PaymentId)

	// 创建订单
	err := s.uc.CreateOrder(ctx,
		userId,
		req.Type,
		int(req.Quantity),
		int(req.Price),
		int(req.Amount),
		int(req.Discount),
		req.Coupon,
		int(req.CouponDiscount),
		int(req.Commission),
		int(req.FeeAmount),
		paymentId,
		req.TradeNo,
		req.Status,
		subscribeId,
	)
	if err != nil {
		s.log.Errorw("msg", "create order failed", "error", err)
		return nil, err
	}

	return &v1.CreateOrderReply{
		Code:    int32(responsecode.AdminCreateOrderSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminCreateOrderSuccess],
	}, nil
}

// UpdateOrderStatus 更新订单状态
func (s *OrderService) UpdateOrderStatus(ctx context.Context, req *v1.UpdateOrderStatusRequest) (*v1.UpdateOrderStatusReply, error) {
	// 验证订单ID
	if req.Id <= 0 {
		return nil, responsecode.ErrOrderIDRequired()
	}

	id := int(req.Id)
	paymentId := int(req.PaymentId)

	// 更新订单状态
	err := s.uc.UpdateOrderStatus(ctx, id, req.Status, paymentId, req.TradeNo)
	if err != nil {
		s.log.Errorw("msg", "update order status failed", "error", err)
		return nil, err
	}

	return &v1.UpdateOrderStatusReply{
		Code:    int32(responsecode.AdminUpdateOrderStatusSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminUpdateOrderStatusSuccess],
	}, nil
}

// DeleteOrder 删除订单（不影响用户套餐与流量）
func (s *OrderService) DeleteOrder(ctx context.Context, req *v1.DeleteOrderRequest) (*v1.DeleteOrderReply, error) {
	if len(req.Ids) == 0 {
		return nil, responsecode.ErrOrderIDRequired()
	}

	deleted, err := s.uc.DeleteOrders(ctx, req.Ids)
	if err != nil {
		s.log.Errorw("msg", "delete orders failed", "error", err)
		return nil, err
	}

	return &v1.DeleteOrderReply{
		Code:         int32(responsecode.AdminDeleteOrderSuccess),
		Message:      responsecode.CodeMessages[responsecode.AdminDeleteOrderSuccess],
		DeletedCount: int64(deleted),
	}, nil
}

// GetOrderList 获取订单列表
func (s *OrderService) GetOrderList(ctx context.Context, req *v1.GetOrderListRequest) (*v1.GetOrderListReply, error) {
	// 设置默认分页参数
	if req.Page == 0 {
		req.Page = 1
	}
	if req.Size == 0 {
		req.Size = 10
	}

	userId := int(req.UserId)
	subscribeId := int(req.SubscribeId)

	// 获取订单列表
	list, total, err := s.uc.GetOrderList(ctx, int(req.Page), int(req.Size), userId, req.Status, subscribeId, req.Search)
	if err != nil {
		s.log.Errorw("msg", "get order list failed", "error", err)
		return nil, responsecode.ErrOrderListFailed()
	}

	// 转换为响应格式
	var items []*v1.OrderItem
	for _, o := range list {
		item := &v1.OrderItem{
			Id:             int64(o.ID),
			ParentId:       int64(o.ParentID),
			UserId:         int64(o.UserID),
			OrderNo:        o.OrderNo,
			Type:           uint32(o.Type),
			Quantity:       int64(o.Quantity),
			Price:          int64(o.Price),
			Amount:         int64(o.Amount),
			GiftAmount:     int64(o.GiftAmount),
			Discount:       int64(o.Discount),
			Coupon:         o.Coupon,
			CouponDiscount: int64(o.CouponDiscount),
			Commission:     int64(o.Commission),
			PaymentId:      int64(o.PaymentID),
			Method:         o.Method,
			FeeAmount:      int64(o.FeeAmount),
			TradeNo:        o.TradeNo,
			Status:         uint32(o.Status),
			SubscribeId:    int64(o.SubscribeID),
			SubscribeToken: o.SubscribeToken,
			IsNew:          o.IsNew,
			CreatedAt:      o.CreatedAt.UnixMilli(),
			UpdatedAt:      o.UpdatedAt.UnixMilli(),
		}
		items = append(items, item)
	}

	return &v1.GetOrderListReply{
		Code:    int32(responsecode.AdminGetOrderListSuccess),
		Message: responsecode.CodeMessages[responsecode.AdminGetOrderListSuccess],
		Data: &v1.GetOrderListData{
			List:  items,
			Total: total,
		},
	}, nil
}
