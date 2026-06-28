package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxycoupon"
	"github.com/npanel-dev/NPanel-backend/ent/proxyorder"
	"github.com/npanel-dev/NPanel-backend/ent/proxypayment"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribecategory"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribepriceoption"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	publicBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	queueTypes "github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/exchangeRate"
	paymentPlatform "github.com/npanel-dev/NPanel-backend/pkg/payment"
	"github.com/npanel-dev/NPanel-backend/pkg/payment/alipay"
	"github.com/npanel-dev/NPanel-backend/pkg/payment/epay"
	"github.com/npanel-dev/NPanel-backend/pkg/payment/stripe"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

const (
	// 支付方式常量
	Epay            = "epay"
	AlipayF2f       = "alipay_f2f"
	StripeAlipay    = "stripe_alipay"
	StripeWeChatPay = "stripe_wechat_pay"
	Balance         = "balance"

	// CloseOrderTimeMinutes 订单自动关闭的时间（分钟）
	CloseOrderTimeMinutes = 15

	// MaxQuantity 最大购买数量
	MaxQuantity = 1000

	// MaxOrderAmount 最大订单金额（分）
	MaxOrderAmount = 2147483647

	// MaxRechargeAmount 最大充值金额（分）
	MaxRechargeAmount = 2000000000
)

// SubscribeDiscount 订阅折扣结构
type SubscribeDiscount struct {
	Quantity int64 `json:"quantity"`
	Discount int64 `json:"discount"`
}

type publicOrderRepo struct {
	data   *Data
	logger *log.Helper
	config *conf.Application
}

// NewPublicOrderRepo 创建公共订单仓储实例
func NewPublicOrderRepo(data *Data, config *conf.Application, logger log.Logger) publicBiz.OrderRepo {
	return &publicOrderRepo{
		data:   data,
		logger: log.NewHelper(logger),
		config: config,
	}
}

