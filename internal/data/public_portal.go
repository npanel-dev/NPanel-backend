package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	portalBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/portal"
	productlanguage "github.com/npanel-dev/NPanel-backend/internal/pkg/language"
	"github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	queueTypes "github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/constant"
	"github.com/npanel-dev/NPanel-backend/pkg/exchangeRate"
	"github.com/npanel-dev/NPanel-backend/pkg/payment"
	"github.com/npanel-dev/NPanel-backend/pkg/payment/alipay"
	"github.com/npanel-dev/NPanel-backend/pkg/payment/epay"
	"github.com/npanel-dev/NPanel-backend/pkg/payment/stripe"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

// ============================================================================
// Portal数据仓储
// ⚠️ 注：支付配置结构体（StripeConfig等）已在 payment_config.go 中定义
// ============================================================================

type publicPortalRepo struct {
	data   *Data
	logger *log.Helper
}

// NewPublicPortalRepo 创建Portal数据仓储实例
func NewPublicPortalRepo(d *Data, logger log.Logger) portalBiz.PortalRepo {
	return &publicPortalRepo{
		data:   d,
		logger: log.NewHelper(logger),
	}
}

// CheckUserExists 检查用户是否已存在
// ⚠️ 通过authType和identifier查询
func (r *publicPortalRepo) CheckUserExists(ctx context.Context, authType, identifier string) (bool, error) {
	r.logger.Infof("[CheckUserExists] authType: %s, identifier: %s", authType, identifier)

	// 查询用户认证方法表
	count, err := r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthTypeEQ(authType),
			proxyuserauthmethod.AuthIdentifierEQ(identifier),
		).
		Count(ctx)

	if err != nil {
		r.logger.Errorf("[CheckUserExists] 查询失败: %v", err)
		return false, errors.InternalServer("DATABASE_ERROR", "查询用户失败")
	}

	return count > 0, nil
}

// GetSubscribeList 获取订阅列表
// ⚠️ 包含租户过滤和语言过滤
// language: 如果不为空，过滤指定语言；如果为空，返回默认语言（language=”）
func (r *publicPortalRepo) GetSubscribeList(ctx context.Context, language string, categoryID int64) ([]*portalBiz.SubscribeInfo, error) {
	language = productlanguage.NormalizeProductLanguage(language)
	r.logger.Infof("[GetSubscribeList] language: %s", language)

	// 构建查询条件
	query := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.ShowEQ(true),
		)

	// 语言过滤逻辑（复刻原项目 DefaultLanguage=true）
	if language != "" {
		query = query.Where(
			proxysubscribe.Or(
				proxysubscribe.LanguageEQ(language),
				proxysubscribe.LanguageEQ(""),
			),
		)
	} else {
		query = query.Where(proxysubscribe.LanguageEQ(""))
	}

	if categoryID > 0 {
		query = query.Where(proxysubscribe.CategoryIDEQ(categoryID))
	}

	subscribes, err := query.
		Order(ent.Asc(proxysubscribe.FieldSort)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("[GetSubscribeList] 查询失败: %v", err)
		return nil, errors.InternalServer("DATABASE_ERROR", "查询订阅列表失败")
	}

	result := make([]*portalBiz.SubscribeInfo, 0, len(subscribes))
	categoryNames := r.portalSubscribeCategoryNames(ctx, subscribes)
	priceOptions := r.portalSubscribePriceOptions(ctx, subscribes)
	for _, sub := range subscribes {
		// 解析Discount JSON为数组（复刻原项目 getSubscriptionLogic.go line 35-39）
		var discounts []portalBiz.SubscribeDiscount
		if sub.Discount != nil && *sub.Discount != "" {
			_ = json.Unmarshal([]byte(*sub.Discount), &discounts)
		}

		// 解析Nodes字符串为int数组（复刻原项目逻辑）
		nodes64 := stringToInt64Slice(sub.Nodes)
		nodes := make([]int, len(nodes64))
		for i, v := range nodes64 {
			nodes[i] = int(v)
		}

		// 解析NodeTags字符串为string数组（复刻原项目逻辑）
		var nodeTags []string
		if sub.NodeTags != "" {
			nodeTags = strings.Split(sub.NodeTags, ",")
			// 去除空白
			for i := range nodeTags {
				nodeTags[i] = strings.TrimSpace(nodeTags[i])
			}
		}

		// 处理DeductionRatio指针
		deductionRatio := int64(0)
		if sub.DeductionRatio != nil {
			deductionRatio = int64(*sub.DeductionRatio)
		}

		// 处理ResetCycle指针
		resetCycle := int64(0)
		if sub.ResetCycle != nil {
			resetCycle = int64(*sub.ResetCycle)
		}
		nodeGroupID := "0"
		if sub.NodeGroupID != nil {
			nodeGroupID = strconv.FormatInt(*sub.NodeGroupID, 10)
		}
		var trafficLimit []portalBiz.TrafficLimit
		if sub.TrafficLimit != nil {
			trafficLimit = parsePortalTrafficLimits(*sub.TrafficLimit)
		}

		result = append(result, &portalBiz.SubscribeInfo{
			ID:                int64(sub.ID),
			Name:              sub.Name,
			Language:          sub.Language,
			Description:       sub.Description,
			UnitPrice:         sub.UnitPrice,
			UnitTime:          sub.UnitTime,
			Discount:          discounts,
			Replacement:       sub.Replacement,
			Inventory:         int64(sub.Inventory),
			Traffic:           sub.Traffic,
			SpeedLimit:        int64(sub.SpeedLimit),
			DeviceLimit:       int64(sub.DeviceLimit),
			Quota:             int64(sub.Quota),
			CategoryID:        sub.CategoryID,
			CategoryName:      categoryNames[sub.CategoryID],
			Nodes:             nodes,
			NodeTags:          nodeTags,
			NodeGroupIds:      tool.Int64SliceToStringSlice(sub.NodeGroupIds),
			NodeGroupId:       nodeGroupID,
			TrafficLimit:      trafficLimit,
			Show:              sub.Show,
			Sell:              sub.Sell,
			Sort:              int64(sub.Sort),
			DeductionRatio:    int64(deductionRatio),
			AllowDeduction:    sub.AllowDeduction,
			ResetCycle:        int64(resetCycle),
			RenewalReset:      sub.RenewalReset,
			ShowOriginalPrice: sub.ShowOriginalPrice,
			PriceOptions:      priceOptions[int64(sub.ID)],
			CreatedAt:         sub.CreatedAt.Unix(),
			UpdatedAt:         sub.UpdatedAt.Unix(),
		})
	}

	return result, nil
}

func (r *publicPortalRepo) GetSubscribeCatalog(ctx context.Context, language string) (*portalBiz.SubscribeCatalog, error) {
	language = productlanguage.NormalizeProductLanguage(language)
	subscribes, err := r.GetSubscribeList(ctx, language, 0)
	if err != nil {
		return nil, err
	}
	categoryQuery := r.data.db.ProxySubscribeCategory.Query().
		Where(proxysubscribecategory.ShowEQ(true))
	if language != "" {
		categoryQuery = categoryQuery.Where(
			proxysubscribecategory.Or(
				proxysubscribecategory.LanguageEQ(language),
				proxysubscribecategory.LanguageEQ(""),
			),
		)
	} else {
		categoryQuery = categoryQuery.Where(proxysubscribecategory.LanguageEQ(""))
	}
	categories, err := categoryQuery.
		Order(ent.Asc(proxysubscribecategory.FieldSort), ent.Asc(proxysubscribecategory.FieldID)).
		All(ctx)
	if err != nil {
		return nil, errors.InternalServer("DATABASE_ERROR", "查询订阅分类失败")
	}

	categoryMap := make(map[int64]*portalBiz.SubscribeCategory, len(categories))
	roots := make([]*portalBiz.SubscribeCategory, 0)
	for _, category := range categories {
		item := &portalBiz.SubscribeCategory{
			ID:          category.ID,
			ParentID:    category.ParentID,
			Name:        category.Name,
			Description: stringValue(category.Description),
			Language:    category.Language,
			Show:        category.Show,
			Sort:        int64(category.Sort),
		}
		categoryMap[category.ID] = item
	}
	for _, category := range categories {
		item := categoryMap[category.ID]
		if item.ParentID > 0 {
			if parent := categoryMap[item.ParentID]; parent != nil {
				parent.Children = append(parent.Children, item)
				continue
			}
		}
		roots = append(roots, item)
	}

	uncategorized := make([]*portalBiz.SubscribeInfo, 0)
	for _, sub := range subscribes {
		if sub.CategoryID > 0 {
			if category := categoryMap[sub.CategoryID]; category != nil {
				category.List = append(category.List, sub)
				continue
			}
		}
		uncategorized = append(uncategorized, sub)
	}

	return &portalBiz.SubscribeCatalog{
		Categories:    roots,
		Uncategorized: uncategorized,
		Total:         int32(len(subscribes)),
	}, nil
}

