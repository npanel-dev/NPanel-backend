package data

import (
	"context"
	"encoding/json"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyorder"
	"github.com/npanel-dev/NPanel-backend/ent/proxypayment"
	"github.com/npanel-dev/NPanel-backend/internal/biz/admin/order"
	"github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
)

const orderModule = "data/admin_order"

type orderRepo struct {
	data  *Data
	log   *log.Helper
	queue *asynq.Client
}

// NewOrderRepo create order repository
func NewOrderRepo(data *Data, logger log.Logger) order.OrderRepo {
	return &orderRepo{
		data:  data,
		log:   log.NewHelper(log.With(logger, "module", orderModule)),
		queue: data.queue,
	}
}

// CreateOrder 创建订单
func (r *orderRepo) CreateOrder(ctx context.Context, userID int, orderType uint32, quantity, price, amount, discount int,
	coupon string, couponDiscount, commission, feeAmount, paymentID int, tradeNo string,
	status uint32, subscribeID int) error {

	// 如果paymentID > 0，验证支付方式是否存在并获取token作为method
	var method string
	if paymentID > 0 {
		payment, err := r.data.db.ProxyPayment.Query().
			Where(
				proxypayment.ID(int64(paymentID)),
			).
			Only(ctx)
		if err != nil {
			r.log.Errorw("msg", "payment method not found", "error", err, "paymentID", paymentID)
			if ent.IsNotFound(err) {
				return responsecode.NewPaymentNotFoundError()
			}
			return err
		}
		method = payment.Token
	}

	// 生成订单号
	orderNo := tool.GenerateTradeNo()

	// 创建订单
	_, err := r.data.db.ProxyOrder.Create().
		SetUserID(int64(userID)).
		SetOrderNo(orderNo).
		SetType(int8(orderType)).
		SetQuantity(int32(quantity)).
		SetPrice(int64(price)).
		SetAmount(int64(amount)).
		SetDiscount(int64(discount)).
		SetCoupon(coupon).
		SetCouponDiscount(int64(couponDiscount)).
		SetCommission(int64(commission)).
		SetFeeAmount(int64(feeAmount)).
		SetPaymentID(int64(paymentID)).
		SetMethod(method).
		SetTradeNo(tradeNo).
		SetStatus(int8(status)).
		SetSubscribeID(int64(subscribeID)).
		Save(ctx)

	return err
}

// UpdateOrderStatus 更新订单状态
func (r *orderRepo) UpdateOrderStatus(ctx context.Context, id int, status uint32, paymentID int, tradeNo string) error {
	// 查询订单
	orderInfo, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.ID(int64(id)),
		).
		Only(ctx)
	if err != nil {
		r.log.Errorw("msg", "order not found", "error", err, "orderID", id)
		return err
	}

	// 使用事务执行更新
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	updater := tx.ProxyOrder.UpdateOneID(int64(id))

	// 更新状态
	updater = updater.SetStatus(int8(status))

	// 如果paymentID > 0，更新支付方式和method
	if paymentID > 0 {
		payment, err := tx.ProxyPayment.Query().
			Where(
				proxypayment.ID(int64(paymentID)),
			).
			Only(ctx)
		if err != nil {
			r.log.Errorw("msg", "payment method not found", "error", err, "paymentID", paymentID)
			if ent.IsNotFound(err) {
				return rollback(tx, responsecode.NewPaymentNotFoundError())
			}
			return rollback(tx, err)
		}
		updater = updater.SetPaymentID(int64(paymentID)).SetMethod(payment.Platform)
	}

	// 如果提供了tradeNo，更新交易号
	if tradeNo != "" {
		updater = updater.SetTradeNo(tradeNo)
	}

	// 执行更新
	if err := updater.Exec(ctx); err != nil {
		r.log.Errorw("msg", "update order failed", "error", err, "orderID", id)
		return rollback(tx, err)
	}

	// 如果订单状态变为2(已付款)，需要入队ForthwithActivateOrder任务
	if status == 2 {
		r.log.Infow("msg", "order status changed to paid, enqueue activate task", "orderNo", orderInfo.OrderNo)

		// 构建任务负载
		payload := types.ForthwithActivateOrderPayload{
			OrderNo: orderInfo.OrderNo,
		}

		// 序列化负载
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			r.log.Errorw("msg", "failed to marshal activate order payload", "error", err, "orderNo", orderInfo.OrderNo)
			return rollback(tx, err)
		}

		// 创建任务并入队
		task := asynq.NewTask(types.ForthwithActivateOrder, payloadBytes)
		_, err = r.queue.Enqueue(task)
		if err != nil {
			r.log.Errorw("msg", "failed to enqueue activate order task", "error", err, "orderNo", orderInfo.OrderNo)
			return rollback(tx, err)
		}

		r.log.Infow("msg", "activate order task enqueued successfully", "orderNo", orderInfo.OrderNo)
	}

	return tx.Commit()
}

// GetOrderList 获取订单列表
func (r *orderRepo) GetOrderList(ctx context.Context, page, size, userID int, status uint32, subscribeID int, search string) ([]*ent.ProxyOrder, int32, error) {
	query := r.data.db.ProxyOrder.Query()

	// 用户ID筛选
	if userID != 0 {
		query = query.Where(proxyorder.UserID(int64(userID)))
	}

	// 订单状态筛选
	if status != 0 {
		query = query.Where(proxyorder.Status(int8(status)))
	}

	// 订阅ID筛选
	if subscribeID != 0 {
		query = query.Where(proxyorder.SubscribeID(int64(subscribeID)))
	}

	// 搜索关键字（订单号、交易号或优惠券）
	if search != "" {
		query = query.Where(
			proxyorder.Or(
				proxyorder.OrderNoContainsFold(search),
				proxyorder.TradeNoContainsFold(search),
				proxyorder.CouponContainsFold(search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	list, err := query.
		Limit(int(size)).
		Offset(int((page - 1) * size)).
		Order(ent.Desc(proxyorder.FieldID)).
		All(ctx)

	return list, int32(total), err
}