func (r *publicOrderRepo) isUserEligibleForNewOrder(ctx context.Context, userID int64) (bool, error) {
	count, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.UserIDEQ(userID),
			proxyorder.StatusIn(2, 5),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (r *publicOrderRepo) listLegacyValidUserSubscriptions(ctx context.Context, userID int64) ([]*ent.ProxyUserSubscribe, error) {
	items, err := r.data.db.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	result := make([]*ent.ProxyUserSubscribe, 0, len(items))
	for _, item := range items {
		if shouldKeepLegacyUserSubscribe(item, now) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *publicOrderRepo) getSellablePriceOption(ctx context.Context, subscribeID, optionID int64) (*ent.ProxySubscribePriceOption, error) {
	query := r.data.db.ProxySubscribePriceOption.Query().
		Where(
			proxysubscribepriceoption.SubscribeIDEQ(subscribeID),
			proxysubscribepriceoption.OptionTypeEQ("duration"),
			proxysubscribepriceoption.ShowEQ(true),
			proxysubscribepriceoption.SellEQ(true),
		)
	if optionID > 0 {
		query = query.Where(proxysubscribepriceoption.IDEQ(optionID))
	} else {
		query = query.Order(
			ent.Desc(proxysubscribepriceoption.FieldIsDefault),
			ent.Desc(proxysubscribepriceoption.FieldSort),
			ent.Asc(proxysubscribepriceoption.FieldID),
		)
	}
	option, err := query.First(ctx)
	if err != nil {
		r.logger.Errorf("[getSellablePriceOption] option not found: subscribeID=%d optionID=%d error=%v", subscribeID, optionID, err)
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if option.Inventory == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeOutOfStock)
	}
	return option, nil
}

func priceFromOption(option *ent.ProxySubscribePriceOption) (price, amount, discount int64) {
	amount = option.Price
	price = option.OriginalPrice
	if price <= 0 || price < amount {
		price = amount
	}
	return price, amount, price - amount
}

func orderDurationQuantity(option *ent.ProxySubscribePriceOption) int32 {
	if option.DurationUnit == "NoLimit" {
		return 0
	}
	if option.DurationValue > 1<<31-1 {
		return 1<<31 - 1
	}
	return int32(option.DurationValue)
}

func decrementSubscribeInventory(ctx context.Context, tx *ent.Tx, subscribeID int64) error {
	affected, err := tx.ProxySubscribe.Update().
		Where(
			proxysubscribe.IDEQ(subscribeID),
			proxysubscribe.SellEQ(true),
			proxysubscribe.InventoryGT(0),
		).
		AddInventory(-1).
		Save(ctx)
	if err != nil {
		return errors.InternalServer("SUBSCRIBE_UPDATE_FAILED", "更新订阅库存失败")
	}
	if affected > 0 {
		return nil
	}

	latestSub, err := tx.ProxySubscribe.Query().
		Where(proxysubscribe.IDEQ(subscribeID)).
		Only(ctx)
	if err != nil {
		return errors.InternalServer("SUBSCRIBE_QUERY_FAILED", "查询订阅失败")
	}
	if !latestSub.Sell {
		return errors.BadRequest("SUBSCRIBE_NOT_FOR_SALE", "此订阅计划不可购买")
	}
	if latestSub.Inventory == -1 {
		return nil
	}
	return responsecode.NewKratosError(responsecode.ErrSubscribeOutOfStock)
}

func decrementDurationPriceOptionInventory(ctx context.Context, tx *ent.Tx, subscribeID, optionID int64) error {
	affected, err := tx.ProxySubscribePriceOption.Update().
		Where(
			proxysubscribepriceoption.IDEQ(optionID),
			proxysubscribepriceoption.SubscribeIDEQ(subscribeID),
			proxysubscribepriceoption.OptionTypeEQ("duration"),
			proxysubscribepriceoption.SellEQ(true),
			proxysubscribepriceoption.InventoryGT(0),
		).
		AddInventory(-1).
		Save(ctx)
	if err != nil {
		return errors.InternalServer("PRICE_OPTION_UPDATE_FAILED", "更新价格档位库存失败")
	}
	if affected > 0 {
		return nil
	}

	latestOption, err := tx.ProxySubscribePriceOption.Query().
		Where(
			proxysubscribepriceoption.IDEQ(optionID),
			proxysubscribepriceoption.SubscribeIDEQ(subscribeID),
		).
		Only(ctx)
	if err != nil {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if latestOption.OptionType != "duration" || !latestOption.Sell {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if latestOption.Inventory == -1 {
		return nil
	}
	return responsecode.NewKratosError(responsecode.ErrSubscribeOutOfStock)
}

// CloseOrder 关闭订单，包含完整的业务逻辑（含赠金退回）
func (r *publicOrderRepo) CloseOrder(ctx context.Context, userID int, orderNo string) error {
	// 通过订单号查找订单信息
	orderInfo, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(orderNo),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[CloseOrder] 查找订单信息失败: %v, orderNo: %s", err, orderNo)
		return nil
	}

	// 如果订单状态不是1，说明订单已关闭或已支付
	if orderInfo.Status != 1 {
		r.logger.Infof("[CloseOrder] Order status is not 1, orderNo: %s, status: %d", orderNo, orderInfo.Status)
		return nil
	}

	subscribeInfo, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(orderInfo.SubscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[CloseOrder] 查找订阅信息失败: %v, subscribeID: %d, orderNo: %s", err, orderInfo.SubscribeID, orderNo)
		return nil
	}

	// 使用TX辅助函数执行事务
	return r.data.db.TX(ctx, func(tx *ent.Tx) error {
		// 更新订单状态为已关闭(3)
		err := tx.ProxyOrder.UpdateOneID(orderInfo.ID).
			SetStatus(3).
			SetUpdatedAt(time.Now()).
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[CloseOrder] Update order status failed: %v, orderNo: %s", err, orderNo)
			return errors.InternalServer("ORDER_UPDATE_FAILED", "关闭订单失败")
		}

		if orderInfo.Type == 1 && subscribeInfo.Inventory != -1 {
			err = tx.ProxySubscribe.UpdateOneID(subscribeInfo.ID).
				AddInventory(1).
				Exec(ctx)
			if err != nil {
				r.logger.Errorf("[CloseOrder] Restore subscribe inventory failed: %v, subscribeID: %d", err, subscribeInfo.ID)
				return err
			}
		}

		if (orderInfo.Type == 1 || orderInfo.Type == 2) && orderInfo.PriceOptionID > 0 {
			optionInfo, err := tx.ProxySubscribePriceOption.Query().
				Where(proxysubscribepriceoption.IDEQ(orderInfo.PriceOptionID)).
				Only(ctx)
			if err == nil && optionInfo.Inventory != -1 {
				if err := tx.ProxySubscribePriceOption.UpdateOneID(optionInfo.ID).
					AddInventory(1).
					Exec(ctx); err != nil {
					r.logger.Errorf("[CloseOrder] Restore price option inventory failed: %v, optionID: %d", err, optionInfo.ID)
					return err
				}
			}
		}

		// 如果用户ID为0，说明是访客订单，无需退款，可直接删除
		if orderInfo.UserID == 0 {
			err = tx.ProxyOrder.DeleteOneID(orderInfo.ID).Exec(ctx)
			if err != nil {
				r.logger.Errorf("[CloseOrder] Delete order failed: %v, orderNo: %s", err, orderNo)
				return errors.InternalServer("ORDER_DELETE_FAILED", "删除订单失败")
			}
			return nil
		}

		// 如果存在赠金，则退回给用户
		if orderInfo.GiftAmount > 0 {
			userInfo, err := tx.ProxyUser.Query().
				Where(
					proxyuser.IDEQ(orderInfo.UserID),
				).
				Only(ctx)
			if err != nil {
				r.logger.Errorf("[CloseOrder] 查找用户 info failed: %v, userID: %d", err, orderInfo.UserID)
				return errors.InternalServer("USER_NOT_FOUND", "用户不存在")
			}

			// 计算新的赠金金额
			currentGiftAmount := int64(0)
			if userInfo.GiftAmount != nil {
				currentGiftAmount = *userInfo.GiftAmount
			}
			newGiftAmount := currentGiftAmount + orderInfo.GiftAmount

			// 更新用户赠金金额
			err = tx.ProxyUser.UpdateOneID(orderInfo.UserID).
				SetGiftAmount(newGiftAmount).
				Exec(ctx)
			if err != nil {
				r.logger.Errorf("[CloseOrder] Refund gift amount failed: %v, userID: %d, giftAmount: %d", err, orderInfo.UserID, orderInfo.GiftAmount)
				return errors.InternalServer("GIFT_REFUND_FAILED", "退回赠金失败")
			}

			// 创建赠金退回日志记录 (已修复：之前缺失)
			giftLogContent := fmt.Sprintf(`{"type":"increase","order_no":"%s","subscribe_id":0,"amount":%d,"balance":%d,"remark":"订单取消退款","timestamp":%d}`,
				orderInfo.OrderNo, orderInfo.GiftAmount, newGiftAmount, time.Now().UnixMilli())

			_, err = tx.ProxySystemLog.Create().
				SetType(34).
				SetDate(time.Now().Format("2006-01-02")).
				SetObjectID(orderInfo.UserID).
				SetContent(giftLogContent).
				Save(ctx)
			if err != nil {
				r.logger.Errorf("[CloseOrder] Create gift refund log failed: %v", err)
				return errors.InternalServer("GIFT_REFUND_LOG_FAILED", "记录赠金退款日志失败")
			}

			r.logger.Infof("[CloseOrder] Refunded gift amount: %d to user: %d, new balance: %d", orderInfo.GiftAmount, orderInfo.UserID, newGiftAmount)
			return nil
		}

		return nil
	})
}

// QueryOrderDetail 查询订单详情及订阅和支付信息
func (r *publicOrderRepo) QueryOrderDetail(ctx context.Context, userID int, orderNo string) (*publicBiz.OrderDetail, error) {
	order, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(orderNo),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[PublicOrderRepo.QueryOrderDetail] query order failed: %v, orderNo: %s", err, orderNo)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// 查询完整的订阅信息
	var subscribe *publicBiz.Subscribe
	if order.SubscribeID > 0 {
		subscribeEnt, err := r.data.db.ProxySubscribe.Query().
			Where(proxysubscribe.IDEQ(order.SubscribeID)).
			Only(ctx)
		if err == nil {
			subscribe = r.convertToSubscribe(ctx, subscribeEnt)
		}
	}

	// 查询完整的支付方式信息
	var payment *publicBiz.PaymentMethod
	if order.PaymentID != 0 {
		paymentEnt, err := r.data.db.ProxyPayment.Query().
			Where(proxypayment.IDEQ(order.PaymentID)).
			Only(ctx)
		if err == nil {
			payment = r.convertToPaymentMethod(paymentEnt)
		}
	}

	return r.convertToOrderDetailFull(order, subscribe, payment), nil
}

// QueryOrderList 查询订单列表及订阅和支付信息
func (r *publicOrderRepo) QueryOrderList(ctx context.Context, userID int, page, size int, status, orderType int32) ([]*publicBiz.OrderDetail, int32, error) {
	query := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.UserIDEQ(int64(userID)),
		)

	// 应用过滤条件
	if status > 0 {
		query = query.Where(proxyorder.StatusEQ(int8(status)))
	}
	if orderType > 0 {
		query = query.Where(proxyorder.TypeEQ(int8(orderType)))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("[PublicOrderRepo.QueryOrderList] count orders failed: %v", err)
		return nil, int32(total), errors.InternalServer("ORDER_COUNT_FAILED", "统计订单失败")
	}

	// 应用分页
	offset := (page - 1) * size
	orders, err := query.
		Offset(int(offset)).
		Limit(int(size)).
		Order(ent.Desc(proxyorder.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("[PublicOrderRepo.QueryOrderList] query orders failed: %v", err)
		return nil, int32(total), errors.InternalServer("ORDER_QUERY_FAILED", "查询订单失败")
	}

	// 收集订阅ID和支付ID用于批量加载
	subscribeIDs := make([]int64, 0)
	paymentIDs := make([]int64, 0)
	for _, order := range orders {
		if order.SubscribeID > 0 {
			subscribeIDs = append(subscribeIDs, order.SubscribeID)
		}
		if order.PaymentID > 0 {
			paymentIDs = append(paymentIDs, order.PaymentID)
		}
	}

	// 批量加载订阅完整对象和名称
	subscribeMap := make(map[int64]*publicBiz.Subscribe)
	subscribeNameMap := make(map[int64]string)
	if len(subscribeIDs) > 0 {
		subscribes, err := r.data.db.ProxySubscribe.Query().
			Where(proxysubscribe.IDIn(subscribeIDs...)).
			All(ctx)
		if err == nil {
			for _, sub := range subscribes {
				subscribeNameMap[int64(sub.ID)] = sub.Name
				subscribeMap[int64(sub.ID)] = r.convertToSubscribe(ctx, sub)
			}
		}
	}

	// 批量加载支付完整对象和名称
	paymentMap := make(map[int64]*publicBiz.PaymentMethod)
	paymentNameMap := make(map[int64]string)
	if len(paymentIDs) > 0 {
		payments, err := r.data.db.ProxyPayment.Query().
			Where(proxypayment.IDIn(paymentIDs...)).
			All(ctx)
		if err == nil {
			for _, payment := range payments {
				paymentNameMap[int64(payment.ID)] = payment.Name
				paymentMap[int64(payment.ID)] = r.convertToPaymentMethod(payment)
			}
		}
	}

	// 转换为订单详情列表
	result := make([]*publicBiz.OrderDetail, len(orders))
	for i, order := range orders {
		subscribe := subscribeMap[order.SubscribeID]
		subscribeName := subscribeNameMap[order.SubscribeID]
		payment := paymentMap[order.PaymentID]
		paymentName := paymentNameMap[order.PaymentID]
		result[i] = r.convertToOrderDetailFull(order, subscribe, payment)
		// 确保使用名称字段（向后兼容）
		result[i].SubscribeName = subscribeName
		result[i].PaymentName = paymentName
		// 防止佣金金额泄露
		result[i].Commission = 0
	}

	return result, int32(total), nil
}

// PreCreateOrder 验证并计算订单价格
func (r *publicOrderRepo) PreCreateOrder(ctx context.Context, req *publicBiz.PreCreateOrderParams) (*publicBiz.PreCreateOrderResult, error) {
	// 查找订阅套餐 (已修复：使用ProxySubscribe而非ProxySubscribeGroup)
	sub, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(req.SubscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[PreCreateOrder] 数据库查询错误, subscribeID: %d, error: %v", req.SubscribeID, err)
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
	}

	if req.Type != 2 && sub.Quota > 0 {
		userSubs, err := r.listLegacyValidUserSubscriptions(ctx, req.UserID)
		if err != nil {
			r.logger.Errorf("[PreCreateOrder] 查询用户订阅失败, userID: %d, error: %v", req.UserID, err)
			return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}
		var count int64
		for _, item := range userSubs {
			if item.SubscribeID == req.SubscribeID {
				count++
			}
		}
		if count >= int64(sub.Quota) {
			return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
	}

	option, err := r.getSellablePriceOption(ctx, req.SubscribeID, req.PriceOptionID)
	if err != nil {
		return nil, err
	}
	price, amount, discountAmount := priceFromOption(option)
	if amount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	var couponAmount int64

	// 验证并计算优惠券折扣
	if req.Coupon != "" {
		couponInfo, err := r.data.db.ProxyCoupon.Query().
			Where(
				proxycoupon.CodeEQ(req.Coupon),
			).
			Only(ctx)
		if err != nil {
			r.logger.Errorf("[PreCreateOrder] Coupon not found: %s", req.Coupon)
			return nil, responsecode.NewKratosError(responsecode.ErrCouponNotFound)
		}

		if couponInfo.Count > 0 && couponInfo.Count <= int32(couponInfo.UsedCount) {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUsedUp)
		}

		// 检查用户使用限制
		count, err := r.data.db.ProxyOrder.Query().
			Where(
				proxyorder.UserIDEQ(req.UserID),
				proxyorder.CouponEQ(req.Coupon),
			).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("[PreCreateOrder] 数据库查询错误: %v", err)
			return nil, errors.InternalServer("DATABASE_QUERY_ERROR", "数据库查询失败")
		}

		if couponInfo.UserLimit > 0 && int64(count) >= couponInfo.UserLimit {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUserLimitExceeded)
		}

		couponSub := tool.StringToInt64Slice(couponInfo.Subscribe)
		if len(couponSub) > 0 && !tool.Contains(couponSub, req.SubscribeID) {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponNotAvailable)
		}

		couponAmount = r.calculateCoupon(amount, couponInfo)
	}
	amount -= couponAmount

	var feeAmount int64
	if req.Payment > 0 {
		payment, err := r.data.db.ProxyPayment.Query().
			Where(
				proxypayment.IDEQ(req.Payment),
			).
			Only(ctx)
		if err != nil {
			r.logger.Errorf("[PreCreateOrder] Payment method not found: %d", req.Payment)
			return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
		}

		// 按老项目顺序：先加手续费，再计算赠金抵扣
		if amount > 0 {
			feeAmount = r.calculateFee(amount, payment)
			amount += feeAmount
		}
	}

	var deductionAmount int64
	// 检查用户赠金金额
	userInfo, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(req.UserID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[PreCreateOrder] 用户不存在: %d", req.UserID)
		return nil, errors.InternalServer("USER_NOT_FOUND", "用户不存在")
	}

	if userInfo.GiftAmount != nil && *userInfo.GiftAmount > 0 {
		if *userInfo.GiftAmount >= amount {
			deductionAmount = amount
			amount = 0
		} else {
			deductionAmount = *userInfo.GiftAmount
			amount -= *userInfo.GiftAmount
		}
	}
	if amount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	resp := &publicBiz.PreCreateOrderResult{
		Price:          price,
		Amount:         amount,
		Discount:       discountAmount,
		CouponDiscount: couponAmount,
		FeeAmount:      feeAmount,
		Commission:     0, // 待办：如需要计算佣金
		GiftAmount:     deductionAmount,
		Valid:          true,
		Message:        "Order preview calculated successfully",
	}

	_ = sub // 避免未使用变量警告
	return resp, nil
}