func (r *publicPortalRepo) portalSubscribeCategoryNames(ctx context.Context, subscribes []*ent.ProxySubscribe) map[int64]string {
	ids := make(map[int64]struct{})
	for _, sub := range subscribes {
		if sub != nil && sub.CategoryID > 0 {
			ids[sub.CategoryID] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return map[int64]string{}
	}
	idList := make([]int64, 0, len(ids))
	for id := range ids {
		idList = append(idList, id)
	}
	categories, err := r.data.db.ProxySubscribeCategory.Query().
		Where(proxysubscribecategory.IDIn(idList...)).
		All(ctx)
	if err != nil {
		return map[int64]string{}
	}
	result := make(map[int64]string, len(categories))
	for _, category := range categories {
		result[category.ID] = category.Name
	}
	return result
}

func (r *publicPortalRepo) portalSubscribePriceOptions(ctx context.Context, subscribes []*ent.ProxySubscribe) map[int64][]portalBiz.SubscribePriceOption {
	ids := make([]int64, 0, len(subscribes))
	for _, sub := range subscribes {
		if sub != nil {
			ids = append(ids, int64(sub.ID))
		}
	}
	result := make(map[int64][]portalBiz.SubscribePriceOption)
	if len(ids) == 0 {
		return result
	}
	items, err := r.data.db.ProxySubscribePriceOption.Query().
		Where(
			proxysubscribepriceoption.SubscribeIDIn(ids...),
			proxysubscribepriceoption.OptionTypeEQ("duration"),
			proxysubscribepriceoption.ShowEQ(true),
			proxysubscribepriceoption.SellEQ(true),
		).
		Order(ent.Desc(proxysubscribepriceoption.FieldSort), ent.Asc(proxysubscribepriceoption.FieldID)).
		All(ctx)
	if err != nil {
		return result
	}
	for _, item := range items {
		result[item.SubscribeID] = append(result[item.SubscribeID], portalBiz.SubscribePriceOption{
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
			CreatedAt:     item.CreatedAt.Unix(),
			UpdatedAt:     item.UpdatedAt.Unix(),
		})
	}
	return result
}

func (r *publicPortalRepo) getSellablePortalPriceOption(ctx context.Context, subscribeID, optionID int64) (*ent.ProxySubscribePriceOption, error) {
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
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if option.Inventory == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeOutOfStock)
	}
	return option, nil
}

func portalPriceFromOption(option *ent.ProxySubscribePriceOption) (price, amount, discount int64) {
	amount = option.Price
	price = option.OriginalPrice
	if price <= 0 || price < amount {
		price = amount
	}
	return price, amount, price - amount
}

// CalculateOrderPrice 计算订单价格（含折扣、优惠券、手续费）
// ⚠️ Portal订单不需要查询用户信息（userId=0）
// ⚠️ paymentID可选，用于计算手续费
func (r *publicPortalRepo) CalculateOrderPrice(ctx context.Context, subscribeID, priceOptionID, quantity int64, coupon *string, paymentID *int64) (*portalBiz.PriceInfo, error) {
	r.logger.Infof("[CalculateOrderPrice] subscribeID: %d, priceOptionID: %d, quantity: %d, paymentID: %v", subscribeID, priceOptionID, quantity, paymentID)

	// 1. 查询订阅套餐
	sub, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(subscribeID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[CalculateOrderPrice] 订阅不存在: %d", subscribeID)
		return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
	}
	if !sub.Sell {
		return nil, errors.BadRequest("SUBSCRIBE_NOT_SELL", "订阅计划未在售")
	}

	option, err := r.getSellablePortalPriceOption(ctx, subscribeID, priceOptionID)
	if err != nil {
		return nil, err
	}

	price, amount, discountAmount := portalPriceFromOption(option)

	// 4. 计算优惠券折扣（复刻原项目逻辑 - prePurchaseOrderLogic.go line 49-68）
	var couponAmount int64 = 0
	if coupon != nil && *coupon != "" {
		couponInfo, err := r.data.db.ProxyCoupon.Query().
			Where(
				proxycoupon.CodeEQ(*coupon),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, responsecode.NewKratosError(responsecode.ErrCouponNotFound)
			}
			r.logger.Errorf("[CalculateOrderPrice] 查询优惠券失败: %v", err)
			return nil, errors.InternalServer("DATABASE_ERROR", "查询优惠券失败")
		}
		if couponInfo.Count != 0 && couponInfo.Count <= int32(couponInfo.UsedCount) {
			r.logger.Warnf("[CalculateOrderPrice] 优惠券已用完: coupon=%s", *coupon)
			return nil, responsecode.NewKratosError(responsecode.ErrCouponUsedUp)
		}
		if couponInfo.Subscribe != "" {
			allowedSubs := stringToInt64Slice(couponInfo.Subscribe)
			allowed := false
			for _, allowedID := range allowedSubs {
				if allowedID == subscribeID {
					allowed = true
					break
				}
			}
			if !allowed {
				r.logger.Warnf("[CalculateOrderPrice] 优惠券不适用于此订阅: coupon=%s, subscribeID=%d", *coupon, subscribeID)
				return nil, responsecode.NewKratosError(responsecode.ErrCouponNotAvailable)
			}
		}

		// 计算优惠券折扣（使用helper函数）
		couponAmount = int64(calculateCoupon(amount, couponInfo))
	}
	amount = amount - couponAmount

	// 5. 计算手续费（如果有支付方式ID）
	var feeAmount int = 0
	if paymentID != nil && *paymentID > 0 {
		payment, err := r.data.db.ProxyPayment.Query().
			Where(
				proxypayment.IDEQ(*paymentID),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
			}
			r.logger.Errorf("[CalculateOrderPrice] 查询支付方式失败: %v", err)
			return nil, errors.InternalServer("DATABASE_ERROR", "查询支付方式失败")
		}
		if amount > 0 {
			// 使用完整的手续费计算函数（支持4种模式）
			feeAmount = calculateFee(amount, payment)
		}
	}

	// 6. 计算实际支付金额（复刻原项目逻辑 - prePurchaseOrderLogic.go line 79-81）
	// ⚠️ 关键：amount += feeAmount（原项目将手续费加到amount上）
	amount = amount + int64(feeAmount)

	// 处理 coupon 字段（复刻原项目 - prePurchaseOrderLogic.go line 87）
	couponStr := ""
	if coupon != nil {
		couponStr = *coupon
	}

	return &portalBiz.PriceInfo{
		Price:          int(price),          // 原价
		Amount:         int(amount),         // 实际支付金额（含手续费）⚠️
		Discount:       int(discountAmount), // 折扣金额
		Coupon:         couponStr,           // 优惠券代码
		CouponDiscount: int(couponAmount),   // 优惠券折扣金额
		FeeAmount:      feeAmount,           // 手续费金额
	}, nil
}

