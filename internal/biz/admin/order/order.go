package order

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
)

// OrderRepo is the interface for order repository
type OrderRepo interface {
	CreateOrder(ctx context.Context, userID int, orderType uint32, quantity, price, amount, discount int,
		coupon string, couponDiscount, commission, feeAmount, paymentID int, tradeNo string,
		status uint32, subscribeID int) error
	UpdateOrderStatus(ctx context.Context, id int, status uint32, paymentID int, tradeNo string) error
	GetOrderList(ctx context.Context, page, size, userID int, status uint32, subscribeID int, search string) ([]*ent.ProxyOrder, int32, error)
	// DeleteOrders 删除订单（不影响用户套餐与流量）：待支付订单先自动关单回滚（库存/赠金），
	// user_subscribe.order_id 置零后物理删除订单，返回实际删除数量
	DeleteOrders(ctx context.Context, ids []int64) (int, error)
}

// OrderUseCase is the use case for order operations
type OrderUseCase struct {
	repo OrderRepo
	log  *log.Helper
}

// NewOrderUseCase creates a new order use case
func NewOrderUseCase(repo OrderRepo, logger log.Logger) *OrderUseCase {
	return &OrderUseCase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/admin/order")),
	}
}

// CreateOrder creates a new order
func (uc *OrderUseCase) CreateOrder(ctx context.Context, userID int, orderType uint32, quantity, price, amount, discount int,
	coupon string, couponDiscount, commission, feeAmount, paymentID int, tradeNo string,
	status uint32, subscribeID int) error {

	return uc.repo.CreateOrder(ctx, userID, orderType, quantity, price, amount, discount,
		coupon, couponDiscount, commission, feeAmount, paymentID, tradeNo, status, subscribeID)
}

// UpdateOrderStatus updates order status
func (uc *OrderUseCase) UpdateOrderStatus(ctx context.Context, id int, status uint32, paymentID int, tradeNo string) error {
	return uc.repo.UpdateOrderStatus(ctx, id, status, paymentID, tradeNo)
}

// GetOrderList gets order list
func (uc *OrderUseCase) GetOrderList(ctx context.Context, page, size, userID int, status uint32, subscribeID int, search string) ([]*ent.ProxyOrder, int32, error) {
	orders, total, err := uc.repo.GetOrderList(ctx, page, size, userID, status, subscribeID, search)
	return orders, total, err
}

// DeleteOrders 删除订单（issue #29）：不影响用户套餐与流量。
// 已支付/已完成订单的副作用（订阅、余额、佣金）已落库，全部保留；
// 待支付订单先自动执行关单回滚（恢复套餐库存、退回赠金）再删除。
func (uc *OrderUseCase) DeleteOrders(ctx context.Context, ids []int64) (int, error) {
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		return 0, nil
	}
	return uc.repo.DeleteOrders(ctx, normalized)
}