// Purchase 创建购买订单
func (r *publicOrderRepo) Purchase(ctx context.Context, req *publicBiz.PurchaseParams) (*publicBiz.OrderResult, error) {
	if req.SubscribeID <= 0 {
		return nil, errors.BadRequest("INVALID_SUBSCRIBE_ID", "Invalid subscribe ID")
	}

	// 查找用户
	userInfo, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(req.UserID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Purchase] 用户不存在: %d", req.UserID)
		return nil, errors.InternalServer("USER_NOT_FOUND", "用户不存在")
	}

	// 检查单一模式限制（已修复：之前缺失）
	// 如果启用单一模式，用户一次只能有一个活动订阅
	singleModelEnabled := r.config != nil && r.config.Subscribe != nil && r.config.Subscribe.SingleModel
	if subscribeValues, err := loadSystemConfigMap(ctx, r.data.db, "subscribe"); err == nil {
		singleModelEnabled = systemConfigBool(subscribeValues, singleModelEnabled, "SingleModel", "single_model")
	}
	if singleModelEnabled {
		existingSubscriptions, err := r.listLegacyValidUserSubscriptions(ctx, req.UserID)
		if err != nil {
			r.logger.Errorf("[Purchase] Check existing subscriptions failed: %v", err)
			return nil, errors.InternalServer("CHECK_SUBSCRIPTION_FAILED", "检查现有订阅失败")
		}
		if len(existingSubscriptions) > 0 {
			r.logger.Warnf("[Purchase] Single model restriction: user %d already has %d subscription(s)", req.UserID, len(existingSubscriptions))
			return nil, errors.BadRequest("USER_SUBSCRIPTION_EXISTS", "用户已有活动订阅。每个用户只允许一个订阅。")
		}
	}

	// 查找订阅套餐 (已修复：使用ProxySubscribe而非ProxySubscribeGroup)
	sub, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(req.SubscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Purchase] Subscribe not found: %d", req.SubscribeID)
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
	}

	// 检查订阅计划状态（售卖标志）
	if !sub.Sell {
		r.logger.Warnf("[Purchase] Subscribe not for sale: %d", req.SubscribeID)
		return nil, errors.BadRequest("SUBSCRIBE_NOT_FOR_SALE", "此订阅计划不可购买")
	}
	if sub.Inventory == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeOutOfStock)
	}
	option, err := r.getSellablePriceOption(ctx, req.SubscribeID, req.PriceOptionID)
	if err != nil {
		return nil, err
	}

	// 检查订阅计划配额（每个计划的用户购买限制）
	// 与老项目 QueryUserSubscribe 及当前项目用户侧有效订阅口径保持一致
	if sub.Quota > 0 {
		existingSubscriptions, err := r.listLegacyValidUserSubscriptions(ctx, req.UserID)
		if err != nil {
			r.logger.Errorf("[Purchase] Check quota failed: %v", err)
			return nil, errors.InternalServer("QUOTA_CHECK_FAILED", "检查配额失败")
		}
		var existingCount int64
		for _, item := range existingSubscriptions {
			if item.SubscribeID == req.SubscribeID {
				existingCount++
			}
		}
		if int64(existingCount) >= int64(sub.Quota) {
			r.logger.Warnf("[Purchase] Quota exceeded: user %d, subscribe %d, count %d, quota %d", req.UserID, req.SubscribeID, existingCount, sub.Quota)
			return nil, errors.BadRequest("QUOTA_EXCEEDED", "订阅配额限制已超过")
		}
	}

	price, amount, discountAmount := priceFromOption(option)
	if amount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	var coupon int64 = 0

	// 计算优惠券扣除金额
	if req.Coupon != "" {
		couponInfo, err := r.data.db.ProxyCoupon.Query().
			Where(
				proxycoupon.CodeEQ(req.Coupon),
			).
			Only(ctx)
		if err != nil {
			r.logger.Errorf("[Purchase] Coupon not found: %s", req.Coupon)
			return nil, responsecode.NewKratosError(responsecode.ErrCouponNotFound)
		}

		if couponInfo.Count != 0 && couponInfo.Count <= int32(couponInfo.UsedCount) {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUsedUp)
		}
		couponSub := tool.StringToInt64Slice(couponInfo.Subscribe)
		if len(couponSub) > 0 && !tool.Contains(couponSub, req.SubscribeID) {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponNotAvailable)
		}

		count, err := r.data.db.ProxyOrder.Query().
			Where(
				proxyorder.UserIDEQ(req.UserID),
				proxyorder.CouponEQ(req.Coupon),
			).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("[Purchase] 数据库查询错误: %v", err)
			return nil, errors.InternalServer("DATABASE_QUERY_ERROR", "数据库查询失败")
		}

		if couponInfo.UserLimit > 0 && int64(count) >= couponInfo.UserLimit {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUserLimitExceeded)
		}

		coupon = r.calculateCoupon(amount, couponInfo)
	}

	amount -= coupon
	// 查找支付方式
	payment, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(req.Payment),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Purchase] Payment method not found: %d", req.Payment)
		return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
	}

	var feeAmount int64
	// 按老项目顺序：先加手续费，再计算赠金抵扣
	if amount > 0 {
		feeAmount = r.calculateFee(amount, payment)
		amount += feeAmount
	}

	var deductionAmount int64
	currentGiftAmount := int64(0)
	if userInfo.GiftAmount != nil {
		currentGiftAmount = *userInfo.GiftAmount
	}

	if currentGiftAmount > 0 {
		if currentGiftAmount >= amount {
			deductionAmount = amount
			amount = 0
			currentGiftAmount -= deductionAmount
		} else {
			deductionAmount = currentGiftAmount
			amount -= currentGiftAmount
			currentGiftAmount = 0
		}
	}

	if amount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 与老项目一致：只要用户存在已完成/已激活订单(status in 2,5)，则不再属于新购
	isNew, err := r.isUserEligibleForNewOrder(ctx, req.UserID)
	if err != nil {
		r.logger.Errorf("[Purchase] 查询用户新购资格失败: %v, userID: %d", err, req.UserID)
		return nil, errors.InternalServer("DATABASE_QUERY_ERROR", "数据库查询失败")
	}

	// 生成订单号
	orderNo := tool.GenerateTradeNo()

	// 创建订单
	orderInfo := &ent.ProxyOrder{

		UserID:          req.UserID,
		OrderNo:         orderNo,
		Type:            1, // 购买类型
		Quantity:        orderDurationQuantity(option),
		Price:           price,
		Amount:          amount,
		Discount:        discountAmount,
		GiftAmount:      deductionAmount,
		Coupon:          req.Coupon,
		CouponDiscount:  coupon,
		PaymentID:       payment.ID,
		Method:          payment.Platform,
		FeeAmount:       feeAmount,
		Status:          1, // 待付款
		IsNew:           isNew,
		SubscribeID:     req.SubscribeID,
		PriceOptionID:   option.ID,
		PriceOptionName: option.Name,
		DurationUnit:    option.DurationUnit,
		DurationValue:   option.DurationValue,
		OptionPrice:     option.Price,
	}

	// 使用TX辅助函数执行事务
	var createdOrder *ent.ProxyOrder
	err = r.data.db.TX(ctx, func(tx *ent.Tx) error {
		if sub.Quota > 0 {
			currentSubscriptions, err := tx.ProxyUserSubscribe.Query().
				Where(proxyusersubscribe.UserIDEQ(req.UserID)).
				All(ctx)
			if err != nil {
				return errors.InternalServer("QUOTA_CHECK_FAILED", "检查配额失败")
			}
			var currentCount int64
			now := time.Now()
			for _, item := range currentSubscriptions {
				if !shouldKeepLegacyUserSubscribe(item, now) {
					continue
				}
				if item.SubscribeID == req.SubscribeID {
					currentCount++
				}
			}
			if int64(currentCount) >= int64(sub.Quota) {
				return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
			}
		}

		if sub.Inventory != -1 {
			if err := decrementSubscribeInventory(ctx, tx, req.SubscribeID); err != nil {
				return err
			}
		}
		if option.Inventory != -1 {
			if err := decrementDurationPriceOptionInventory(ctx, tx, req.SubscribeID, option.ID); err != nil {
				return err
			}
		}

		// 更新用户赠金金额 如果存在扣除
		if orderInfo.GiftAmount > 0 {
			err := tx.ProxyUser.UpdateOneID(req.UserID).
				SetGiftAmount(currentGiftAmount).
				Exec(ctx)
			if err != nil {
				r.logger.Errorf("[Purchase] Update user gift amount failed: %v", err)
				return errors.InternalServer("USER_UPDATE_FAILED", "更新用户失败")
			}

			// 在系统日志中创建赠金日志记录
			giftLogContent := fmt.Sprintf(`{"type":"reduce","order_no":"%s","subscribe_id":0,"amount":%d,"balance":%d,"remark":"购买订单扣除","timestamp":%d}`,
				orderInfo.OrderNo, orderInfo.GiftAmount, currentGiftAmount, time.Now().UnixMilli())

			_, err = tx.ProxySystemLog.Create().
				SetType(34). // 34 = Gift log type
				SetDate(time.Now().Format("2006-01-02")).
				SetObjectID(req.UserID).
				SetContent(giftLogContent).
				Save(ctx)
			if err != nil {
				r.logger.Errorf("[Purchase] Create gift log failed: %v", err)
				// 日志失败不回滚，仅记录错误
			}

			r.logger.Infof("[Purchase] Deducted gift amount: %d from user: %d, remaining: %d", orderInfo.GiftAmount, req.UserID, currentGiftAmount)
		}

		// 插入订单
		order, err := tx.ProxyOrder.Create().
			SetUserID(orderInfo.UserID).
			SetOrderNo(orderInfo.OrderNo).
			SetType(orderInfo.Type).
			SetQuantity(orderInfo.Quantity).
			SetPrice(orderInfo.Price).
			SetAmount(orderInfo.Amount).
			SetDiscount(orderInfo.Discount).
			SetGiftAmount(orderInfo.GiftAmount).
			SetCoupon(orderInfo.Coupon).
			SetCouponDiscount(orderInfo.CouponDiscount).
			SetPaymentID(orderInfo.PaymentID).
			SetMethod(orderInfo.Method).
			SetFeeAmount(orderInfo.FeeAmount).
			SetStatus(orderInfo.Status).
			SetIsNew(orderInfo.IsNew).
			SetSubscribeID(orderInfo.SubscribeID).
			SetPriceOptionID(orderInfo.PriceOptionID).
			SetPriceOptionName(orderInfo.PriceOptionName).
			SetDurationUnit(orderInfo.DurationUnit).
			SetDurationValue(orderInfo.DurationValue).
			SetOptionPrice(orderInfo.OptionPrice).
			Save(ctx)
		if err != nil {
			r.logger.Errorf("[Purchase] Insert order failed: %v", err)
			return errors.InternalServer("ORDER_CREATE_FAILED", "创建订单失败")
		}
		createdOrder = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 与旧项目保持一致：创建订单后统一进入待支付状态，不在创建接口内立即扣款
	r.enqueueDeferCloseOrderTask(ctx, int(req.UserID), createdOrder.OrderNo)

	// 根据支付方式生成支付信息
	paymentURL, qrCode := r.generatePaymentInfo(ctx, createdOrder, payment)

	return &publicBiz.OrderResult{
		OrderID:    createdOrder.ID,
		OrderNo:    createdOrder.OrderNo,
		Amount:     createdOrder.Amount,
		PaymentURL: paymentURL,
		QRCode:     qrCode,
	}, nil
}

// Recharge 创建充值订单
func (r *publicOrderRepo) Recharge(ctx context.Context, req *publicBiz.RechargeParams) (*publicBiz.OrderResult, error) {
	if req.Amount <= 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if req.Amount > MaxRechargeAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 查找支付方式
	payment, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(req.Payment),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Recharge] Payment method not found: %d", req.Payment)
		return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
	}

	// 计算手续费
	feeAmount := r.calculateFee(req.Amount, payment)
	totalAmount := req.Amount + feeAmount
	if totalAmount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 与老项目一致：只要用户存在已完成/已激活订单(status in 2,5)，则不再属于新购
	isNew, err := r.isUserEligibleForNewOrder(ctx, req.UserID)
	if err != nil {
		r.logger.Errorf("[Recharge] 查询用户新购资格失败: %v, userID: %d", err, req.UserID)
		return nil, errors.InternalServer("DATABASE_QUERY_ERROR", "数据库查询失败")
	}

	// 生成订单号
	orderNo := tool.GenerateTradeNo()

	// 创建订单
	orderInfo, err := r.data.db.ProxyOrder.Create().
		SetUserID(req.UserID).
		SetOrderNo(orderNo).
		SetType(4). // 充值类型
		SetPrice(req.Amount).
		SetAmount(totalAmount).
		SetFeeAmount(feeAmount).
		SetPaymentID(payment.ID).
		SetMethod(payment.Platform).
		SetStatus(1). // 待付款
		SetIsNew(isNew).
		Save(ctx)
	if err != nil {
		r.logger.Errorf("[Recharge] Insert order failed: %v", err)
		return nil, errors.InternalServer("ORDER_CREATE_FAILED", "创建订单失败")
	}

	// 与旧项目保持一致：创建订单后统一进入待支付状态，不在创建接口内立即扣款
	r.enqueueDeferCloseOrderTask(ctx, int(req.UserID), orderInfo.OrderNo)

	// 根据支付方式生成支付信息
	paymentURL, qrCode := r.generatePaymentInfo(ctx, orderInfo, payment)

	return &publicBiz.OrderResult{
		OrderID:    orderInfo.ID,
		OrderNo:    orderInfo.OrderNo,
		Amount:     orderInfo.Amount,
		PaymentURL: paymentURL,
		QRCode:     qrCode,
	}, nil
}