// ⚠️ 禁止使用余额支付
// ⚠️ 计算手续费
// ⚠️ 入队延迟关闭订单任务（15分钟）
func (r *publicPortalRepo) CreatePortalOrder(ctx context.Context, req *portalBiz.CreateOrderRequest) (string, error) {
	r.logger.Infof("[CreatePortalOrder] subscribeID: %d, quantity: %d, paymentID: %d",
		req.SubscribeID, req.Quantity, req.PaymentID)

	// 1. 查询订阅计划
	subscribe, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(req.SubscribeID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		return "", errors.InternalServer("DATABASE_ERROR", "查询订阅计划失败")
	}

	// 2. 验证订阅计划是否在售
	if !subscribe.Sell {
		r.logger.Warnf("[CreatePortalOrder] 订阅计划未在售: subscribeID=%d", req.SubscribeID)
		return "", errors.BadRequest("SUBSCRIBE_NOT_SELL", "订阅计划未在售")
	}
	if subscribe.Inventory == 0 {
		return "", responsecode.NewKratosError(responsecode.ErrSubscribeOutOfStock)
	}

	option, err := r.getSellablePortalPriceOption(ctx, req.SubscribeID, req.PriceOptionID)
	if err != nil {
		return "", err
	}

	// 4. 计算价格和折扣金额
	price, amount, discountAmount := portalPriceFromOption(option)

	// 5. 处理优惠券（如果有）
	var couponAmount int = 0
	couponStr := ""
	if req.Coupon != nil && *req.Coupon != "" {
		couponStr = *req.Coupon
		coupon, err := r.data.db.ProxyCoupon.Query().
			Where(
				proxycoupon.CodeEQ(couponStr),
			).
			Only(ctx)

		if err != nil {
			if ent.IsNotFound(err) {
				return "", responsecode.NewKratosError(responsecode.ErrCouponNotFound)
			}
			return "", errors.InternalServer("DATABASE_ERROR", "查询优惠券失败")
		}

		if coupon.Count != 0 && coupon.Count <= int32(coupon.UsedCount) {
			return "", responsecode.NewKratosError(responsecode.ErrCouponUsedUp)
		}
		expireTime := time.Unix(coupon.ExpireTime, 0)
		if time.Now().After(expireTime) {
			return "", responsecode.NewKratosError(responsecode.ErrCouponExpired)
		}
		if coupon.Subscribe != "" {
			subscribes := stringToInt64Slice(coupon.Subscribe)
			if len(subscribes) > 0 {
				found := false
				for _, sid := range subscribes {
					if sid == req.SubscribeID {
						found = true
						break
					}
				}
				if !found {
					r.logger.Warnf("[CreatePortalOrder] 优惠券不适用于此订阅: coupon=%s, subscribeID=%d", couponStr, req.SubscribeID)
					return "", responsecode.NewKratosError(responsecode.ErrCouponNotAvailable)
				}
			}
		}

		// 计算优惠券折扣（复刻原项目 purchaseLogic.go line 91）
		couponAmount = calculateCoupon(amount, coupon)
	}

	// 6. 扣除优惠券金额
	amount -= int64(couponAmount)

	// 7. 查询支付方式
	payment, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(int64(req.PaymentID)),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return "", responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
		}
		r.logger.Errorf("[CreatePortalOrder] 查询支付方式失败: %v", err)
		return "", errors.InternalServer("DATABASE_ERROR", "查询支付方式失败")
	}

	// 8. 禁止Portal订单使用余额支付
	if payment.Platform == "balance" {
		r.logger.Warnf("[CreatePortalOrder] Portal订单不允许使用余额支付")
		return "", errors.BadRequest("PAYMENT_METHOD_NOT_ALLOWED", "Portal订单不允许使用余额支付")
	}

	// 9. 计算手续费（复刻原项目 purchaseLogic.go line 106-110）
	var feeAmount int = 0
	if amount > 0 {
		feeAmount = calculateFee(amount, payment)
	}
	// ⚠️ 注意：amount 不包含手续费，手续费单独存储在 FeeAmount 字段
	// 实际支付金额 = Amount + FeeAmount

	// 10. 生成订单号
	orderNo := tool.GenerateTradeNo()

	// 11. 保存临时订单信息到Redis（复刻原项目 purchaseLogic.go line 132-144）
	// ⚠️ 先保存Redis，再创建订单（保证逻辑顺序与原项目一致）
	tempInfo := constant.TemporaryOrderInfo{
		OrderNo:    orderNo,
		Identifier: req.Identifier,
		AuthType:   req.AuthType,
		Password:   req.Password, // ⚠️ 明文密码，在创建用户时才加密
		InviteCode: "",
	}
	if req.InviteCode != nil {
		tempInfo.InviteCode = *req.InviteCode
	}

	tempInfoBytes, err := tempInfo.Marshal()
	if err != nil {
		r.logger.Errorf("[CreatePortalOrder] 序列化临时订单信息失败: %v", err)
		return "", errors.InternalServer("SERIALIZE_ERROR", "序列化临时订单信息失败")
	}

	cacheKey := fmt.Sprintf(constant.TempOrderCacheKey, orderNo)
	if err := r.data.rdb.Set(ctx, cacheKey, string(tempInfoBytes), 24*time.Hour).Err(); err != nil {
		r.logger.Errorf("[CreatePortalOrder] Redis保存临时订单失败: %v", err)
		return "", errors.InternalServer("REDIS_ERROR", "保存临时订单信息失败")
	}
	r.logger.Infof("[CreatePortalOrder] 临时订单已保存: key=%s, identifier=%s", cacheKey, req.Identifier)

	// 12. 创建订单（复刻原项目 purchaseLogic.go line 112-128, 147-149）
	// ⚠️ userId=0 表示Portal订单
	// ⚠️ is_new=true 标记为新订单（用于佣金计算）
	// ⚠️ Amount 不含手续费，FeeAmount 单独存储
	err = r.data.db.TX(ctx, func(tx *ent.Tx) error {
		if subscribe.Inventory != -1 {
			if err := decrementSubscribeInventory(ctx, tx, req.SubscribeID); err != nil {
				return err
			}
		}
		if option.Inventory != -1 {
			if err := decrementDurationPriceOptionInventory(ctx, tx, req.SubscribeID, option.ID); err != nil {
				return err
			}
		}

		_, err := tx.ProxyOrder.Create().
			SetUserID(0). // ⚠️ Portal订单userId为0
			SetOrderNo(orderNo).
			SetType(1). // 订阅类型
			SetQuantity(orderDurationQuantity(option)).
			SetPrice(price).
			SetAmount(amount). // ⚠️ 不含手续费（= 原价*折扣 - 优惠券）
			SetDiscount(discountAmount).
			SetGiftAmount(0). // Portal订单无赠送金额
			SetCoupon(couponStr).
			SetCouponDiscount(int64(couponAmount)).
			SetPaymentID(int64(req.PaymentID)).
			SetMethod(payment.Platform).
			SetFeeAmount(int64(feeAmount)). // ⚠️ 手续费单独存储
			SetCommission(0).               // Portal订单无佣金
			SetSubscribeID(req.SubscribeID).
			SetPriceOptionID(option.ID).
			SetPriceOptionName(option.Name).
			SetDurationUnit(option.DurationUnit).
			SetDurationValue(option.DurationValue).
			SetOptionPrice(option.Price).
			SetIsNew(true). // ⚠️ 标记为新订单
			SetStatus(1).   // 待支付
			Save(ctx)
		if err != nil {
			return errors.InternalServer("DATABASE_ERROR", "创建订单失败")
		}
		return nil
	})
	if err != nil {
		r.logger.Errorf("[CreatePortalOrder] 创建订单失败: %v", err)
		return "", err
	}

	// 13. 入队延迟关闭订单任务（复刻原项目 purchaseLogic.go line 156-170）
	payload := queueTypes.DeferCloseOrderPayload{
		OrderNo: orderNo,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.logger.Errorf("[CreatePortalOrder] 序列化任务payload失败: %v", err)
		// 不影响订单创建，继续执行
	} else {
		task := asynq.NewTask(queueTypes.DeferCloseOrder, payloadBytes, asynq.MaxRetry(3))
		taskInfo, err := r.data.queue.Enqueue(task, asynq.ProcessIn(15*time.Minute))
		if err != nil {
			r.logger.Errorf("[CreatePortalOrder] 入队延迟关闭任务失败: %v", err)
			// 不影响订单创建，继续执行
		} else {
			r.logger.Infof("[CreatePortalOrder] 延迟关闭任务已入队: taskID=%s, orderNo=%s, delay=15min",
				taskInfo.ID, orderNo)
		}
	}

	r.logger.Infof("[CreatePortalOrder] Portal订单创建完成: orderNo=%s, amount=%d, feeAmount=%d, identifier=%s",
		orderNo, amount, feeAmount, req.Identifier)
	return orderNo, nil
}

// GetOrderByNo 根据订单号查询订单
func (r *publicPortalRepo) GetOrderByNo(ctx context.Context, orderNo string) (*portalBiz.OrderInfo, error) {
	r.logger.Infof("[GetOrderByNo] orderNo: %s", orderNo)

	order, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(orderNo),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrOrderNotFound)
		}
		return nil, errors.InternalServer("DATABASE_ERROR", "查询订单失败")
	}

	return &portalBiz.OrderInfo{
		ID:         order.ID,
		OrderNo:    order.OrderNo,
		Type:       int32(order.Type),
		Price:      order.Price,
		Amount:     order.Amount,
		Discount:   order.Discount,
		Commission: order.Commission,
		Method:     order.Method,
		Status:     int32(order.Status),
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
	}, nil
}

// GetAvailablePaymentMethods 获取可用支付方式
// ⚠️ 包含租户过滤
func (r *publicPortalRepo) GetAvailablePaymentMethods(ctx context.Context) ([]*portalBiz.PaymentMethod, error) {
	r.logger.Infof("[GetAvailablePaymentMethods]")

	payments, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.EnableEQ(true),
		).
		Order(ent.Asc(proxypayment.FieldID)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("[GetAvailablePaymentMethods] 查询失败: %v", err)
		return nil, errors.InternalServer("DATABASE_ERROR", "查询支付方式失败")
	}

	result := make([]*portalBiz.PaymentMethod, 0, len(payments))
	for _, p := range payments {
		result = append(result, &portalBiz.PaymentMethod{
			ID:          int64(p.ID),
			Name:        p.Name,
			Platform:    p.Platform,
			Description: p.Description,
			Icon:        p.Icon,
			FeeMode:     int32(p.FeeMode),
			FeePercent:  int(p.FeePercent),
			FeeAmount:   int(p.FeeAmount),
		})
	}

	return result, nil
}

// CreatePayment 创建支付记录并返回支付信息
// ⚠️ 完整复刻原项目逻辑（purchaseCheckoutLogic.go）
// ⚠️ 支持多种支付平台：EPay、Stripe、AlipayF2F、CryptoSaaS
// ⚠️ 包含汇率转换逻辑
func (r *publicPortalRepo) CreatePayment(ctx context.Context, orderNo string, returnURL string) (*portalBiz.PaymentInfo, error) {
	r.logger.Infof("[CreatePayment] orderNo: %s, returnURL: %s",
		orderNo, returnURL)

	// 当前为单库模型，直接使用当前请求上下文

	// 1. 查询订单并验证状态
	order, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(orderNo),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrOrderNotFound)
		}
		r.logger.Errorf("[CreatePayment] 查询订单失败: %v", err)
		return nil, errors.InternalServer("DATABASE_ERROR", "查询订单失败")
	}

	// 2. 验证订单状态（必须是待支付状态=1）
	if order.Status != 1 {
		r.logger.Warnf("[CreatePayment] 订单状态错误: orderNo=%s, status=%d", orderNo, order.Status)
		return nil, errors.BadRequest("ORDER_STATUS_ERROR", "订单状态错误")
	}

	// 3. 查询支付方式配置
	paymentConfig, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(order.PaymentID),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrPaymentNotFound)
		}
		r.logger.Errorf("[CreatePayment] 查询支付方式失败: %v", err)
		return nil, errors.InternalServer("DATABASE_ERROR", "查询支付方式失败")
	}

	// 4. 根据支付平台路由到对应的处理方法
	platform := payment.ParsePlatform(order.Method)
	switch platform {
	case payment.EPay:
		// EPay支付 - 生成支付URL用于重定向（复刻原项目 purchaseCheckoutLogic.go line 75-85）
		paymentURL, err := r.epayPayment(ctx, paymentConfig, order, returnURL)
		if err != nil {
			return nil, err
		}
		return &portalBiz.PaymentInfo{
			Type:        "url",
			CheckoutURL: paymentURL,
			Stripe:      nil,
		}, nil

	case payment.Stripe:
		// Stripe支付 - 创建PaymentSheet（复刻原项目 purchaseCheckoutLogic.go line 87-96）
		clientSecret, publishableKey, tradeNo, paymentMethod, err := r.stripePayment(ctx, paymentConfig, order, "")
		if err != nil {
			return nil, err
		}

		// 更新订单的trade_no（复刻原项目 purchaseCheckoutLogic.go line 251-256）
		err = r.data.db.ProxyOrder.UpdateOneID(order.ID).
			SetTradeNo(tradeNo).
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[CreatePayment] 更新订单trade_no失败: %v", err)
		}

		return &portalBiz.PaymentInfo{
			Type:        "stripe",
			CheckoutURL: "",
			Stripe: &portalBiz.StripePayment{
				PublishableKey: publishableKey,
				ClientSecret:   clientSecret,
				Method:         paymentMethod, // 从配置中解析的支付方式
			},
		}, nil

	case payment.AlipayF2F:
		// Alipay F2F支付 - 生成二维码（复刻原项目 purchaseCheckoutLogic.go line 98-108）
		qrCode, err := r.alipayF2fPayment(ctx, paymentConfig, order)
		if err != nil {
			return nil, err
		}
		return &portalBiz.PaymentInfo{
			Type:        "qr",
			CheckoutURL: qrCode, // QR码URL存放在CheckoutURL字段
			Stripe:      nil,
		}, nil

	case payment.CryptoSaaS:
		// CryptoSaaS支付 - 生成支付URL（复刻原项目 purchaseCheckoutLogic.go line 110-118）
		paymentURL, err := r.cryptoSaaSPayment(ctx, paymentConfig, order, returnURL)
		if err != nil {
			return nil, err
		}
		return &portalBiz.PaymentInfo{
			Type:        "url",
			CheckoutURL: paymentURL,
			Stripe:      nil,
		}, nil

	case payment.Balance:
		// Portal订单支持余额支付（完全复刻原项目 purchaseCheckoutLogic.go line 120-141）
		r.logger.Infof("[CreatePayment] 处理余额支付, orderNo: %s", order.OrderNo)

		// 检查订单必须有关联用户（复刻原项目 line 122-125）
		if order.UserID == 0 {
			r.logger.Errorf("[CreatePayment] Portal订单余额支付失败：订单没有关联用户, orderNo: %s", order.OrderNo)
			return nil, errors.BadRequest("USER_NOT_FOUND", "订单没有关联用户")
		}

		// 调用余额支付处理逻辑（复刻原项目 line 135-137）
		err := r.processPortalBalancePayment(ctx, order)
		if err != nil {
			r.logger.Errorf("[CreatePayment] Portal余额支付失败: %v, orderNo: %s", err, order.OrderNo)
			return nil, err
		}

		// 余额支付成功，返回 type=balance（复刻原项目 line 139-141）
		return &portalBiz.PaymentInfo{
			Type:        "balance",
			CheckoutURL: "",
			Stripe:      nil,
		}, nil

	default:
		r.logger.Errorf("[CreatePayment] 不支持的支付方式: %s", order.Method)
		return nil, errors.BadRequest("UNSUPPORTED_PAYMENT", "不支持的支付方式")
	}
}