// Renewal 创建续费订单
func (r *publicOrderRepo) Renewal(ctx context.Context, req *publicBiz.RenewalParams) (*publicBiz.OrderResult, error) {
	// 查找用户订阅
	userSubscribe, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.IDEQ(req.UserSubscribeID),
			proxyusersubscribe.UserIDEQ(req.UserID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Renewal] Query user subscribe failed: %v, userSubscribeID: %d", err, req.UserSubscribeID)
		return nil, responsecode.NewDatabaseQueryError()
	}

	// 查找订阅（已修复：使用ProxySubscribe而非ProxySubscribeGroup）
	sub, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(userSubscribe.SubscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Renewal] Query subscribe failed: %v, subscribeID: %d", err, userSubscribe.SubscribeID)
		return nil, responsecode.NewDatabaseQueryError()
	}

	// 检查订阅计划状态（售卖标志）
	if !sub.Sell {
		r.logger.Warnf("[Renewal] Subscribe not for sale: %d", userSubscribe.SubscribeID)
		return nil, errors.BadRequest("SUBSCRIBE_NOT_FOR_SALE", "此订阅计划不可续费")
	}

	option, err := r.getSellablePriceOption(ctx, userSubscribe.SubscribeID, req.PriceOptionID)
	if err != nil {
		return nil, err
	}
	price, amount, discountAmount := priceFromOption(option)
	if amount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	var coupon int64 = 0

	// 计算优惠券扣除金额
	if req.Coupon != "" {
		couponInfo, err := r.data.db.ProxyCoupon.Query().
			Where(
				proxycoupon.CodeEQ(req.Coupon),
			).
			Only(ctx)
		if err != nil {
			r.logger.Errorf("[Renewal] Coupon not found: %s", req.Coupon)
			return nil, responsecode.NewKratosError(responsecode.ErrCouponNotFound)
		}

		if couponInfo.Count != 0 && couponInfo.Count <= int32(couponInfo.UsedCount) {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUsedUp)
		}
		couponSub := tool.StringToInt64Slice(couponInfo.Subscribe)
		if len(couponSub) > 0 && !tool.Contains(couponSub, userSubscribe.SubscribeID) {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponNotAvailable)
		}

		count, err := r.data.db.ProxyOrder.Query().
			Where(
				proxyorder.UserIDEQ(req.UserID),
				proxyorder.CouponEQ(req.Coupon),
			).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("[Renewal] 数据库查询错误: %v", err)
			return nil, errors.InternalServer("DATABASE_QUERY_ERROR", "数据库查询失败")
		}

		if couponInfo.UserLimit > 0 && int64(count) >= couponInfo.UserLimit {
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUserLimitExceeded)
		}

		coupon = r.calculateCoupon(amount, couponInfo)
	}

	amount -= coupon

	// 查找用户
	userInfo, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(req.UserID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Renewal] 用户不存在: %d", req.UserID)
		return nil, errors.InternalServer("USER_NOT_FOUND", "用户不存在")
	}

	var deductionAmount int64
	currentGiftAmount := int64(0)
	if userInfo.GiftAmount != nil {
		currentGiftAmount = *userInfo.GiftAmount
	}

	// 检查用户扣除金额
	if currentGiftAmount > 0 {
		if currentGiftAmount >= amount {
			deductionAmount = amount
			currentGiftAmount -= deductionAmount
			amount = 0
		} else {
			deductionAmount = currentGiftAmount
			amount -= currentGiftAmount
			currentGiftAmount = 0
		}
	}

	// 查找支付方式
	payment, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(req.Payment),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[Renewal] Payment method not found: %d", req.Payment)
		return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
	}

	var feeAmount int64
	// 计算手续费
	if amount > 0 {
		feeAmount = r.calculateFee(amount, payment)
	}
	amount += feeAmount
	if amount > MaxOrderAmount {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 生成订单号
	orderNo := tool.GenerateTradeNo()

	// 获取订阅令牌
	subscribeToken := ""
	if userSubscribe.Token != nil {
		subscribeToken = *userSubscribe.Token
	}

	// 使用TX辅助函数执行事务
	var createdOrder *ent.ProxyOrder
	err = r.data.db.TX(ctx, func(tx *ent.Tx) error {
		if option.Inventory != -1 {
			if err := decrementDurationPriceOptionInventory(ctx, tx, userSubscribe.SubscribeID, option.ID); err != nil {
				return err
			}
		}

		// 更新用户赠金金额 如果存在扣除
		if deductionAmount > 0 {
			err := tx.ProxyUser.UpdateOneID(req.UserID).
				SetGiftAmount(currentGiftAmount).
				Exec(ctx)
			if err != nil {
				r.logger.Errorf("[Renewal] Update user gift amount failed: %v", err)
				return errors.InternalServer("USER_UPDATE_FAILED", "更新用户失败")
			}

			// 在系统日志中创建赠金日志记录
			giftLogContent := fmt.Sprintf(`{"type":"reduce","order_no":"%s","subscribe_id":0,"amount":%d,"balance":%d,"remark":"续费订单扣除","timestamp":%d}`,
				orderNo, deductionAmount, currentGiftAmount, time.Now().UnixMilli())

			_, err = tx.ProxySystemLog.Create().
				SetType(34). // 34 = Gift log type
				SetDate(time.Now().Format("2006-01-02")).
				SetObjectID(req.UserID).
				SetContent(giftLogContent).
				Save(ctx)
			if err != nil {
				r.logger.Errorf("[Renewal] Create gift log failed: %v", err)
				// 日志失败不回滚，仅记录错误
			}

			r.logger.Infof("[Renewal] Deducted gift amount: %d from user: %d, remaining: %d", deductionAmount, req.UserID, currentGiftAmount)
		}

		// 创建订单
		order, err := tx.ProxyOrder.Create().
			SetUserID(req.UserID).
			SetParentID(userSubscribe.OrderID).
			SetOrderNo(orderNo).
			SetType(2). // 续费类型
			SetQuantity(orderDurationQuantity(option)).
			SetPrice(price).
			SetAmount(amount).
			SetGiftAmount(deductionAmount).
			SetDiscount(discountAmount).
			SetCoupon(req.Coupon).
			SetCouponDiscount(coupon).
			SetPaymentID(payment.ID).
			SetMethod(payment.Platform).
			SetFeeAmount(feeAmount).
			SetStatus(1). // 待付款
			SetSubscribeID(userSubscribe.SubscribeID).
			SetPriceOptionID(option.ID).
			SetPriceOptionName(option.Name).
			SetDurationUnit(option.DurationUnit).
			SetDurationValue(option.DurationValue).
			SetOptionPrice(option.Price).
			SetSubscribeToken(subscribeToken).
			Save(ctx)
		if err != nil {
			r.logger.Errorf("[Renewal] Insert order failed: %v", err)
			return errors.InternalServer("ORDER_CREATE_FAILED", "创建订单失败")
		}
		createdOrder = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 与旧项目保持一致：创建订单后统一进入待支付状态，不在创建接口内立即扣款
	r.enqueueDeferCloseOrderTask(ctx, int(req.UserID), createdOrder.OrderNo)

	// 根据支付方式生成支付信息
	paymentURL, qrCode := r.generatePaymentInfo(ctx, createdOrder, payment)

	_ = sub // 避免未使用变量警告

	return &publicBiz.OrderResult{
		OrderID:    createdOrder.ID,
		OrderNo:    createdOrder.OrderNo,
		Amount:     createdOrder.Amount,
		PaymentURL: paymentURL,
		QRCode:     qrCode,
	}, nil
}

// ResetTraffic 创建流量重置订单
func (r *publicOrderRepo) ResetTraffic(ctx context.Context, req *publicBiz.ResetTrafficParams) (*publicBiz.OrderResult, error) {
	// 查找用户订阅
	userSubscribe, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.IDEQ(req.UserSubscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[ResetTraffic] User subscribe not found: %d", req.UserSubscribeID)
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
	}

	// 查找订阅以获取重置金额（已修复：查询ProxySubscribe获取Replacement）
	subscribe, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(userSubscribe.SubscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[ResetTraffic] Subscribe not found: %d", userSubscribe.SubscribeID)
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
	}

	// 使用订阅的重置金额
	amount := int64(subscribe.Replacement)

	// 查找用户
	userInfo, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(req.UserID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[ResetTraffic] 用户不存在: %d", req.UserID)
		return nil, errors.InternalServer("USER_NOT_FOUND", "用户不存在")
	}

	var deductionAmount int64
	currentGiftAmount := int64(0)
	if userInfo.GiftAmount != nil {
		currentGiftAmount = *userInfo.GiftAmount
	}

	// 检查用户扣除金额
	if currentGiftAmount > 0 {
		if currentGiftAmount >= amount {
			deductionAmount = amount
			amount = 0
			currentGiftAmount -= amount
		} else {
			deductionAmount = currentGiftAmount
			amount -= currentGiftAmount
			currentGiftAmount = 0
		}
	}

	// 查找支付方式
	payment, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(req.Payment),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[ResetTraffic] Payment method not found: %d", req.Payment)
		return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
	}

	var feeAmount int64
	// 计算手续费
	if amount > 0 {
		feeAmount = r.calculateFee(amount, payment)
	}

	// 获取订阅令牌
	subscribeToken := ""
	if userSubscribe.Token != nil {
		subscribeToken = *userSubscribe.Token
	}

	// 生成订单号
	orderNo := tool.GenerateTradeNo()

	// 使用TX辅助函数执行事务
	var createdOrder *ent.ProxyOrder
	err = r.data.db.TX(ctx, func(tx *ent.Tx) error {
		// 更新用户赠金金额 如果存在扣除
		if deductionAmount > 0 {
			err := tx.ProxyUser.UpdateOneID(req.UserID).
				SetGiftAmount(currentGiftAmount).
				Exec(ctx)
			if err != nil {
				r.logger.Errorf("[ResetTraffic] Update user gift amount failed: %v", err)
				return errors.InternalServer("USER_UPDATE_FAILED", "更新用户失败")
			}

			// 在系统日志中创建赠金日志记录
			giftLogContent := fmt.Sprintf(`{"type":"reduce","order_no":"%s","subscribe_id":0,"amount":%d,"balance":%d,"remark":"重置流量订单扣除","timestamp":%d}`,
				orderNo, deductionAmount, currentGiftAmount, time.Now().UnixMilli())

			_, err = tx.ProxySystemLog.Create().
				SetType(34). // 34 = Gift log type
				SetDate(time.Now().Format("2006-01-02")).
				SetObjectID(req.UserID).
				SetContent(giftLogContent).
				Save(ctx)
			if err != nil {
				r.logger.Errorf("[ResetTraffic] Create gift log failed: %v", err)
				// 日志失败不回滚，仅记录错误
			}

			r.logger.Infof("[ResetTraffic] Deducted gift amount: %d from user: %d, remaining: %d", deductionAmount, req.UserID, currentGiftAmount)
		}

		// 创建订单
		order, err := tx.ProxyOrder.Create().
			SetParentID(userSubscribe.OrderID).
			SetUserID(req.UserID).
			SetOrderNo(orderNo).
			SetType(3).                      // 重置流量类型
			SetPrice(subscribe.Replacement). // 已修复：使用实际重置价格
			SetAmount(amount + feeAmount).
			SetGiftAmount(deductionAmount).
			SetFeeAmount(feeAmount).
			SetPaymentID(payment.ID).
			SetMethod(payment.Platform).
			SetStatus(1). // 待付款
			SetSubscribeID(userSubscribe.SubscribeID).
			SetSubscribeToken(subscribeToken).
			Save(ctx)
		if err != nil {
			r.logger.Errorf("[ResetTraffic] Insert order failed: %v", err)
			return errors.InternalServer("ORDER_CREATE_FAILED", "创建订单失败")
		}
		createdOrder = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	// 与旧项目保持一致：创建订单后统一进入待支付状态，不在创建接口内立即扣款
	r.enqueueDeferCloseOrderTask(ctx, int(req.UserID), createdOrder.OrderNo)

	// 根据支付方式生成支付信息
	paymentURL, qrCode := r.generatePaymentInfo(ctx, createdOrder, payment)

	return &publicBiz.OrderResult{
		OrderID:    createdOrder.ID,
		OrderNo:    createdOrder.OrderNo,
		Amount:     createdOrder.Amount,
		PaymentURL: paymentURL,
		QRCode:     qrCode,
	}, nil
}

// generatePaymentInfo 根据支付方式生成支付URL和二维码
// 返回（支付URL，二维码）
func (r *publicOrderRepo) generatePaymentInfo(ctx context.Context, order *ent.ProxyOrder, payment *ent.ProxyPayment) (string, string) {
	// 如需要，在文件顶部导入支付包
	// 对于余额支付，不需要URL或二维码
	if payment.Platform == Balance {
		return "", ""
	}

	// 构建支付回调通知URL
	notifyURL := r.buildNotifyURL(ctx, payment)

	// 解析支付平台
	platform := paymentPlatform.ParsePlatform(payment.Platform)

	switch platform {
	case paymentPlatform.AlipayF2F:
		return r.generateAlipayF2FPayment(ctx, order, payment, notifyURL)

	case paymentPlatform.EPay:
		return r.generateEPayPayment(ctx, order, payment, notifyURL)

	case paymentPlatform.Stripe:
		return r.generateStripePayment(ctx, order, payment)

	case paymentPlatform.CryptoSaaS:
		return r.generateCryptoSaaSPayment(ctx, order, payment, notifyURL)

	default:
		r.logger.Warnf("[generatePaymentInfo] Unsupported payment platform: %s, order: %s", payment.Platform, order.OrderNo)
		return "", ""
	}
}

// convertToOrderDetail 将ent订单转换为biz订单详情（不含名称）
func (r *publicOrderRepo) convertToOrderDetail(order *ent.ProxyOrder) *publicBiz.OrderDetail {
	return r.convertToOrderDetailWithNames(order, "", "")
}

// convertToOrderDetailWithNames 将ent订单转换为biz订单详情（含订阅和支付名称）
func (r *publicOrderRepo) convertToOrderDetailWithNames(order *ent.ProxyOrder, subscribeName, paymentName string) *publicBiz.OrderDetail {
	return &publicBiz.OrderDetail{
		ID:              order.ID,
		ParentID:        order.ParentID,
		UserID:          order.UserID,
		OrderNo:         order.OrderNo,
		Type:            int32(order.Type),
		Quantity:        int64(order.Quantity),
		Price:           order.Price,
		Amount:          order.Amount,
		GiftAmount:      order.GiftAmount,
		Discount:        order.Discount,
		Coupon:          order.Coupon,
		CouponDiscount:  order.CouponDiscount,
		Commission:      order.Commission,
		Payment:         nil, // Will be populated if needed
		Method:          order.Method,
		FeeAmount:       order.FeeAmount,
		TradeNo:         order.TradeNo,
		Status:          int32(order.Status),
		SubscribeID:     order.SubscribeID,
		PriceOptionID:   order.PriceOptionID,
		PriceOptionName: order.PriceOptionName,
		DurationUnit:    order.DurationUnit,
		DurationValue:   order.DurationValue,
		OptionPrice:     order.OptionPrice,
		SubscribeToken:  order.SubscribeToken,
		IsNew:           order.IsNew,
		CreatedAt:       order.CreatedAt.UnixMilli(),
		UpdatedAt:       order.UpdatedAt.UnixMilli(),
		Subscribe:       nil, // Will be populated if needed
		SubscribeName:   subscribeName,
		PaymentName:     paymentName,
		StatusText:      r.getOrderStatusText(order.Status),
		TypeText:        r.getOrderTypeText(order.Type),
	}
}

// convertToOrderDetailFull 将ent订单转换为biz订单详情（含完整的订阅和支付对象）
func (r *publicOrderRepo) convertToOrderDetailFull(order *ent.ProxyOrder, subscribe *publicBiz.Subscribe, payment *publicBiz.PaymentMethod) *publicBiz.OrderDetail {
	// 获取名称（用于向后兼容）
	subscribeName := ""
	if subscribe != nil {
		subscribeName = subscribe.Name
	}
	paymentName := ""
	if payment != nil {
		paymentName = payment.Name
	}

	return &publicBiz.OrderDetail{
		ID:              order.ID,
		ParentID:        order.ParentID,
		UserID:          order.UserID,
		OrderNo:         order.OrderNo,
		Type:            int32(order.Type),
		Quantity:        int64(order.Quantity),
		Price:           order.Price,
		Amount:          order.Amount,
		GiftAmount:      order.GiftAmount,
		Discount:        order.Discount,
		Coupon:          order.Coupon,
		CouponDiscount:  order.CouponDiscount,
		Commission:      0, // 防止佣金金额泄露
		Payment:         payment,
		Method:          order.Method,
		FeeAmount:       order.FeeAmount,
		TradeNo:         order.TradeNo,
		Status:          int32(order.Status),
		SubscribeID:     order.SubscribeID,
		PriceOptionID:   order.PriceOptionID,
		PriceOptionName: order.PriceOptionName,
		DurationUnit:    order.DurationUnit,
		DurationValue:   order.DurationValue,
		OptionPrice:     order.OptionPrice,
		SubscribeToken:  order.SubscribeToken,
		IsNew:           order.IsNew,
		CreatedAt:       order.CreatedAt.UnixMilli(),
		UpdatedAt:       order.UpdatedAt.UnixMilli(),
		Subscribe:       subscribe,
		SubscribeName:   subscribeName,
		PaymentName:     paymentName,
		StatusText:      r.getOrderStatusText(order.Status),
		TypeText:        r.getOrderTypeText(order.Type),
	}
}

// convertToPaymentMethod 将ent支付方式转换为biz支付方式
func (r *publicOrderRepo) convertToPaymentMethod(payment *ent.ProxyPayment) *publicBiz.PaymentMethod {
	return &publicBiz.PaymentMethod{
		ID:          int64(payment.ID),
		Name:        payment.Name,
		Platform:    payment.Platform,
		Description: payment.Description,
		Icon:        payment.Icon,
		FeeMode:     int32(payment.FeeMode),
		FeePercent:  int64(payment.FeePercent),
		FeeAmount:   int64(payment.FeeAmount),
	}
}

// convertToSubscribe 将ent订阅转换为biz订阅
func (r *publicOrderRepo) convertToSubscribe(ctx context.Context, subscribe *ent.ProxySubscribe) *publicBiz.Subscribe {
	// 解析折扣信息
	var discounts []publicBiz.SubscribeDiscount
	if subscribe.Discount != nil && *subscribe.Discount != "" {
		var entDiscounts []SubscribeDiscount
		if err := json.Unmarshal([]byte(*subscribe.Discount), &entDiscounts); err == nil {
			for _, d := range entDiscounts {
				discounts = append(discounts, publicBiz.SubscribeDiscount{
					Quantity: d.Quantity,
					Discount: d.Discount,
				})
			}
		}
	}

	// 解析节点ID
	var nodes []int
	if subscribe.Nodes != "" {
		nodes64 := tool.StringToInt64Slice(subscribe.Nodes)
		nodes = make([]int, len(nodes64))
		for i, v := range nodes64 {
			nodes[i] = int(v)
		}
	}

	// 解析节点标签
	var nodeTags []string
	if subscribe.NodeTags != "" {
		if err := json.Unmarshal([]byte(subscribe.NodeTags), &nodeTags); err != nil {
			// 如果JSON解析失败，尝试作为逗号分隔的字符串处理
			nodeTags = []string{subscribe.NodeTags}
		}
	}

	// 处理 Description 指针
	description := ""
	if subscribe.Description != nil {
		description = *subscribe.Description
	}

	// 处理 DeductionRatio 指针
	deductionRatio := int64(0)
	if subscribe.DeductionRatio != nil {
		deductionRatio = int64(*subscribe.DeductionRatio)
	}

	nodeGroupID := ""
	if subscribe.NodeGroupID != nil {
		nodeGroupID = strconv.FormatInt(*subscribe.NodeGroupID, 10)
	}

	resetCycle := int64(0)
	if subscribe.ResetCycle != nil {
		resetCycle = int64(*subscribe.ResetCycle)
	}

	categoryName := ""
	if subscribe.CategoryID > 0 {
		if category, err := r.data.db.ProxySubscribeCategory.Query().
			Where(proxysubscribecategory.IDEQ(subscribe.CategoryID)).
			Only(ctx); err == nil {
			categoryName = category.Name
		}
	}

	return &publicBiz.Subscribe{
		ID:                int64(subscribe.ID),
		Name:              subscribe.Name,
		Language:          subscribe.Language,
		Description:       description,
		UnitPrice:         subscribe.UnitPrice,
		UnitTime:          subscribe.UnitTime,
		Discount:          discounts,
		Replacement:       int64(subscribe.Replacement),
		Inventory:         int64(subscribe.Inventory),
		Traffic:           subscribe.Traffic,
		SpeedLimit:        int64(subscribe.SpeedLimit),
		DeviceLimit:       int64(subscribe.DeviceLimit),
		Quota:             int64(subscribe.Quota),
		CategoryID:        subscribe.CategoryID,
		CategoryName:      categoryName,
		Nodes:             nodes,
		NodeTags:          nodeTags,
		Show:              subscribe.Show,
		Sell:              subscribe.Sell,
		Sort:              int64(subscribe.Sort),
		DeductionRatio:    deductionRatio,
		AllowDeduction:    subscribe.AllowDeduction,
		NodeGroupIds:      tool.Int64SliceToStringSlice(subscribe.NodeGroupIds),
		NodeGroupId:       nodeGroupID,
		ResetCycle:        resetCycle,
		RenewalReset:      subscribe.RenewalReset,
		ShowOriginalPrice: subscribe.ShowOriginalPrice,
		PriceOptions:      r.convertToPublicPriceOptions(ctx, subscribe.ID),
		CreatedAt:         subscribe.CreatedAt.UnixMilli(),
		UpdatedAt:         subscribe.UpdatedAt.UnixMilli(),
	}
}

func (r *publicOrderRepo) convertToPublicPriceOptions(ctx context.Context, subscribeID int64) []publicBiz.SubscribePriceOption {
	items, err := r.data.db.ProxySubscribePriceOption.Query().
		Where(
			proxysubscribepriceoption.SubscribeIDEQ(subscribeID),
			proxysubscribepriceoption.OptionTypeEQ("duration"),
			proxysubscribepriceoption.ShowEQ(true),
			proxysubscribepriceoption.SellEQ(true),
		).
		Order(ent.Desc(proxysubscribepriceoption.FieldSort), ent.Asc(proxysubscribepriceoption.FieldID)).
		All(ctx)
	if err != nil {
		return []publicBiz.SubscribePriceOption{}
	}
	result := make([]publicBiz.SubscribePriceOption, 0, len(items))
	for _, item := range items {
		result = append(result, publicBiz.SubscribePriceOption{
			ID:            item.ID,
			SubscribeID:   item.SubscribeID,
			Code:          item.Code,
			Type:          item.OptionType,
			Name:          item.Name,
			DurationUnit:  item.DurationUnit,
			DurationValue: item.DurationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     int64(item.Inventory),
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          int64(item.Sort),
			CreatedAt:     item.CreatedAt.UnixMilli(),
			UpdatedAt:     item.UpdatedAt.UnixMilli(),
		})
	}
	return result
}

// 订单类型和状态文本的辅助函数
func (r *publicOrderRepo) getOrderTypeText(orderType int8) string {
	switch orderType {
	case 1:
		return "订阅"
	case 2:
		return "续费"
	case 3:
		return "重置流量"
	case 4:
		return "充值"
	default:
		return "未知"
	}
}

func (r *publicOrderRepo) getOrderStatusText(status int8) string {
	switch status {
	case 1:
		return "待付款"
	case 2:
		return "已付款"
	case 3:
		return "已关闭"
	case 4:
		return "失败"
	case 5:
		return "已完成"
	default:
		return "未知"
	}
}

// enqueueDeferCloseOrderTask 将任务加入队列以在15分钟后自动关闭未支付订单
func (r *publicOrderRepo) enqueueDeferCloseOrderTask(ctx context.Context, userID int, orderNo string) {
	// 创建任务负载
	payload := queueTypes.DeferCloseOrderPayload{
		OrderNo: orderNo,
	}

	// 序列化负载为JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.logger.Errorf("[EnqueueDeferCloseOrderTask] Marshal payload failed: %v, payload: %+v", err, payload)
		return // 如果任务入队失败，不要让订单创建失败
	}

	// 创建带3次重试的asynq任务
	task := asynq.NewTask(queueTypes.DeferCloseOrder, payloadBytes, asynq.MaxRetry(3))

	// 将任务入队，在CloseOrderTimeMinutes（15分钟）后处理
	taskInfo, err := r.data.queue.Enqueue(task, asynq.ProcessIn(CloseOrderTimeMinutes*time.Minute))
	if err != nil {
		r.logger.Errorf("[EnqueueDeferCloseOrderTask] Enqueue task failed: %v, task: %+v", err, task)
		return // 如果任务入队失败，不要让订单创建失败
	}

	r.logger.Infof("[EnqueueDeferCloseOrderTask] Enqueue task success, TaskID: %s, OrderNo: %s, ProcessAt: %v",
		taskInfo.ID, orderNo, taskInfo.NextProcessAt)
}

// calculateCoupon 计算优惠券折扣金额
func (r *publicOrderRepo) calculateCoupon(amount int64, couponInfo *ent.ProxyCoupon) int64 {
	if couponInfo.Type == 1 {
		// 百分比折扣
		return int64(float64(amount) * (float64(couponInfo.Discount) / float64(100)))
	} else {
		// 固定金额折扣
		if couponInfo.Discount < amount {
			return couponInfo.Discount
		}
		return amount
	}
}

// calculateFee 计算支付手续费
func (r *publicOrderRepo) calculateFee(amount int64, config *ent.ProxyPayment) int64 {
	var fee float64
	switch config.FeeMode {
	case 0:
		// 无手续费
		return 0
	case 1:
		// 百分比手续费 - config.FeePercent是int64非指针
		fee = float64(amount) * (float64(config.FeePercent) / float64(100))
	case 2:
		// 固定手续费 - config.FeeAmount是int64非指针
		if amount > 0 {
			fee = float64(config.FeeAmount)
		}
	case 3:
		// 百分比+固定手续费 - 两者都是int64非指针
		fee = float64(amount)*(float64(config.FeePercent)/float64(100)) + float64(config.FeeAmount)
	}
	return int64(fee)
}

// getDiscount 根据数量获取折扣
func (r *publicOrderRepo) getDiscount(discounts []SubscribeDiscount, inputMonths int64) float64 {
	var finalDiscount int64 = 100

	for _, discount := range discounts {
		if inputMonths >= discount.Quantity && discount.Discount < finalDiscount {
			finalDiscount = discount.Discount
		}
	}

	return float64(finalDiscount) / float64(100)
}

// buildNotifyURL 构建支付通知回调URL
// 优先级: payment.Domain > 上下文主机 > config.Site.Host
func (r *publicOrderRepo) buildNotifyURL(ctx context.Context, payment *ent.ProxyPayment) string {
	// 第一优先级：使用支付配置的域名
	if payment.Domain != "" {
		return payment.Domain + "/v1/notify/" + payment.Platform + "/" + payment.Token
	}

	// 第二优先级：尝试从请求上下文获取主机
	var host string
	if contextHost, ok := ctx.Value("requestHost").(string); ok && contextHost != "" {
		host = contextHost
		r.logger.Infof("[buildNotifyURL] Using context host: %s", host)
	} else if siteValues, err := loadSystemConfigMap(ctx, r.data.db, "site"); err == nil && systemConfigString(siteValues, "Host", "host") != "" {
		host = systemConfigString(siteValues, "Host", "host")
		r.logger.Infof("[buildNotifyURL] Using database host: %s", host)
	} else if r.config != nil && r.config.Site != nil && r.config.Site.Host != "" {
		// 第三优先级：使用配置站点主机作为后备
		host = r.config.Site.Host
		r.logger.Infof("[buildNotifyURL] Using config host: %s", host)
	}

	if host != "" {
		return "https://" + host + "/v1/notify/" + payment.Platform + "/" + payment.Token
	}

	r.logger.Warnf("[buildNotifyURL] No domain configured for payment: %d", payment.ID)
	return ""
}

// generateAlipayF2FPayment 生成支付宝当面付二维码
func (r *publicOrderRepo) generateAlipayF2FPayment(ctx context.Context, order *ent.ProxyOrder, payment *ent.ProxyPayment, notifyURL string) (string, string) {
	// 解析支付宝当面付配置
	var config struct {
		AppID       string `json:"app_id"`
		PrivateKey  string `json:"private_key"`
		PublicKey   string `json:"public_key"`
		InvoiceName string `json:"invoice_name"`
		Sandbox     bool   `json:"sandbox"`
	}

	if err := json.Unmarshal([]byte(payment.Config), &config); err != nil {
		r.logger.Errorf("[generateAlipayF2FPayment] Unmarshal config failed: %v", err)
		return "", ""
	}

	// 初始化支付宝客户端
	client := alipay.NewClient(alipay.Config{
		AppId:       config.AppID,
		PrivateKey:  config.PrivateKey,
		PublicKey:   config.PublicKey,
		InvoiceName: config.InvoiceName,
		NotifyURL:   notifyURL,
		Sandbox:     config.Sandbox,
	})

	if client == nil {
		r.logger.Errorf("[generateAlipayF2FPayment] Failed to create Alipay client")
		return "", ""
	}

	// 使用当前汇率将订单金额转换为CNY
	amountFloat, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount))
	if err != nil {
		r.logger.Errorf("[generateAlipayF2FPayment] queryExchangeRate failed: %v, orderNo: %s", err, order.OrderNo)
		return "", ""
	}
	convertedAmount := int64(amountFloat * 100) // 转换为分 用于API

	// 创建预付款交易并生成二维码
	qrCode, err := client.PreCreateTrade(ctx, alipay.Order{
		OrderNo: order.OrderNo,
		Amount:  convertedAmount,
	})

	if err != nil {
		r.logger.Errorf("[generateAlipayF2FPayment] PreCreateTrade failed: %v, orderNo: %s", err, order.OrderNo)
		return "", ""
	}

	// 返回空URL和二维码
	return "", qrCode
}