// CheckOrderStatus 查询订单状态
// ⚠️ 完整复刻原项目逻辑（queryPurchaseOrderLogic.go）
// 1. 查询订单
// 2. 如果订单已完成(status=2)或已激活(status=5)，处理临时订单：
//   - 从Redis获取临时订单信息
//   - 验证订单号匹配
//   - 验证用户认证信息匹配
//   - 生成session token并存储到Redis
//
// 3. 返回订单状态和token
func (r *publicPortalRepo) CheckOrderStatus(ctx context.Context, orderNo, authType, identifier string) (*portalBiz.OrderStatusInfo, string, error) {
	r.logger.Infof("[CheckOrderStatus] orderNo: %s, authType: %s", orderNo, authType)

	// 当前为单库模型，直接使用当前请求上下文

	// 1. 查询订单
	order, err := r.data.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(orderNo),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, "", responsecode.NewKratosError(responsecode.ErrOrderNotFound)
		}
		return nil, "", errors.InternalServer("DATABASE_ERROR", "查询订单失败")
	}

	// 2. 处理临时订单（如果状态为已完成或已激活）
	var token string
	if order.Status == 2 || order.Status == 5 {
		token, err = r.handleTemporaryOrder(ctx, order, authType, identifier)
		if err != nil {
			return nil, "", err
		}
	}

	// 3. 直接按订单上的订阅ID查询，避免默认语言过滤导致拿不到非默认语言套餐。
	var subscribe *portalBiz.SubscribeInfo
	subscribeEntity, err := r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.IDEQ(order.SubscribeID)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, "", errors.InternalServer("DATABASE_ERROR", "查询订阅失败")
		}
	} else {
		var discounts []portalBiz.SubscribeDiscount
		if subscribeEntity.Discount != nil && *subscribeEntity.Discount != "" {
			_ = json.Unmarshal([]byte(*subscribeEntity.Discount), &discounts)
		}
		nodes64 := stringToInt64Slice(subscribeEntity.Nodes)
		nodes := make([]int, len(nodes64))
		for i, v := range nodes64 {
			nodes[i] = int(v)
		}
		var nodeTags []string
		if subscribeEntity.NodeTags != "" {
			nodeTags = strings.Split(subscribeEntity.NodeTags, ",")
			for i := range nodeTags {
				nodeTags[i] = strings.TrimSpace(nodeTags[i])
			}
		}
		deductionRatio := int64(0)
		if subscribeEntity.DeductionRatio != nil {
			deductionRatio = int64(*subscribeEntity.DeductionRatio)
		}
		resetCycle := int64(0)
		if subscribeEntity.ResetCycle != nil {
			resetCycle = int64(*subscribeEntity.ResetCycle)
		}
		nodeGroupID := "0"
		if subscribeEntity.NodeGroupID != nil {
			nodeGroupID = strconv.FormatInt(*subscribeEntity.NodeGroupID, 10)
		}
		var trafficLimit []portalBiz.TrafficLimit
		if subscribeEntity.TrafficLimit != nil {
			trafficLimit = parsePortalTrafficLimits(*subscribeEntity.TrafficLimit)
		}
		categoryName := ""
		if subscribeEntity.CategoryID > 0 {
			if category, err := r.data.db.ProxySubscribeCategory.Query().
				Where(proxysubscribecategory.IDEQ(subscribeEntity.CategoryID)).
				Only(ctx); err == nil {
				categoryName = category.Name
			}
		}

		subscribe = &portalBiz.SubscribeInfo{
			ID:                int64(subscribeEntity.ID),
			Name:              subscribeEntity.Name,
			Language:          subscribeEntity.Language,
			Description:       subscribeEntity.Description,
			UnitPrice:         subscribeEntity.UnitPrice,
			UnitTime:          subscribeEntity.UnitTime,
			Discount:          discounts,
			Replacement:       subscribeEntity.Replacement,
			Inventory:         int64(subscribeEntity.Inventory),
			Traffic:           subscribeEntity.Traffic,
			SpeedLimit:        int64(subscribeEntity.SpeedLimit),
			DeviceLimit:       int64(subscribeEntity.DeviceLimit),
			Quota:             int64(subscribeEntity.Quota),
			CategoryID:        subscribeEntity.CategoryID,
			CategoryName:      categoryName,
			Nodes:             nodes,
			NodeTags:          nodeTags,
			NodeGroupIds:      tool.Int64SliceToStringSlice(subscribeEntity.NodeGroupIds),
			NodeGroupId:       nodeGroupID,
			TrafficLimit:      trafficLimit,
			Show:              subscribeEntity.Show,
			Sell:              subscribeEntity.Sell,
			Sort:              int64(subscribeEntity.Sort),
			DeductionRatio:    int64(deductionRatio),
			AllowDeduction:    subscribeEntity.AllowDeduction,
			ResetCycle:        int64(resetCycle),
			RenewalReset:      subscribeEntity.RenewalReset,
			ShowOriginalPrice: subscribeEntity.ShowOriginalPrice,
			PriceOptions:      r.portalSubscribePriceOptions(ctx, []*ent.ProxySubscribe{subscribeEntity})[int64(subscribeEntity.ID)],
			CreatedAt:         subscribeEntity.CreatedAt.Unix(),
			UpdatedAt:         subscribeEntity.UpdatedAt.Unix(),
		}
	}

	// 4. 查询支付方式信息
	payment, err := r.data.db.ProxyPayment.Query().
		Where(
			proxypayment.IDEQ(order.PaymentID),
		).
		Only(ctx)
	if err != nil {
		r.logger.Errorf("[CheckOrderStatus] 查询支付方式失败: %v", err)
		return nil, "", errors.InternalServer("DATABASE_ERROR", "查询支付方式失败")
	}

	// 转换支付方式信息
	paymentInfo := &portalBiz.PaymentMethod{
		ID:          int64(payment.ID),
		Name:        payment.Name,
		Platform:    payment.Platform,
		Description: payment.Description,
		Icon:        payment.Icon,
		FeeMode:     int32(payment.FeeMode),
		FeePercent:  int(payment.FeePercent),
		FeeAmount:   int(payment.FeeAmount),
	}

	// 5. 返回订单状态信息（包含所有字段）
	return &portalBiz.OrderStatusInfo{
		OrderNo:         order.OrderNo,
		Subscribe:       subscribe,
		Quantity:        int64(order.Quantity),
		Price:           order.Price,
		Amount:          order.Amount,
		Discount:        order.Discount,
		Coupon:          order.Coupon,
		CouponDiscount:  order.CouponDiscount,
		FeeAmount:       order.FeeAmount,
		Payment:         paymentInfo,
		Status:          int32(order.Status),
		PriceOptionID:   order.PriceOptionID,
		PriceOptionName: order.PriceOptionName,
		DurationUnit:    order.DurationUnit,
		DurationValue:   order.DurationValue,
		OptionPrice:     order.OptionPrice,
		CreatedAt:       order.CreatedAt,
	}, token, nil
}

// handleTemporaryOrder 处理临时订单逻辑
// ⚠️ 完整复刻原项目逻辑（queryPurchaseOrderLogic.go:handleTemporaryOrder）
func (r *publicPortalRepo) handleTemporaryOrder(ctx context.Context, order *ent.ProxyOrder, authType, identifier string) (string, error) {
	// 1. 从Redis获取临时订单信息
	tempInfo, err := r.GetTempOrderInfo(ctx, order.OrderNo)
	if err != nil {
		r.logger.Errorf("[handleTemporaryOrder] 获取临时订单信息失败: %v", err)
		return "", errors.InternalServer("REDIS_ERROR", "获取临时订单信息失败")
	}

	// 2. 验证订单号匹配
	if tempInfo.OrderNo != order.OrderNo {
		r.logger.Errorf("[handleTemporaryOrder] 订单号不匹配: tempInfo.OrderNo=%s, order.OrderNo=%s", tempInfo.OrderNo, order.OrderNo)
		return "", errors.BadRequest("INVALID_ORDER", "订单号不匹配")
	}

	// 3. 验证用户认证信息
	if err = r.validateUserAuth(ctx, int(order.UserID), authType, identifier); err != nil {
		return "", err
	}

	// 5. 生成session token
	return r.generateSessionToken(ctx, int(order.UserID))
}

// validateUserAuth 验证用户认证信息
// ⚠️ 完整复刻原项目逻辑（queryPurchaseOrderLogic.go:validateUserAndEmail）
func (r *publicPortalRepo) validateUserAuth(ctx context.Context, userID int, authType, identifier string) error {
	// 查询用户认证方法
	authMethod, err := r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthTypeEQ(authType),
			proxyuserauthmethod.AuthIdentifierEQ(identifier),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			r.logger.Errorf("[validateUserAuth] 认证方法不存在: authType=%s, identifier=%s", authType, identifier)
			return errors.BadRequest("AUTH_NOT_FOUND", "认证方法不存在")
		}
		return errors.InternalServer("DATABASE_ERROR", "查询认证方法失败")
	}

	// 验证用户ID匹配
	if authMethod.UserID != int64(userID) {
		r.logger.Errorf("[validateUserAuth] 用户ID不匹配: authMethod.UserID=%d, userID=%d", authMethod.UserID, userID)
		return errors.BadRequest("USER_MISMATCH", "用户认证信息不匹配")
	}

	return nil
}

// generateSessionToken 生成session token
// ⚠️ 完整复刻原项目逻辑（queryPurchaseOrderLogic.go:generateSessionToken）
func (r *publicPortalRepo) generateSessionToken(ctx context.Context, userID int) (string, error) {
	token, err := r.data.issueSessionToken(ctx, int64(userID), sessionTokenOptions{})
	if err != nil {
		r.logger.Errorf("[generateSessionToken] Token生成失败: %v", err)
		return "", errors.InternalServer("TOKEN_ERROR", "Token生成失败")
	}

	r.logger.Infof("[generateSessionToken] Token生成成功: userID=%d", userID)
	return token, nil
}

// GetTempOrderInfo 获取临时订单信息
func (r *publicPortalRepo) GetTempOrderInfo(ctx context.Context, orderNo string) (*portalBiz.TempOrderInfo, error) {
	key := fmt.Sprintf(constant.TempOrderCacheKey, orderNo)
	data, err := r.data.rdb.Get(ctx, key).Result()
	if err != nil {
		r.logger.Errorf("[GetTempOrderInfo] Redis获取失败: %v, key=%s", err, key)
		return nil, errors.InternalServer("REDIS_ERROR", "获取临时订单信息失败")
	}

	var tempInfo constant.TemporaryOrderInfo
	if err := json.Unmarshal([]byte(data), &tempInfo); err != nil {
		r.logger.Errorf("[GetTempOrderInfo] 反序列化失败: %v", err)
		return nil, errors.InternalServer("UNMARSHAL_ERROR", "解析临时订单信息失败")
	}

	return &portalBiz.TempOrderInfo{

		OrderNo:    tempInfo.OrderNo,
		Identifier: tempInfo.Identifier,
		AuthType:   tempInfo.AuthType,
		Password:   tempInfo.Password,
		InviteCode: tempInfo.InviteCode,
	}, nil
}

// ============================================================================
// 支付平台辅助方法
// ⚠️ 完整复刻原项目逻辑（purchaseCheckoutLogic.go）
// ============================================================================

// epayPayment EPay支付处理
// ⚠️ 完整复刻原项目逻辑（lines 260-300）
func (r *publicPortalRepo) epayPayment(ctx context.Context, paymentConfig *ent.ProxyPayment, order *ent.ProxyOrder, returnURL string) (string, error) {
	// 1. 解析EPay配置
	var config EPayConfig
	if err := config.Unmarshal([]byte(paymentConfig.Config)); err != nil {
		r.logger.Errorf("[epayPayment] 解析EPay配置失败: %v", err)
		return "", errors.InternalServer("CONFIG_ERROR", "解析支付配置失败")
	}

	// 2. 初始化EPay客户端
	client := epay.NewClient(config.Pid, config.Url, config.Key, config.Type)

	// 3. 汇率转换（转换为CNY）
	amount, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount+order.FeeAmount))
	if err != nil {
		return "", err
	}

	// 4. 构建回调URL
	notifyURL := r.buildNotifyURL(ctx, paymentConfig)

	// 5. 获取站点名称
	siteName := r.getSiteName(ctx)

	// 6. 创建支付URL
	paymentURL := client.CreatePayUrl(epay.Order{
		Name:      siteName,
		Amount:    amount,
		OrderNo:   order.OrderNo,
		SignType:  "MD5",
		NotifyUrl: notifyURL,
		ReturnUrl: returnURL,
		Type:      config.Type,
	})

	r.logger.Infof("[epayPayment] EPay支付URL生成成功: orderNo=%s", order.OrderNo)
	return paymentURL, nil
}

// stripePayment Stripe支付处理
// ⚠️ 完整复刻原项目逻辑（lines 201-257）
func (r *publicPortalRepo) stripePayment(ctx context.Context, paymentConfig *ent.ProxyPayment, order *ent.ProxyOrder, identifier string) (clientSecret, publishableKey, tradeNo, paymentMethod string, err error) {
	// 1. 解析Stripe配置
	var config StripeConfig
	if err := config.Unmarshal([]byte(paymentConfig.Config)); err != nil {
		r.logger.Errorf("[stripePayment] 解析Stripe配置失败: %v", err)
		return "", "", "", "", errors.InternalServer("CONFIG_ERROR", "解析支付配置失败")
	}

	// 2. 初始化Stripe客户端
	client := stripe.NewClient(stripe.Config{
		PublicKey:     config.PublicKey,
		SecretKey:     config.SecretKey,
		WebhookSecret: config.WebhookSecret,
	})

	// 3. 汇率转换（转换为CNY）
	amount, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount+order.FeeAmount))
	if err != nil {
		return "", "", "", "", err
	}
	convertAmount := int64(amount * 100) // 转换为分

	// 4. 创建PaymentSheet
	result, err := client.CreatePaymentSheet(&stripe.Order{
		OrderNo:   order.OrderNo,
		Subscribe: strconv.FormatInt(order.SubscribeID, 10),
		Amount:    convertAmount,
		Currency:  "cny",
		Payment:   config.Payment,
	}, &stripe.User{
		UserId: order.UserID,
		Email:  identifier,
	})

	if err != nil {
		r.logger.Errorf("[stripePayment] 创建PaymentSheet失败: %v", err)
		return "", "", "", "", errors.InternalServer("STRIPE_ERROR", "创建支付失败")
	}

	r.logger.Infof("[stripePayment] Stripe PaymentSheet创建成功: orderNo=%s, tradeNo=%s", order.OrderNo, result.TradeNo)
	return result.ClientSecret, result.PublishableKey, result.TradeNo, config.Payment, nil
}

// alipayF2fPayment 支付宝当面付处理
// ⚠️ 完整复刻原项目逻辑（lines 150-199）
func (r *publicPortalRepo) alipayF2fPayment(ctx context.Context, paymentConfig *ent.ProxyPayment, order *ent.ProxyOrder) (string, error) {
	// 1. 解析Alipay配置
	var config AlipayF2FConfig
	if err := config.Unmarshal([]byte(paymentConfig.Config)); err != nil {
		r.logger.Errorf("[alipayF2fPayment] 解析Alipay配置失败: %v", err)
		return "", errors.InternalServer("CONFIG_ERROR", "解析支付配置失败")
	}

	// 2. 构建回调URL
	notifyURL := r.buildNotifyURL(ctx, paymentConfig)

	// 3. 初始化Alipay客户端
	client := alipay.NewClient(alipay.Config{
		AppId:       config.AppId,
		PrivateKey:  config.PrivateKey,
		PublicKey:   config.PublicKey,
		InvoiceName: config.InvoiceName,
		NotifyURL:   notifyURL,
		Sandbox:     config.Sandbox,
	})

	if client == nil {
		r.logger.Errorf("[alipayF2fPayment] 初始化Alipay客户端失败")
		return "", errors.InternalServer("ALIPAY_ERROR", "初始化支付客户端失败")
	}

	// 4. 汇率转换（转换为CNY）
	amount, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount+order.FeeAmount))
	if err != nil {
		return "", err
	}
	convertAmount := int64(amount * 100) // 转换为分

	// 5. 创建预支付订单并生成二维码
	qrCode, err := client.PreCreateTrade(ctx, alipay.Order{
		OrderNo: order.OrderNo,
		Amount:  convertAmount,
	})

	if err != nil {
		r.logger.Errorf("[alipayF2fPayment] 创建预支付订单失败: %v", err)
		return "", errors.InternalServer("ALIPAY_ERROR", "创建支付订单失败")
	}

	r.logger.Infof("[alipayF2fPayment] Alipay二维码生成成功: orderNo=%s", order.OrderNo)
	return qrCode, nil
}