// generateEPayPayment 生成EPay支付URL
func (r *publicOrderRepo) generateEPayPayment(ctx context.Context, order *ent.ProxyOrder, payment *ent.ProxyPayment, notifyURL string) (string, string) {
	// 解析EPay配置
	var config struct {
		PID  string `json:"pid"`
		URL  string `json:"url"`
		Key  string `json:"key"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal([]byte(payment.Config), &config); err != nil {
		r.logger.Errorf("[generateEPayPayment] Unmarshal config failed: %v", err)
		return "", ""
	}

	// 初始化EPay客户端
	client := epay.NewClient(config.PID, config.URL, config.Key, config.Type)

	// 使用当前汇率将订单金额转换为CNY
	amountFloat, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount))
	if err != nil {
		r.logger.Errorf("[generateEPayPayment] queryExchangeRate failed: %v, orderNo: %s", err, order.OrderNo)
		return "", ""
	}

	// 从配置生成站点名称或使用默认值
	siteName := "Order Payment"
	if siteValues, err := loadSystemConfigMap(ctx, r.data.db, "site"); err == nil {
		if value := systemConfigString(siteValues, "SiteName", "site_name"); value != "" {
			siteName = value
		}
	}
	if siteName == "Order Payment" && r.config != nil && r.config.Site != nil && r.config.Site.SiteName != "" {
		siteName = r.config.Site.SiteName
	}

	// 创建支付URL
	payURL := client.CreatePayUrl(epay.Order{
		Name:      siteName,
		Amount:    amountFloat,
		OrderNo:   order.OrderNo,
		SignType:  "MD5",
		NotifyUrl: notifyURL,
		ReturnUrl: "", // 如需要可从前端设置
		Type:      config.Type,
	})

	// 返回支付URL和空二维码
	return payURL, ""
}

// generateStripePayment 生成Stripe支付客户端密钥
func (r *publicOrderRepo) generateStripePayment(ctx context.Context, order *ent.ProxyOrder, payment *ent.ProxyPayment) (string, string) {
	// 解析Stripe配置
	var config struct {
		PublicKey     string `json:"public_key"`
		SecretKey     string `json:"secret_key"`
		WebhookSecret string `json:"webhook_secret"`
		Payment       string `json:"payment"` // 支付方式：card/alipay/wechat_pay
	}

	if err := json.Unmarshal([]byte(payment.Config), &config); err != nil {
		r.logger.Errorf("[generateStripePayment] Unmarshal config failed: %v", err)
		return "", ""
	}

	// 初始化Stripe客户端
	client := stripe.NewClient(stripe.Config{
		PublicKey:     config.PublicKey,
		SecretKey:     config.SecretKey,
		WebhookSecret: config.WebhookSecret,
	})

	// 使用当前汇率将订单金额转换为CNY
	amountFloat, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount))
	if err != nil {
		r.logger.Errorf("[generateStripePayment] queryExchangeRate failed: %v, orderNo: %s", err, order.OrderNo)
		return "", ""
	}
	convertedAmount := int64(amountFloat * 100) // 转换为分 用于Stripe API

	// 创建Stripe支付单
	result, err := client.CreatePaymentSheet(
		&stripe.Order{
			OrderNo:   order.OrderNo,
			Subscribe: fmt.Sprintf("%d", order.SubscribeID),
			Amount:    convertedAmount,
			Currency:  "cny",
			Payment:   config.Payment,
		},
		&stripe.User{
			UserId: order.UserID,
			Email:  "", // 待办：如果可用，获取用户邮箱
		},
	)

	if err != nil {
		r.logger.Errorf("[generateStripePayment] CreatePaymentSheet failed: %v, orderNo: %s", err, order.OrderNo)
		return "", ""
	}

	// 使用Stripe支付意图ID更新订单交易号
	// 这对支付验证很重要
	err = r.data.db.ProxyOrder.UpdateOneID(order.ID).
		SetTradeNo(result.TradeNo).
		Exec(ctx)
	if err != nil {
		r.logger.Errorf("[generateStripePayment] Update trade_no failed: %v, orderNo: %s", err, order.OrderNo)
	}

	// 返回客户端密钥作为支付URL（前端将使用Stripe SDK处理）
	// 格式：publicKey|clientSecret|ephemeralKey|customer
	paymentInfo := fmt.Sprintf("%s|%s|%s|%s", result.PublishableKey, result.ClientSecret, result.EphemeralKey, result.Customer)
	return paymentInfo, ""
}

// generateCryptoSaaSPayment 生成CryptoSaaS支付URL
func (r *publicOrderRepo) generateCryptoSaaSPayment(ctx context.Context, order *ent.ProxyOrder, payment *ent.ProxyPayment, notifyURL string) (string, string) {
	// 解析CryptoSaaS配置
	var config struct {
		Endpoint  string `json:"endpoint"`
		AccountID string `json:"account_id"`
		SecretKey string `json:"secret_key"`
		Type      string `json:"type"`
	}

	if err := json.Unmarshal([]byte(payment.Config), &config); err != nil {
		r.logger.Errorf("[generateCryptoSaaSPayment] Unmarshal config failed: %v", err)
		return "", ""
	}

	// CryptoSaaS使用相同的EPay接口
	client := epay.NewClient(config.AccountID, config.Endpoint, config.SecretKey, config.Type)

	// 使用当前汇率将订单金额转换为CNY
	amountFloat, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount))
	if err != nil {
		r.logger.Errorf("[generateCryptoSaaSPayment] queryExchangeRate failed: %v, orderNo: %s", err, order.OrderNo)
		return "", ""
	}

	// 从配置生成站点名称或使用默认值
	siteName := "Order Payment"
	if siteValues, err := loadSystemConfigMap(ctx, r.data.db, "site"); err == nil {
		if value := systemConfigString(siteValues, "SiteName", "site_name"); value != "" {
			siteName = value
		}
	}
	if siteName == "Order Payment" && r.config != nil && r.config.Site != nil && r.config.Site.SiteName != "" {
		siteName = r.config.Site.SiteName
	}

	// 创建支付URL
	payURL := client.CreatePayUrl(epay.Order{
		Name:      siteName,
		Amount:    amountFloat,
		OrderNo:   order.OrderNo,
		SignType:  "MD5",
		NotifyUrl: notifyURL,
		ReturnUrl: "", // 如需要可从前端设置
		Type:      config.Type,
	})

	// 返回支付URL和空二维码
	return payURL, ""
}

// processBalancePayment 处理余额支付
// ⚠️ 重要：赠金已在订单创建时扣除并记录在 order.GiftAmount 字段中
// 这里只需从用户 Balance 扣除 order.Amount 即可
func (r *publicOrderRepo) processBalancePayment(ctx context.Context, order *ent.ProxyOrder) error {
	userID := order.UserID

	// 处理零金额订单（赠金已全额覆盖）
	if order.Amount == 0 {
		r.logger.Infof("[processBalancePayment] Zero amount order (fully covered by gift), mark as paid directly. OrderNo: %s, GiftAmount: %d",
			order.OrderNo, order.GiftAmount)
		// 更新订单状态为已支付（gift_amount 已在创建时设置）
		err := r.data.db.ProxyOrder.UpdateOneID(order.ID).SetStatus(2).Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processBalancePayment] Update order status failed: %v, orderNo: %s", err, order.OrderNo)
			return responsecode.NewKratosError(responsecode.ErrOrderPaymentFailed)
		}
		// 将立即激活任务入队
		return r.enqueueActivateOrderTask(ctx, int(userID), order.OrderNo)
	}

	// ✅ 在事务中执行所有操作（包括查询用户），确保并发安全
	err := r.data.db.TX(ctx, func(tx *ent.Tx) error {
		// ✅ 在事务内查询用户（自动加行锁，防止并发问题）
		user, err := tx.ProxyUser.Query().
			Where(proxyuser.IDEQ(userID)).
			Only(ctx)
		if err != nil {
			r.logger.Errorf("[processBalancePayment] Query user failed: %v, userID: %d", err, userID)
			return responsecode.NewKratosError(responsecode.ErrUserNotFound)
		}

		// 安全获取余额（赠金已在订单创建时扣除，这里不需要再扣）
		userBalance := int64(0)
		if user.Balance != nil {
			userBalance = *user.Balance
		}

		// 检查用户余额是否足够支付 order.Amount
		// 注意：order.Amount 已经扣除了赠金，只需要检查 Balance 是否足够
		if userBalance < order.Amount {
			r.logger.Errorf("[processBalancePayment] Insufficient balance. Required: %d, Available: %d, userID: %d, orderNo: %s",
				order.Amount, userBalance, userID, order.OrderNo)
			return responsecode.NewKratosError(responsecode.ErrInsufficientBalance)
		}

		// 从用户余额扣除订单金额
		newBalance := userBalance - order.Amount

		// 更新用户余额（不更新赠金，因为已在创建时扣除）
		err = tx.ProxyUser.UpdateOneID(user.ID).
			SetBalance(newBalance).
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processBalancePayment] Update user balance failed: %v, userID: %d", err, userID)
			return err
		}

		// 创建余额日志
		balanceLog := map[string]interface{}{
			"type":      323, // BalanceTypePayment = 323
			"amount":    order.Amount,
			"order_no":  order.OrderNo,
			"balance":   newBalance,
			"timestamp": time.Now().UnixMilli(),
		}
		balanceLogJSON, _ := json.Marshal(balanceLog)

		_, err = tx.ProxySystemLog.Create().
			SetType(32). // TypeBalance = 32
			SetDate(time.Now().Format(time.DateOnly)).
			SetObjectID(userID).
			SetContent(string(balanceLogJSON)).
			Save(ctx)
		if err != nil {
			r.logger.Errorf("[processBalancePayment] Create balance log failed: %v, userID: %d", err, userID)
			return err
		}

		// 更新订单状态为已支付（不更新 gift_amount，已在创建时设置）
		err = tx.ProxyOrder.UpdateOneID(order.ID).
			SetStatus(2). // 已支付
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processBalancePayment] Update order status failed: %v, orderNo: %s", err, order.OrderNo)
			return err
		}

		return nil
	})

	if err != nil {
		r.logger.Errorf("[processBalancePayment] Transaction failed: %v, orderNo: %s", err, order.OrderNo)
		return responsecode.NewKratosError(responsecode.ErrOrderPaymentFailed)
	}
	// 将立即激活任务入队 (在事务外)
	err = r.enqueueActivateOrderTask(ctx, int(userID), order.OrderNo)
	if err != nil {
		r.logger.Warnf("[processBalancePayment] Enqueue activation task failed: %v, orderNo: %s", err, order.OrderNo)
		// 如果激活任务入队失败，不要让支付失败
	}

	r.logger.Infof("[processBalancePayment] Balance payment completed successfully. OrderNo: %s, Amount: %d, GiftAmount: %d (already deducted during order creation)",
		order.OrderNo, order.Amount, order.GiftAmount)

	return nil
}

// enqueueActivateOrderTask 将立即激活订单任务加入队列
func (r *publicOrderRepo) enqueueActivateOrderTask(ctx context.Context, userID int, orderNo string) error {
	payload := queueTypes.ForthwithActivateOrderPayload{
		OrderNo: orderNo,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.logger.Errorf("[enqueueActivateOrderTask] Marshal payload failed: %v, orderNo: %s", err, orderNo)
		return err
	}

	task := asynq.NewTask(queueTypes.ForthwithActivateOrder, payloadBytes)
	_, err = r.data.queue.Enqueue(task)
	if err != nil {
		r.logger.Errorf("[enqueueActivateOrderTask] Enqueue task failed: %v, orderNo: %s", err, orderNo)
		return err
	}

	r.logger.Infof("[enqueueActivateOrderTask] Enqueue activation task success, OrderNo: %s", orderNo)
	return nil
}

// queryExchangeRate 将订单金额从系统货币转换为目标货币
// 获取当前汇率并在需要时执行货币转换
func (r *publicOrderRepo) queryExchangeRate(ctx context.Context, targetCurrency string, amountInCents int) (float64, error) {
	// 将分转换为十进制金额
	amount := float64(amountInCents) / float64(100)

	// 从 proxy_system 获取货币配置 表
	configMap, err := loadSystemConfigMap(ctx, r.data.db, "currency")
	if err != nil {
		r.logger.Errorf("[queryExchangeRate] Query currency config failed: %v", err)
		return 0, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// 提取货币单位和访问密钥
	currencyUnit := systemConfigString(configMap, "CurrencyUnit", "Currency", "default_currency")
	accessKey := systemConfigString(configMap, "AccessKey", "access_key")

	// 如果未配置汇率API密钥，跳过转换
	if accessKey == "" {
		r.logger.Warnf("[queryExchangeRate] AccessKey not configured, skip conversion.")
		return amount, nil
	}

	// 如果货币相同，跳过转换
	if currencyUnit == targetCurrency {
		return amount, nil
	}

	// 调用汇率API 转换货币
	result, err := exchangeRate.GetExchangeRete(currencyUnit, targetCurrency, accessKey, 1)
	if err != nil {
		r.logger.Errorf("[queryExchangeRate] Get exchange rate failed: %v, from: %s, to: %s", err, currencyUnit, targetCurrency)
		return 0, responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	// 将汇率应用于金额
	convertedAmount := result * amount
	r.logger.Infof("[queryExchangeRate] Currency conversion: %s %.2f -> %s %.2f (rate: %.4f)",
		currencyUnit, amount, targetCurrency, convertedAmount, result)

	return convertedAmount, nil
}