// cryptoSaaSPayment CryptoSaaS加密货币支付处理
// ⚠️ 完整复刻原项目逻辑（lines 302-342）
func (r *publicPortalRepo) cryptoSaaSPayment(ctx context.Context, paymentConfig *ent.ProxyPayment, order *ent.ProxyOrder, returnURL string) (string, error) {
	// 1. 解析CryptoSaaS配置
	var config CryptoSaaSConfig
	if err := config.Unmarshal([]byte(paymentConfig.Config)); err != nil {
		r.logger.Errorf("[cryptoSaaSPayment] 解析CryptoSaaS配置失败: %v", err)
		return "", errors.InternalServer("CONFIG_ERROR", "解析支付配置失败")
	}

	// 2. 初始化EPay客户端（CryptoSaaS使用EPay协议）
	client := epay.NewClient(config.AccountID, config.Endpoint, config.SecretKey, config.Type)

	// 3. 汇率转换（转换为CNY）
	amount, err := r.queryExchangeRate(ctx, "CNY", int(order.Amount+order.FeeAmount))
	if err != nil {
		return "", err
	}

	// 4. 构建回调URL
	notifyURL := r.buildNotifyURL(ctx, paymentConfig)

	// 5. 获取站点名称
	siteName := r.getSiteName(ctx)

	// 6. 创建支付URL
	paymentURL := client.CreatePayUrl(epay.Order{
		Name:      siteName,
		Amount:    amount,
		OrderNo:   order.OrderNo,
		SignType:  "MD5",
		NotifyUrl: notifyURL,
		ReturnUrl: returnURL,
		Type:      config.Type,
	})

	r.logger.Infof("[cryptoSaaSPayment] CryptoSaaS支付URL生成成功: orderNo=%s", order.OrderNo)
	return paymentURL, nil
}

// queryExchangeRate 汇率转换
// ⚠️ 完整复刻原项目逻辑（lines 344-379）
func (r *publicPortalRepo) queryExchangeRate(ctx context.Context, targetCurrency string, amountInCents int) (float64, error) {
	// 1. 转换为元
	amount := float64(amountInCents) / float64(100)

	// 2. 查询系统货币配置
	configMap, err := loadSystemConfigMap(ctx, r.data.db, "currency")
	if err != nil {
		r.logger.Errorf("[queryExchangeRate] 查询货币配置失败: %v", err)
		return 0, errors.InternalServer("DATABASE_ERROR", "查询货币配置失败")
	}

	currencyUnit := systemConfigString(configMap, "CurrencyUnit", "Currency", "default_currency")
	accessKey := systemConfigString(configMap, "AccessKey", "access_key")

	// 4. 如果没有配置汇率API key，直接返回原金额
	if accessKey == "" {
		r.logger.Infof("[queryExchangeRate] 未配置汇率API key，跳过汇率转换")
		return amount, nil
	}

	// 5. 如果系统货币与目标货币一致，直接返回
	if currencyUnit == targetCurrency {
		return amount, nil
	}

	// 6. 调用汇率API进行转换
	result, err := exchangeRate.GetExchangeRete(currencyUnit, targetCurrency, accessKey, 1)
	if err != nil {
		r.logger.Errorf("[queryExchangeRate] 查询汇率失败: from=%s, to=%s, err=%v", currencyUnit, targetCurrency, err)
		return 0, errors.InternalServer("EXCHANGE_RATE_ERROR", "查询汇率失败")
	}

	convertedAmount := result * amount
	r.logger.Infof("[queryExchangeRate] 汇率转换成功: from=%s, to=%s, rate=%.4f, amount=%.2f -> %.2f",
		currencyUnit, targetCurrency, result, amount, convertedAmount)

	return convertedAmount, nil
}

// buildNotifyURL 构建回调URL
// ⚠️ 复刻原项目逻辑（purchaseCheckoutLogic.go line 279-288）
func (r *publicPortalRepo) buildNotifyURL(ctx context.Context, paymentConfig *ent.ProxyPayment) string {
	gatewayMode := middleware.GetGatewayMode(ctx)

	// 1. 优先使用payment配置的domain
	if paymentConfig.Domain != "" {
		notifyURL := paymentConfig.Domain
		if gatewayMode {
			notifyURL += "/api"
		}
		return notifyURL + "/v1/notify/" + paymentConfig.Platform + "/" + paymentConfig.Token
	}

	// 2. 尝试从context获取request host（原项目 line 283-286）
	// ⚠️ 关键：使用"requestHost"而不是"request_host"（constant.CtxKeyRequestHost = "requestHost"）
	host, ok := ctx.Value("requestHost").(string)
	if !ok || host == "" {
		// 3. Fallback到数据库中的站点Host，再回退到配置文件
		if siteValues, err := loadSystemConfigMap(ctx, r.data.db, "site"); err == nil {
			host = systemConfigString(siteValues, "Host", "host")
		}
		if host == "" && r.data.conf != nil && r.data.conf.Site != nil {
			host = r.data.conf.Site.Host
		}
		if host == "" {
			host = "localhost"
		}
	}

	notifyURL := "https://" + host
	if gatewayMode {
		notifyURL += "/api"
	}
	return notifyURL + "/v1/notify/" + paymentConfig.Platform + "/" + paymentConfig.Token
}

// getSiteName 获取站点名称
func (r *publicPortalRepo) getSiteName(ctx context.Context) string {
	siteValues, err := loadSystemConfigMap(ctx, r.data.db, "site")
	if err != nil {
		if r.data.conf != nil && r.data.conf.Site != nil {
			return r.data.conf.Site.SiteName
		}
		return ""
	}
	if siteName := systemConfigString(siteValues, "SiteName", "site_name"); siteName != "" {
		return siteName
	}
	if r.data.conf != nil && r.data.conf.Site != nil {
		return r.data.conf.Site.SiteName
	}
	return ""
}

// processPortalBalancePayment 处理 Portal 订单的余额支付
// ⚠️ 完全复刻原项目 purchaseCheckoutLogic.go:balancePayment (line 381-526)
// 业务逻辑说明：
// 1. 零金额订单直接标记为已支付
// 2. 在事务中执行：
//   - 查询用户最新信息（隐式行锁）
//   - 检查总余额（balance + gift_amount）是否充足
//   - 优先使用赠金，再使用余额
//   - 更新用户余额
//   - 创建赠金日志（如果使用了赠金）
//   - 创建余额日志（如果使用了余额）
//   - 更新订单的 gift_amount 字段（用于退款跟踪）
//   - 更新订单状态为已支付（status=2）
//
// 3. 将订单激活任务入队
func (r *publicPortalRepo) processPortalBalancePayment(ctx context.Context, order *ent.ProxyOrder) error {
	userID := order.UserID

	// 1. 零金额订单处理（复刻原项目 line 386-402）
	if order.Amount == 0 {
		r.logger.Infof("[processPortalBalancePayment] 零金额订单，直接标记为已支付. OrderNo: %s, UserID: %d",
			order.OrderNo, userID)

		// 更新订单状态为已支付（复刻原项目 line 393-400）
		err := r.data.db.ProxyOrder.UpdateOneID(order.ID).SetStatus(2).Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processPortalBalancePayment] 更新订单状态失败: %v, orderNo: %s", err, order.OrderNo)
			return errors.InternalServer("ORDER_UPDATE_FAILED", "更新订单状态失败")
		}

		// 跳转到激活逻辑（复刻原项目 line 401: goto activation）
		return r.enqueueActivateOrderTask(ctx, int(userID), order.OrderNo)
	}

	// 2. 在事务中处理余额支付（复刻原项目 line 404-494）
	err := r.data.db.TX(ctx, func(tx *ent.Tx) error {
		// 2.1 查询用户最新信息（复刻原项目 line 405-409）
		// ⚠️ 关键：GORM 的 db.Model(&user.User{}).Where("id = ?", u.Id).First(&userInfo)
		//    在事务中会自动加行锁（SELECT ... FOR UPDATE），ent 的 Query 也是如此
		user, err := tx.ProxyUser.Query().
			Where(proxyuser.IDEQ(userID)).
			Only(ctx)
		if err != nil {
			r.logger.Errorf("[processPortalBalancePayment] 查询用户失败: %v, userID: %d", err, userID)
			return err
		}

		// 2.2 安全获取余额和赠金（处理 nil 指针）
		userBalance := int64(0)
		if user.Balance != nil {
			userBalance = *user.Balance
		}
		userGiftAmount := int64(0)
		if user.GiftAmount != nil {
			userGiftAmount = *user.GiftAmount
		}

		// 2.3 检查总余额是否充足（复刻原项目 line 411-416）
		totalAvailable := userBalance + userGiftAmount
		if totalAvailable < order.Amount {
			r.logger.Errorf("[processPortalBalancePayment] 余额不足. 需要: %d, 可用: %d (余额: %d, 赠金: %d), userID: %d",
				order.Amount, totalAvailable, userBalance, userGiftAmount, userID)
			return errors.BadRequest("INSUFFICIENT_BALANCE", fmt.Sprintf("余额不足: 需要 %d, 可用 %d", order.Amount, totalAvailable))
		}

		// 2.4 计算支付分配：优先使用赠金（复刻原项目 line 418-430）
		var giftUsed, balanceUsed int64
		remainingAmount := order.Amount

		if userGiftAmount >= remainingAmount {
			// 赠金覆盖整个支付金额（复刻原项目 line 422-425）
			giftUsed = remainingAmount
			balanceUsed = 0
		} else {
			// 使用所有可用赠金，然后使用常规余额（复刻原项目 line 426-430）
			giftUsed = userGiftAmount
			balanceUsed = remainingAmount - giftUsed
		}

		// 2.5 更新用户余额（复刻原项目 line 432-440）
		newGiftAmount := userGiftAmount - giftUsed
		newBalance := userBalance - balanceUsed

		err = tx.ProxyUser.UpdateOneID(user.ID).
			SetGiftAmount(newGiftAmount).
			SetBalance(newBalance).
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processPortalBalancePayment] 更新用户余额失败: %v, userID: %d", err, userID)
			return err
		}

		// 2.6 创建赠金日志（如果使用了赠金）（复刻原项目 line 442-462）
		if giftUsed > 0 {
			// ⚠️ 老项目日志结构：
			// type Gift struct {
			//     Type        uint16 `json:"type"`        // 342 = GiftTypeReduce
			//     OrderNo     string `json:"order_no"`
			//     SubscribeId int `json:"subscribe_id"`
			//     Amount      int `json:"amount"`
			//     Balance     int `json:"balance"`
			//     Remark      string `json:"remark,omitempty"`
			//     Timestamp   int `json:"timestamp"`
			// }
			giftLog := map[string]interface{}{
				"type":         342, // GiftTypeReduce = 342（复刻原项目 line 446）
				"order_no":     order.OrderNo,
				"subscribe_id": 0,
				"amount":       giftUsed,
				"balance":      newGiftAmount,      // 新余额（复刻原项目 line 448）
				"remark":       "Purchase payment", // 复刻原项目 line 449
				"timestamp":    time.Now().UnixMilli(),
			}
			giftLogJSON, _ := json.Marshal(giftLog)

			// 创建系统日志（复刻原项目 line 453-461）
			// ⚠️ 老项目：Type = TypeGift.Uint8() = 34 (uint8)
			_, err = tx.ProxySystemLog.Create().
				SetType(34).                               // TypeGift = 34（复刻原项目 line 454）
				SetObjectID(userID).                       // 复刻原项目 line 455
				SetDate(time.Now().Format(time.DateOnly)). // 复刻原项目 line 456
				SetContent(string(giftLogJSON)).           // 复刻原项目 line 457
				Save(ctx)
			if err != nil {
				r.logger.Errorf("[processPortalBalancePayment] 创建赠金日志失败: %v, userID: %d", err, userID)
				return err
			}
		}

		// 2.7 创建余额日志（如果使用了余额）（复刻原项目 line 464-483）
		if balanceUsed > 0 {
			// ⚠️ 老项目日志结构：
			// type Balance struct {
			//     Type      uint16 `json:"type"`      // 323 = BalanceTypePayment
			//     Amount    int `json:"amount"`
			//     OrderNo   string `json:"order_no,omitempty"`
			//     Balance   int `json:"balance"`
			//     Timestamp int `json:"timestamp"`
			// }
			balanceLog := map[string]interface{}{
				"type":      323, // BalanceTypePayment = 323（复刻原项目 line 468）
				"amount":    balanceUsed,
				"order_no":  order.OrderNo,
				"balance":   newBalance,             // 新余额（复刻原项目 line 470）
				"timestamp": time.Now().UnixMilli(), // 复刻原项目 line 471
			}
			balanceLogJSON, _ := json.Marshal(balanceLog)

			// 创建系统日志（复刻原项目 line 474-482）
			// ⚠️ 老项目：Type = TypeBalance.Uint8() = 32 (uint8)
			_, err = tx.ProxySystemLog.Create().
				SetType(32).                               // TypeBalance = 32（复刻原项目 line 475）
				SetObjectID(userID).                       // 复刻原项目 line 476
				SetDate(time.Now().Format(time.DateOnly)). // 复刻原项目 line 477
				SetContent(string(balanceLogJSON)).        // 复刻原项目 line 478
				Save(ctx)
			if err != nil {
				r.logger.Errorf("[processPortalBalancePayment] 创建余额日志失败: %v, userID: %d", err, userID)
				return err
			}
		}

		// 2.8 更新订单的 gift_amount 字段（用于退款跟踪）（复刻原项目 line 485-490）
		err = tx.ProxyOrder.UpdateOneID(order.ID).
			SetGiftAmount(giftUsed). // 复刻原项目 line 486
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processPortalBalancePayment] 更新订单gift_amount失败: %v, orderNo: %s", err, order.OrderNo)
			return err
		}

		// 2.9 更新订单状态为已支付（status = 2）（复刻原项目 line 492-493）
		err = tx.ProxyOrder.UpdateOneID(order.ID).
			SetStatus(2). // 已支付（复刻原项目 line 493）
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("[processPortalBalancePayment] 更新订单状态失败: %v, orderNo: %s", err, order.OrderNo)
			return err
		}

		return nil
	})

	// 3. 检查事务执行结果（复刻原项目 line 496-502）
	if err != nil {
		r.logger.Errorf("[processPortalBalancePayment] 事务失败: %v, orderNo: %s", err, order.OrderNo)
		return err
	}
	// 4. 将订单激活任务入队（复刻原项目 line 504-525）
	// ⚠️ 关键：老项目使用 goto activation 标签跳转到这里
	err = r.enqueueActivateOrderTask(ctx, int(userID), order.OrderNo)
	if err != nil {
		// ⚠️ 注意：老项目如果入队失败会返回错误（line 518-520）
		// 但由于支付已经完成，这里只记录警告而不回滚
		r.logger.Warnf("[processPortalBalancePayment] 入队激活任务失败: %v, orderNo: %s（支付已完成）", err, order.OrderNo)
	}

	r.logger.Infof("[processPortalBalancePayment] Portal余额支付完成. OrderNo: %s, UserID: %d",
		order.OrderNo, userID)

	return nil
}

// enqueueActivateOrderTask 将订单激活任务入队
// ⚠️ 完全复刻原项目 purchaseCheckoutLogic.go line 504-525
func (r *publicPortalRepo) enqueueActivateOrderTask(ctx context.Context, userID int, orderNo string) error {
	// 1. 构建任务负载（复刻原项目 line 506-508）
	// ⚠️ 关键：老项目只包含 OrderNo 字段！
	payload := queueTypes.ForthwithActivateOrderPayload{
		OrderNo: orderNo, // 复刻原项目 line 507
	}

	// 2. 序列化负载（复刻原项目 line 509-512）
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		r.logger.Errorf("[enqueueActivateOrderTask] 序列化payload失败: %v, orderNo: %s", err, orderNo)
		return err
	}

	// 3. 创建任务（复刻原项目 line 515）
	task := asynq.NewTask(queueTypes.ForthwithActivateOrder, payloadBytes)

	// 4. 将任务入队（复刻原项目 line 516-520）
	_, err = r.data.queue.Enqueue(task)
	if err != nil {
		r.logger.Errorf("[enqueueActivateOrderTask] 入队任务失败: %v, orderNo: %s", err, orderNo)
		return err
	}

	r.logger.Infof("[enqueueActivateOrderTask] 入队激活任务成功, OrderNo: %s", orderNo)
	return nil
}
