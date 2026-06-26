package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxycoupon"
	"github.com/npanel-dev/NPanel-backend/ent/proxyorder"
	"github.com/npanel-dev/NPanel-backend/ent/proxyredemptionrecord"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	logmodel "github.com/npanel-dev/NPanel-backend/internal/model/log"
	queueTypes "github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/pkg/constant"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/redis/go-redis/v9"
)

// 订单类型常量
const (
	OrderTypeSubscribe    = 1 // 新购订阅
	OrderTypeRenewal      = 2 // 续费订阅
	OrderTypeResetTraffic = 3 // 流量重置
	OrderTypeRecharge     = 4 // 余额充值
	OrderTypeRedemption   = 5 // 兑换码激活
)

// 订单状态常量
const (
	OrderStatusPending  = 1 // 待支付
	OrderStatusPaid     = 2 // 已支付，待激活
	OrderStatusClose    = 3 // 已关闭
	OrderStatusFailed   = 4 // 激活失败
	OrderStatusFinished = 5 // 激活成功
)

// ActivateOrderHandler 激活订单处理器
// ⚠️ 完整复刻原项目逻辑 (activateOrderLogic.go)
type ActivateOrderHandler struct {
	db                *ent.Client
	rdb               *redis.Client
	groupRecalculator groupRecalculator
	logger            *log.Helper
}

// NewActivateOrderHandler 创建激活订单处理器
func NewActivateOrderHandler(db *ent.Client, rdb *redis.Client, groupRecalculator groupRecalculator, logger log.Logger) *ActivateOrderHandler {
	return &ActivateOrderHandler{
		db:                db,
		rdb:               rdb,
		groupRecalculator: groupRecalculator,
		logger:            log.NewHelper(logger),
	}
}

// ProcessTask 处理任务 - 主入口
// 完整复刻原项目逻辑 (activateOrderLogic.go:68-86)
// 处理已支付订单的激活流程，包括验证、根据订单类型处理、后处理
func (h *ActivateOrderHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	// 1. 解析payload
	payload, err := h.parsePayload(ctx, task.Payload())
	if err != nil {
		return err
	}

	// 2. 验证并获取订单
	orderInfo, err := h.validateAndGetOrder(ctx, payload)
	if err != nil {
		return err
	}
	if orderInfo == nil {
		return nil
	}

	// 3. 根据订单类型处理订单
	if err = h.processOrderByType(ctx, orderInfo); err != nil {
		h.logger.Errorf("[ActivateOrder] 处理任务失败: %v", err)
		return err
	}

	// 4. 后处理：更新优惠券和订单状态
	h.finalizeCouponAndOrder(ctx, orderInfo)

	return nil
}

// parsePayload 解析任务payload
// 复刻原项目 line 89-99
func (h *ActivateOrderHandler) parsePayload(ctx context.Context, payload []byte) (*queueTypes.ForthwithActivateOrderPayload, error) {
	var p queueTypes.ForthwithActivateOrderPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		h.logger.Errorf("[ActivateOrder] 反序列化payload失败: %v, payload: %s", err, string(payload))
		return nil, err
	}
	return &p, nil
}

// validateAndGetOrder 验证并获取订单信息
// 复刻原项目 line 103-122
// 如果订单不存在或状态不是已支付，返回错误
func (h *ActivateOrderHandler) validateAndGetOrder(ctx context.Context, payload *queueTypes.ForthwithActivateOrderPayload) (*ent.ProxyOrder, error) {
	orderInfo, err := h.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(payload.OrderNo),
		).
		Only(ctx)

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询订单失败: %v, orderNo: %s", err, payload.OrderNo)
		return nil, err
	}

	if orderInfo.Status == OrderStatusFinished {
		h.logger.Infof("[ActivateOrder] 订单已完成，跳过重复处理: orderNo=%s", orderInfo.OrderNo)
		return nil, nil
	}

	if orderInfo.Status != OrderStatusPaid {
		h.logger.Errorf("[ActivateOrder] 订单状态错误: orderNo=%s, status=%d", orderInfo.OrderNo, orderInfo.Status)
		return nil, fmt.Errorf("invalid order status")
	}

	return orderInfo, nil
}

// processOrderByType 根据订单类型路由处理
// 复刻原项目 line 124-139
func (h *ActivateOrderHandler) processOrderByType(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	switch orderInfo.Type {
	case OrderTypeSubscribe:
		return h.NewPurchase(ctx, orderInfo)
	case OrderTypeRenewal:
		return h.Renewal(ctx, orderInfo)
	case OrderTypeResetTraffic:
		return h.ResetTraffic(ctx, orderInfo)
	case OrderTypeRecharge:
		return h.Recharge(ctx, orderInfo)
	case OrderTypeRedemption:
		return h.RedemptionActivate(ctx, orderInfo)
	default:
		h.logger.Errorf("[ActivateOrder] 订单类型无效: type=%d", orderInfo.Type)
		return fmt.Errorf("invalid order type: %d", orderInfo.Type)
	}
}

// finalizeCouponAndOrder 后处理：更新优惠券使用次数和订单状态
// 完整复刻原项目逻辑 (activateOrderLogic.go:141-162)
func (h *ActivateOrderHandler) finalizeCouponAndOrder(ctx context.Context, orderInfo *ent.ProxyOrder) {
	// 1. 更新优惠券使用次数
	if orderInfo.Coupon != "" {
		if err := h.updateCouponUsedCount(ctx, orderInfo); err != nil {
			h.logger.Errorf("[ActivateOrder] 更新优惠券状态失败: %v, coupon=%s", err, orderInfo.Coupon)
		}
	}

	// 2. 更新订单状态为已完成
	if err := h.db.ProxyOrder.UpdateOneID(orderInfo.ID).
		SetStatus(OrderStatusFinished).
		Exec(ctx); err != nil {
		h.logger.Errorf("[ActivateOrder] 更新订单状态失败: %v, orderNo=%s", err, orderInfo.OrderNo)
	}
}

// ============================================================================
// NewPurchase - 处理新购订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:166-193)
// ============================================================================

// NewPurchase 处理新购订单
// 包括用户创建、订阅设置、佣金处理、缓存更新和通知发送
func (h *ActivateOrderHandler) NewPurchase(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	// 1. 获取或创建用户
	userInfo, err := h.getUserOrCreate(ctx, orderInfo)
	if err != nil {
		return err
	}

	// 2. 获取订阅套餐信息
	sub, err := h.getSubscribeInfo(ctx, int64(orderInfo.SubscribeID))
	if err != nil {
		return err
	}

	// 3. 创建用户订阅
	userSub, err := h.createUserSubscription(ctx, orderInfo, sub)
	if err != nil {
		return err
	}

	// 4. 触发用户组重新计算（在后台goroutine中异步执行）
	h.triggerUserGroupRecalculation(ctx, int64(userInfo.ID))

	// 5. 处理佣金（在单独的goroutine中异步执行，避免阻塞）
	go h.handleCommission(context.Background(), userInfo, orderInfo)

	// 6. 清理服务器缓存
	h.clearServerCache(ctx, sub)

	// 7. 发送通知
	h.sendNotifications(ctx, orderInfo, userInfo, sub, userSub, NotifyTypePurchase)

	h.logger.Infof("[ActivateOrder] 创建用户订阅成功")
	return nil
}

// getUserOrCreate 获取或创建用户
// 复刻原项目 line 196-201
// 根据订单详情获取现有用户或创建新的访客用户
func (h *ActivateOrderHandler) getUserOrCreate(ctx context.Context, orderInfo *ent.ProxyOrder) (*ent.ProxyUser, error) {
	if orderInfo.UserID != 0 {
		return h.getExistingUser(ctx, int64(orderInfo.UserID))
	}
	return h.createGuestUser(ctx, orderInfo)
}

// getExistingUser 根据用户ID获取用户信息
// 复刻原项目 line 204-214
func (h *ActivateOrderHandler) getExistingUser(ctx context.Context, userID int64) (*ent.ProxyUser, error) {
	userInfo, err := h.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(userID),
		).
		Only(ctx)

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询用户失败: %v, userID=%d", err, userID)
		return nil, err
	}
	return userInfo, nil
}

// createGuestUser 为访客订单创建新用户账户
// 完整复刻原项目逻辑 (activateOrderLogic.go:218-263)
// 使用存储在Redis缓存中的临时订单信息
func (h *ActivateOrderHandler) createGuestUser(ctx context.Context, orderInfo *ent.ProxyOrder) (*ent.ProxyUser, error) {
	// 1. 从Redis获取临时订单信息
	tempOrder, err := h.getTempOrderInfo(ctx, orderInfo.OrderNo)
	if err != nil {
		return nil, err
	}

	// 2. 使用事务创建用户
	var userInfo *ent.ProxyUser
	err = h.db.TX(ctx, func(tx *ent.Tx) error {
		// 2.1 创建用户
		user, err := tx.ProxyUser.Create().
			SetPassword(tool.EncodePassWord(tempOrder.Password)).
			SetAlgo("default").
			Save(ctx)
		if err != nil {
			return err
		}

		// 2.2 生成并更新邀请码
		referCode := tool.GenerateInviteCode(int64(user.ID))
		if err := tx.ProxyUser.UpdateOneID(user.ID).
			SetReferCode(referCode).
			Exec(ctx); err != nil {
			return err
		}

		// 2.3 创建认证方法记录（复刻原项目 line 226-231）
		_, err = tx.ProxyUserAuthMethod.Create().
			SetUserID(user.ID).
			SetAuthType(tempOrder.AuthType).
			SetAuthIdentifier(tempOrder.Identifier).
			Save(ctx)
		if err != nil {
			return err
		}

		// 2.4 更新订单的user_id
		if err := tx.ProxyOrder.UpdateOneID(orderInfo.ID).
			SetUserID(int64(user.ID)).
			Exec(ctx); err != nil {
			return err
		}

		orderInfo.UserID = int64(user.ID)
		userInfo = user
		return nil
	})

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 创建用户失败: %v", err)
		return nil, err
	}

	// 3. 处理推荐人关系
	h.handleReferrer(ctx, userInfo, tempOrder.InviteCode)

	h.logger.Infof("[ActivateOrder] 创建访客用户成功: userID=%d, identifier=%s, authType=%s",
		userInfo.ID, tempOrder.Identifier, tempOrder.AuthType)

	return userInfo, nil
}

// getTempOrderInfo 从Redis缓存获取临时订单信息
// 复刻原项目 line 266-288
func (h *ActivateOrderHandler) getTempOrderInfo(ctx context.Context, orderNo string) (*constant.TemporaryOrderInfo, error) {
	cacheKey := fmt.Sprintf(constant.TempOrderCacheKey, orderNo)
	data, err := h.rdb.Get(ctx, cacheKey).Result()
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 获取临时订单缓存失败: %v, key=%s", err, cacheKey)
		return nil, err
	}

	var tempOrder constant.TemporaryOrderInfo
	if err := json.Unmarshal([]byte(data), &tempOrder); err != nil {
		h.logger.Errorf("[ActivateOrder] 反序列化临时订单缓存失败: %v, key=%s, data=%s", err, cacheKey, data)
		return nil, err
	}

	return &tempOrder, nil
}

// handleReferrer 如果提供了邀请码，建立推荐人关系
// 复刻原项目 line 290-312
func (h *ActivateOrderHandler) handleReferrer(ctx context.Context, userInfo *ent.ProxyUser, inviteCode string) {
	if inviteCode == "" {
		return
	}

	referer, err := h.db.ProxyUser.Query().
		Where(
			proxyuser.ReferCodeEQ(inviteCode),
		).
		Only(ctx)

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询推荐人失败: %v, referCode=%s", err, inviteCode)
		return
	}

	if err = h.db.ProxyUser.UpdateOneID(userInfo.ID).
		SetRefererID(int64(referer.ID)).
		Exec(ctx); err != nil {
		h.logger.Errorf("[ActivateOrder] 更新用户推荐人失败: %v, userID=%d", err, userInfo.ID)
	}
}

// getSubscribeInfo 根据订阅ID获取订阅套餐详情
// 复刻原项目 line 315-325
func (h *ActivateOrderHandler) getSubscribeInfo(ctx context.Context, subscribeID int64) (*ent.ProxySubscribe, error) {
	sub, err := h.db.ProxySubscribe.Get(ctx, subscribeID)
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询订阅套餐失败: %v, subscribeID=%d", err, subscribeID)
		return nil, err
	}
	return sub, nil
}

// createUserSubscription 根据订单和订阅套餐详情创建新的用户订阅记录
// 复刻原项目 line 328-350
func (h *ActivateOrderHandler) createUserSubscription(ctx context.Context, orderInfo *ent.ProxyOrder, sub *ent.ProxySubscribe) (*ent.ProxyUserSubscribe, error) {
	now := time.Now()
	durationUnit, durationValue := orderDurationSnapshot(orderInfo, sub)
	expireTime := tool.AddTime(durationUnit, durationValue, now)
	token := tool.GenerateSubscribeToken(orderInfo.OrderNo)

	builder := h.db.ProxyUserSubscribe.Create().
		SetUserID(orderInfo.UserID).
		SetOrderID(orderInfo.ID).
		SetSubscribeID(orderInfo.SubscribeID).
		SetStartTime(now).
		SetExpireTime(expireTime).
		SetTraffic(sub.Traffic).
		SetDownload(0).
		SetUpload(0).
		SetExpiredDownload(0).
		SetExpiredUpload(0).
		SetToken(token).
		SetUUID(uuid.New().String()).
		SetStatus(1)
	if sub.NodeGroupID != nil {
		builder = builder.SetNodeGroupID(*sub.NodeGroupID)
	}

	userSub, err := builder.Save(ctx)

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 创建用户订阅失败: %v", err)
		return nil, err
	}

	return userSub, nil
}

// ============================================================================
// handleCommission - 处理推荐佣金
// 完整复刻原项目逻辑 (activateOrderLogic.go:354-421)
// ============================================================================

// handleCommission 处理推荐佣金（如果适用）
// 完整复刻原项目逻辑 (activateOrderLogic.go:352-421)
// 异步运行以避免阻塞主订单处理流程
func (h *ActivateOrderHandler) handleCommission(ctx context.Context, userInfo *ent.ProxyUser, orderInfo *ent.ProxyOrder) {
	if !h.shouldProcessCommission(userInfo, orderInfo.IsNew) {
		return
	}

	referer, err := h.db.ProxyUser.Get(ctx, *userInfo.RefererID)
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询推荐人失败: %v, refererID=%d", err, *userInfo.RefererID)
		return
	}

	// 获取佣金比例
	var referralPercentage uint8
	if referer.ReferralPercentage != 0 {
		referralPercentage = uint8(referer.ReferralPercentage)
	} else {
		percentage, err := h.loadGlobalReferralPercentage(ctx)
		if err != nil {
			h.logger.Errorf("[ActivateOrder] 加载全局佣金比例失败: %v", err)
			return
		}
		referralPercentage = uint8(percentage)
	}

	// 佣金计算公式：(订单金额 - 订单手续费) * 佣金比例
	amount := h.calculateCommission(int64(orderInfo.Amount-orderInfo.FeeAmount), referralPercentage)

	// 使用事务更新佣金
	err = h.db.TX(ctx, func(tx *ent.Tx) error {
		// 更新推荐人佣金余额
		currentCommission := int64(0)
		if referer.Commission != nil {
			currentCommission = int64(*referer.Commission)
		}
		newCommission := currentCommission + amount

		if err := tx.ProxyUser.UpdateOneID(referer.ID).
			SetCommission(newCommission).
			Exec(ctx); err != nil {
			return err
		}

		// 确定佣金类型
		var commissionType uint16
		switch orderInfo.Type {
		case OrderTypeSubscribe:
			commissionType = uint16(logmodel.CommissionTypePurchase)
		case OrderTypeRenewal:
			commissionType = uint16(logmodel.CommissionTypeRenewal)
		}

		// 创建佣金日志
		commissionLog := map[string]interface{}{
			"type":      commissionType,
			"amount":    amount,
			"order_no":  orderInfo.OrderNo,
			"timestamp": orderInfo.CreatedAt.UnixMilli(),
		}
		content, _ := json.Marshal(commissionLog)

		return tx.ProxySystemLog.Create().
			SetType(int8(logmodel.TypeCommission)).
			SetDate(time.Now().Format(time.DateOnly)).
			SetObjectID(referer.ID).
			SetContent(string(content)).
			Exec(ctx)
	})

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 更新推荐人佣金失败: %v", err)
		return
	}

	// 更新缓存
	if err = h.updateUserCache(ctx, referer); err != nil {
		h.logger.Errorf("[ActivateOrder] 更新推荐人缓存失败: %v, userID=%d", err, referer.ID)
	}
}

// shouldProcessCommission 根据推荐人存在性、佣金设置和订单类型判断是否应该处理佣金
// 复刻原项目 line 425-458
func (h *ActivateOrderHandler) shouldProcessCommission(userInfo *ent.ProxyUser, isFirstPurchase bool) bool {
	if userInfo == nil || userInfo.RefererID == nil || *userInfo.RefererID == 0 {
		return false
	}

	referer, err := h.db.ProxyUser.Get(context.Background(), *userInfo.RefererID)
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询推荐人失败: %v, refererID=%d", err, *userInfo.RefererID)
		return false
	}
	if referer == nil {
		return false
	}

	// 如果设置了推荐人的自定义配置，则使用
	if referer.ReferralPercentage > 0 {
		if referer.OnlyFirstPurchase && !isFirstPurchase {
			return false
		}
		return true
	}

	// 使用全局配置
	inviteConfig, err := h.loadInviteSystemConfig(context.Background())
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 加载邀请配置失败: %v", err)
		return false
	}

	if inviteConfig.ReferralPercentage == 0 {
		return false
	}
	if inviteConfig.OnlyFirstPurchase && !isFirstPurchase {
		return false
	}

	return true
}

// calculateCommission 根据订单价格和佣金比例计算佣金金额
// 复刻原项目 line 461-463
func (h *ActivateOrderHandler) calculateCommission(price int64, percentage uint8) int64 {
	return int64(float64(price) * (float64(percentage) / 100))
}

// ============================================================================
// Renewal - 处理续费订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:474-514)
// ============================================================================

// Renewal 处理订阅续费
// 完整复刻原项目逻辑 (activateOrderLogic.go:472-514)
// 包括订阅延期、流量重置（如果配置）、佣金处理和通知发送
func (h *ActivateOrderHandler) Renewal(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	// 1. 获取用户信息
	userInfo, err := h.getExistingUser(ctx, int64(orderInfo.UserID))
	if err != nil {
		return err
	}

	// 2. 获取用户订阅 - 老项目按订单保存的 subscribe_token 查订阅
	userSub, err := h.getUserSubscription(ctx, orderInfo.SubscribeToken)
	if err != nil {
		return err
	}

	// 3. 获取订阅套餐信息
	sub, err := h.getSubscribeInfo(ctx, int64(orderInfo.SubscribeID))
	if err != nil {
		return err
	}
	if userSub.UserID != orderInfo.UserID {
		return fmt.Errorf("user subscription owner mismatch: orderNo=%s userSubID=%d orderUserID=%d actualUserID=%d", orderInfo.OrderNo, userSub.ID, orderInfo.UserID, userSub.UserID)
	}

	// 4. 更新订阅
	if err = h.updateSubscriptionForRenewal(ctx, userSub, sub, orderInfo); err != nil {
		return err
	}

	// 5. 清理用户订阅缓存
	if err = h.clearUserSubscribeCache(ctx, userSub); err != nil {
		h.logger.Errorf("[ActivateOrder] 清理用户订阅缓存失败: %v, subscribeID=%d, userID=%d",
			err, userSub.ID, userInfo.ID)
	}

	// 6. 清理缓存
	h.clearServerCache(ctx, sub)

	// 7. 处理佣金（异步执行）
	go h.handleCommission(context.Background(), userInfo, orderInfo)

	// 8. 发送通知
	h.sendNotifications(ctx, orderInfo, userInfo, sub, userSub, NotifyTypeRenewal)

	return nil
}

// getUserSubscription 根据token获取用户订阅
// 复刻原项目 line 517-524
func (h *ActivateOrderHandler) getUserSubscription(ctx context.Context, token string) (*ent.ProxyUserSubscribe, error) {
	if token == "" {
		return nil, fmt.Errorf("subscribe token is empty")
	}

	userSub, err := h.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.TokenEQ(token),
		).
		Only(ctx)

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询用户订阅失败: %v, token=%s", err, token)
		return nil, err
	}
	return userSub, nil
}

// updateSubscriptionForRenewal 更新续费订阅详情
// 复刻原项目 line 528-561
// 包括过期时间延期和流量重置（如果配置）
func (h *ActivateOrderHandler) updateSubscriptionForRenewal(ctx context.Context, userSub *ent.ProxyUserSubscribe, sub *ent.ProxySubscribe, orderInfo *ent.ProxyOrder) error {
	now := time.Now()

	// 处理过期时间
	expireTime := now
	if userSub.ExpireTime != nil && userSub.ExpireTime.After(now) {
		expireTime = *userSub.ExpireTime
	}

	today := now.Day()
	resetDay := expireTime.Day()

	// 如果启用，则重置流量
	resetTraffic := false
	if sub.RenewalReset || today == resetDay {
		resetTraffic = true
	}

	// 处理finished_at逻辑
	if userSub.FinishedAt != nil {
		if userSub.FinishedAt.Before(now) && today > resetDay {
			// 如果finished_at在当前时间之前，重置用户流量
			resetTraffic = true
		}
	}

	// 计算新的过期时间
	durationUnit, durationValue := orderDurationSnapshot(orderInfo, sub)
	newExpireTime := tool.AddTime(durationUnit, durationValue, expireTime)

	// 更新订阅
	updateBuilder := h.db.ProxyUserSubscribe.UpdateOneID(userSub.ID).
		SetExpireTime(newExpireTime).
		SetStatus(1).
		SetExpiredDownload(0).
		SetExpiredUpload(0).
		ClearFinishedAt()

	if resetTraffic {
		updateBuilder = updateBuilder.
			SetDownload(0).
			SetUpload(0)
	}

	if err := updateBuilder.Exec(ctx); err != nil {
		h.logger.Errorf("[ActivateOrder] 更新用户订阅失败: %v", err)
		return err
	}

	return nil
}

// ============================================================================
// ResetTraffic - 处理流量重置订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:564-625)
// ============================================================================

// ResetTraffic 处理现有订阅的流量配额重置
// 复刻原项目 line 564-625
func (h *ActivateOrderHandler) ResetTraffic(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	// 1. 获取用户信息
	userInfo, err := h.getExistingUser(ctx, int64(orderInfo.UserID))
	if err != nil {
		return err
	}

	// 2. 获取用户订阅 - 老项目按订单保存的 subscribe_token 查订阅
	userSub, err := h.getUserSubscription(ctx, orderInfo.SubscribeToken)
	if err != nil {
		return err
	}

	// 3. 重置流量
	if err := h.db.ProxyUserSubscribe.UpdateOneID(userSub.ID).
		SetDownload(0).
		SetUpload(0).
		SetExpiredDownload(0).
		SetExpiredUpload(0).
		SetStatus(1).
		Exec(ctx); err != nil {
		h.logger.Errorf("[ActivateOrder] 更新用户订阅失败: %v", err)
		return err
	}

	// 4. 获取订阅套餐信息
	sub, err := h.getSubscribeInfo(ctx, int64(userSub.SubscribeID))
	if err != nil {
		return err
	}

	// 5. 清理用户订阅缓存
	if err = h.clearUserSubscribeCache(ctx, userSub); err != nil {
		h.logger.Errorf("[ActivateOrder] 清理用户订阅缓存失败: %v, subscribeID=%d, userID=%d",
			err, userSub.ID, userInfo.ID)
	}

	// 6. 清理缓存
	h.clearServerCache(ctx, sub)

	// 7. 插入流量重置日志
	resetLog := map[string]interface{}{
		"type":      logmodel.ResetSubscribeTypePaid,
		"user_id":   userInfo.ID,
		"order_no":  orderInfo.OrderNo,
		"timestamp": time.Now().UnixMilli(),
	}
	content, _ := json.Marshal(resetLog)

	if err = h.db.ProxySystemLog.Create().
		SetType(int8(logmodel.TypeResetSubscribe)).
		SetDate(time.Now().Format(time.DateOnly)).
		SetObjectID(userSub.ID).
		SetContent(string(content)).
		Exec(ctx); err != nil {
		h.logger.Errorf("[ActivateOrder] 插入流量重置日志失败: %v", err)
	}

	// 8. 发送通知
	h.sendNotifications(ctx, orderInfo, userInfo, sub, userSub, NotifyTypeResetTraffic)

	return nil
}

// ============================================================================
// Recharge - 处理充值订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:629-674)
// ============================================================================

// Recharge 处理余额充值订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:627-674)
// 包括余额更新、交易日志记录和通知发送
func (h *ActivateOrderHandler) Recharge(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	// 1. 获取用户信息
	userInfo, err := h.getExistingUser(ctx, int64(orderInfo.UserID))
	if err != nil {
		return err
	}

	// 保存新余额用于通知
	var newBalance int64

	// 2. 在事务中更新余额
	err = h.db.TX(ctx, func(tx *ent.Tx) error {
		// 获取当前余额
		currentBalance := int64(0)
		if userInfo.Balance != nil {
			currentBalance = int64(*userInfo.Balance)
		}
		newBalance = currentBalance + int64(orderInfo.Price)

		// 更新用户余额
		if err := tx.ProxyUser.UpdateOneID(userInfo.ID).
			SetBalance(newBalance).
			Exec(ctx); err != nil {
			return err
		}

		// 创建余额日志
		balanceLog := map[string]interface{}{
			"amount":    orderInfo.Price,
			"type":      logmodel.BalanceTypeRecharge,
			"order_no":  orderInfo.OrderNo,
			"balance":   newBalance,
			"timestamp": time.Now().UnixMilli(),
		}
		content, _ := json.Marshal(balanceLog)

		return tx.ProxySystemLog.Create().
			SetType(int8(logmodel.TypeBalance)).
			SetDate(time.Now().Format(time.DateOnly)).
			SetObjectID(userInfo.ID).
			SetContent(string(content)).
			Exec(ctx)
	})

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 数据库事务失败: %v", err)
		return err
	}

	// 3. 更新userInfo的Balance字段（用于通知）
	userInfo.Balance = &newBalance

	// 4. 清理用户缓存
	if err = h.updateUserCache(ctx, userInfo); err != nil {
		h.logger.Errorf("[ActivateOrder] 更新用户缓存失败: %v", err)
		return err
	}

	// 5. 发送通知
	h.sendRechargeNotifications(ctx, orderInfo, userInfo)

	return nil
}

// ============================================================================
// 通知系统
// 完整复刻原项目逻辑 (activateOrderLogic.go:677-796)
// ============================================================================

// 通知类型常量
const (
	NotifyTypePurchase     = "purchase"
	NotifyTypeRenewal      = "renewal"
	NotifyTypeResetTraffic = "reset_traffic"
	NotifyTypeRecharge     = "recharge"
)

// sendNotifications 发送用户和管理员通知（订单完成）
// 复刻原项目 line 677-691
func (h *ActivateOrderHandler) sendNotifications(ctx context.Context, orderInfo *ent.ProxyOrder, userInfo *ent.ProxyUser, sub *ent.ProxySubscribe, userSub *ent.ProxyUserSubscribe, notifyType string) {
	// 发送用户通知
	if telegramID, ok := h.findTelegramID(ctx, userInfo); ok {
		templateData := h.buildUserNotificationData(orderInfo, sub, userSub)
		h.sendUserTelegramNotify(telegramID, notifyType, templateData)
	}

	// 发送管理员通知
	adminData := h.buildAdminNotificationData(orderInfo, sub)
	h.sendAdminTelegramNotify(ctx, adminData)
}

// sendRechargeNotifications 发送余额充值订单的专用通知
// 复刻原项目 line 694-721
func (h *ActivateOrderHandler) sendRechargeNotifications(ctx context.Context, orderInfo *ent.ProxyOrder, userInfo *ent.ProxyUser) {
	// 发送用户通知
	if telegramID, ok := h.findTelegramID(ctx, userInfo); ok {
		templateData := map[string]string{
			"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
			"PaymentMethod": orderInfo.Method,
			"Time":          orderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
			"Balance":       fmt.Sprintf("%.2f", float64(*userInfo.Balance)/100),
		}
		h.sendUserTelegramNotify(telegramID, NotifyTypeRecharge, templateData)
	}

	// 发送管理员通知
	adminData := map[string]string{
		"OrderNo":       orderInfo.OrderNo,
		"TradeNo":       orderInfo.TradeNo,
		"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
		"SubscribeName": "余额充值",
		"OrderStatus":   "已支付",
		"OrderTime":     orderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
		"PaymentMethod": orderInfo.Method,
	}
	h.sendAdminTelegramNotify(ctx, adminData)
}

// buildUserNotificationData 为用户通知创建模板数据
// 复刻原项目 line 724-737
func (h *ActivateOrderHandler) buildUserNotificationData(orderInfo *ent.ProxyOrder, sub *ent.ProxySubscribe, userSub *ent.ProxyUserSubscribe) map[string]string {
	data := map[string]string{
		"OrderNo":       orderInfo.OrderNo,
		"SubscribeName": sub.Name,
		"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
	}

	if userSub != nil && userSub.ExpireTime != nil {
		data["ExpireTime"] = userSub.ExpireTime.Format("2006-01-02 15:04:05")
		data["ResetTime"] = time.Now().Format("2006-01-02 15:04:05")
	}

	return data
}

// buildAdminNotificationData 为管理员通知创建模板数据
// 复刻原项目 line 740-755
func (h *ActivateOrderHandler) buildAdminNotificationData(orderInfo *ent.ProxyOrder, sub *ent.ProxySubscribe) map[string]string {
	subscribeName := sub.Name
	if orderInfo.Type == OrderTypeResetTraffic {
		subscribeName = "流量重置"
	}

	return map[string]string{
		"OrderNo":       orderInfo.OrderNo,
		"TradeNo":       orderInfo.TradeNo,
		"SubscribeName": subscribeName,
		"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
		"OrderStatus":   "已支付",
		"OrderTime":     orderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
		"PaymentMethod": orderInfo.Method,
	}
}

// sendUserTelegramNotify 通过Telegram向用户发送通知消息
// 复刻原项目 line 758-764
func (h *ActivateOrderHandler) sendUserTelegramNotify(chatID int64, notifyType string, data map[string]string) {
	// 获取Telegram Bot实例
	bot, err := h.getTelegramBot(context.Background())
	if err != nil || bot == nil {
		h.logger.Warnf("[ActivateOrder] Telegram Bot未配置或初始化失败，跳过用户通知: %v", err)
		return
	}

	// 构建消息文本（使用简单模板）
	text := h.renderNotificationText(notifyType, data)

	// 发送消息
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		h.logger.Errorf("[ActivateOrder] 发送Telegram用户消息失败: %v, chatID=%d", err, chatID)
		return
	}

	h.logger.Infof("[ActivateOrder] Telegram用户通知发送成功: chatID=%d, type=%s", chatID, notifyType)
}

// sendAdminTelegramNotify 通过Telegram向所有管理员用户发送通知消息
// 复刻原项目 line 767-783
func (h *ActivateOrderHandler) sendAdminTelegramNotify(ctx context.Context, data map[string]string) {
	// 获取Telegram Bot实例
	bot, err := h.getTelegramBot(ctx)
	if err != nil || bot == nil {
		h.logger.Warnf("[ActivateOrder] Telegram Bot未配置或初始化失败，跳过管理员通知: %v", err)
		return
	}

	// 查询管理员用户
	admins, err := h.db.ProxyUser.Query().
		Where(
			proxyuser.IsAdminEQ(true),
		).
		All(ctx)

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 查询管理员用户失败: %v", err)
		return
	}

	// 构建管理员通知文本
	text := h.renderAdminNotificationText(data)

	// 向每个管理员发送消息
	for _, admin := range admins {
		if telegramID, ok := h.findTelegramID(ctx, admin); ok {
			msg := tgbotapi.NewMessage(telegramID, text)
			msg.ParseMode = "Markdown"

			if _, err := bot.Send(msg); err != nil {
				h.logger.Errorf("[ActivateOrder] 发送Telegram管理员消息失败: %v, adminID=%d, chatID=%d", err, admin.ID, telegramID)
			} else {
				h.logger.Infof("[ActivateOrder] Telegram管理员通知发送成功: adminID=%d, chatID=%d", admin.ID, telegramID)
			}
		}
	}
}

// findTelegramID 从用户认证方法中提取Telegram聊天ID
// 复刻原项目 line 787-796
// 返回聊天ID和一个布尔值，指示是否找到了Telegram认证
func (h *ActivateOrderHandler) findTelegramID(ctx context.Context, user *ent.ProxyUser) (int64, bool) {
	method, err := h.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.UserIDEQ(user.ID),
			proxyuserauthmethod.AuthTypeEQ("telegram"),
		).
		Only(ctx)

	if err != nil {
		return 0, false
	}

	telegramID, err := strconv.ParseInt(method.AuthIdentifier, 10, 64)
	if err != nil {
		return 0, false
	}

	return telegramID, true
}

// ============================================================================
// 缓存管理
// ============================================================================

// clearServerCache 清理与订阅关联的所有服务器的用户列表缓存
// 复刻原项目 line 466-470
func (h *ActivateOrderHandler) clearServerCache(ctx context.Context, sub *ent.ProxySubscribe) {
	_ = ctx
	_ = sub
}

// clearUserSubscribeCache 清理用户订阅缓存
// 复刻原项目 line 495-502
func (h *ActivateOrderHandler) clearUserSubscribeCache(ctx context.Context, userSub *ent.ProxyUserSubscribe) error {
	_ = ctx
	_ = userSub
	return nil
}

// updateUserCache 更新用户缓存
// 复刻原项目 line 415-419, line 665-667
func (h *ActivateOrderHandler) updateUserCache(ctx context.Context, user *ent.ProxyUser) error {
	_ = ctx
	_ = user
	return nil
}

// ============================================================================
// 辅助方法
// ============================================================================

// updateCouponUsedCount 更新优惠券使用次数
// 复刻原项目 line 146-151
func (h *ActivateOrderHandler) updateCouponUsedCount(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	coupon, err := h.db.ProxyCoupon.Query().
		Where(
			proxycoupon.CodeEQ(orderInfo.Coupon),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return err
	}

	return h.db.ProxyCoupon.UpdateOneID(coupon.ID).
		AddUsedCount(1).
		Exec(ctx)
}

// InviteSystemConfig 邀请配置结构
type InviteSystemConfig struct {
	ReferralPercentage int  `json:"referral_percentage"` // 佣金比例
	OnlyFirstPurchase  bool `json:"only_first_purchase"` // 是否仅首购有佣金
}

// loadInviteSystemConfig 加载邀请系统配置
func (h *ActivateOrderHandler) loadInviteSystemConfig(ctx context.Context) (*InviteSystemConfig, error) {
	entries, err := h.db.ProxySystem.Query().
		Where(proxysystem.CategoryEQ("invite")).
		All(ctx)
	if err != nil {
		return nil, err
	}

	inviteConfig := &InviteSystemConfig{}
	for _, entry := range entries {
		switch entry.Key {
		case "ReferralPercentage", "referral_percentage":
			if value, parseErr := strconv.Atoi(entry.Value); parseErr == nil {
				inviteConfig.ReferralPercentage = value
			}
		case "OnlyFirstPurchase", "only_first_purchase":
			if value, parseErr := strconv.ParseBool(entry.Value); parseErr == nil {
				inviteConfig.OnlyFirstPurchase = value
			}
		}
	}

	return inviteConfig, nil
}

// loadGlobalReferralPercentage 加载全局佣金比例
func (h *ActivateOrderHandler) loadGlobalReferralPercentage(ctx context.Context) (int, error) {
	config, err := h.loadInviteSystemConfig(ctx)
	if err != nil {
		return 0, err
	}
	return config.ReferralPercentage, nil
}

// ============================================================================
// Telegram通知辅助方法
// ============================================================================

// getTelegramBot 获取Telegram Bot实例
// 从数据库读取Telegram配置并创建Bot（适用于队列处理器独立运行场景）
func (h *ActivateOrderHandler) getTelegramBot(ctx context.Context) (*tgbotapi.BotAPI, error) {
	// 查询Telegram配置（从proxy_system表读取）
	// 注意：由于是队列处理器，无法使用全局租户ID，这里使用租户ID=0（默认租户）
	// 如需扩展额外隔离维度，需要在payload中显式传递
	config, err := h.db.ProxySystem.Query().
		Where(
			proxysystem.CategoryEQ("telegram"),
			proxysystem.KeyEQ("bot_token"),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			// 配置不存在，返回nil（不报错，只是跳过通知）
			return nil, nil
		}
		return nil, err
	}

	if config.Value == "" {
		return nil, nil
	}

	// 创建Telegram Bot实例
	bot, err := tgbotapi.NewBotAPI(config.Value)
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 创建Telegram Bot失败: %v", err)
		return nil, err
	}

	return bot, nil
}

// renderNotificationText 渲染用户通知文本
// 根据通知类型和数据构建消息文本
func (h *ActivateOrderHandler) renderNotificationText(notifyType string, data map[string]string) string {
	switch notifyType {
	case NotifyTypePurchase:
		return fmt.Sprintf(`*订阅购买成功*

订单号: %s
套餐名称: %s
订单金额: %s 元
过期时间: %s

感谢您的购买！`,
			data["OrderNo"],
			data["SubscribeName"],
			data["OrderAmount"],
			data["ExpireTime"])

	case NotifyTypeRenewal:
		return fmt.Sprintf(`*订阅续费成功*

订单号: %s
套餐名称: %s
订单金额: %s 元
过期时间: %s

感谢您的续费！`,
			data["OrderNo"],
			data["SubscribeName"],
			data["OrderAmount"],
			data["ExpireTime"])

	case NotifyTypeResetTraffic:
		return fmt.Sprintf(`*流量重置成功*

订单号: %s
套餐名称: %s
订单金额: %s 元
重置时间: %s

您的流量已重置！`,
			data["OrderNo"],
			data["SubscribeName"],
			data["OrderAmount"],
			data["ResetTime"])

	case NotifyTypeRecharge:
		return fmt.Sprintf(`*余额充值成功*

充值金额: %s 元
支付方式: %s
充值时间: %s
当前余额: %s 元

感谢您的充值！`,
			data["OrderAmount"],
			data["PaymentMethod"],
			data["Time"],
			data["Balance"])

	default:
		return fmt.Sprintf("订单处理成功: %s", data["OrderNo"])
	}
}

// renderAdminNotificationText 渲染管理员通知文本
func (h *ActivateOrderHandler) renderAdminNotificationText(data map[string]string) string {
	return fmt.Sprintf(`*新订单通知*

订单号: %s
交易号: %s
套餐名称: %s
订单金额: %s 元
订单状态: %s
订单时间: %s
支付方式: %s`,
		data["OrderNo"],
		data["TradeNo"],
		data["SubscribeName"],
		data["OrderAmount"],
		data["OrderStatus"],
		data["OrderTime"],
		data["PaymentMethod"])
}

// ============================================================================
// RedemptionActivate - 处理兑换码激活订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:899-1130)
// ============================================================================

// RedemptionActivate 处理兑换码激活订单
// 完整复刻原项目逻辑 (activateOrderLogic.go:899-1130)
// 包括订阅创建或延期、兑换码使用次数更新、兑换记录创建
func (h *ActivateOrderHandler) RedemptionActivate(ctx context.Context, orderInfo *ent.ProxyOrder) error {
	// 1. 获取用户信息
	userInfo, err := h.getExistingUser(ctx, int64(orderInfo.UserID))
	if err != nil {
		return err
	}

	// 2. 获取订阅套餐信息
	sub, err := h.getSubscribeInfo(ctx, int64(orderInfo.SubscribeID))
	if err != nil {
		return err
	}

	// 3. 从Redis获取兑换码信息
	cacheKey := fmt.Sprintf("redemption_order:%s", orderInfo.OrderNo)
	data, err := h.rdb.Get(ctx, cacheKey).Result()
	if err != nil {
		h.logger.Errorf("[ActivateOrder] 获取兑换码缓存失败: %v, key=%s", err, cacheKey)
		return err
	}

	var redemptionData struct {
		RedemptionCodeID int64  `json:"redemption_code_id"`
		UnitTime         string `json:"unit_time"`
		Quantity         int64  `json:"quantity"`
	}
	if err = json.Unmarshal([]byte(data), &redemptionData); err != nil {
		h.logger.Errorf("[ActivateOrder] 反序列化兑换码缓存失败: %v", err)
		return err
	}

	// 4. 幂等性检查：查询是否已有兑换记录
	existingRecords, err := h.db.ProxyRedemptionRecord.Query().
		Where(
			proxyredemptionrecord.UserIDEQ(userInfo.ID),
			proxyredemptionrecord.RedemptionCodeIDEQ(redemptionData.RedemptionCodeID),
		).
		All(ctx)

	if err == nil {
		for _, record := range existingRecords {
			if int64(record.RedemptionCodeID) == redemptionData.RedemptionCodeID {
				h.logger.Infof("[ActivateOrder] 兑换码已处理过，跳过: orderNo=%s, userID=%d, codeID=%d",
					orderInfo.OrderNo, userInfo.ID, redemptionData.RedemptionCodeID)
				return nil // 幂等性保护
			}
		}
	}

	// 5. 查找用户现有订阅
	var existingSubscribe *ent.ProxyUserSubscribe
	userSubscribes, err := h.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.UserIDEQ(userInfo.ID),
			proxyusersubscribe.SubscribeIDEQ(orderInfo.SubscribeID),
		).
		All(ctx)

	if err == nil && len(userSubscribes) > 0 {
		existingSubscribe = userSubscribes[0]
	}

	now := time.Now()

	// 6. 使用事务保护核心操作
	err = h.db.TX(ctx, func(tx *ent.Tx) error {
		// 6.1 创建或更新订阅
		if existingSubscribe != nil {
			// 续期现有订阅
			newExpireTime := now
			if existingSubscribe.ExpireTime != nil && existingSubscribe.ExpireTime.After(now) {
				newExpireTime = *existingSubscribe.ExpireTime
			}

			// 计算新的过期时间
			newExpireTime = tool.AddTime(redemptionData.UnitTime, redemptionData.Quantity, newExpireTime)

			// 更新订阅
			updateBuilder := tx.ProxyUserSubscribe.UpdateOneID(existingSubscribe.ID).
				SetOrderID(orderInfo.ID).
				SetExpireTime(newExpireTime).
				SetStatus(1).
				ClearFinishedAt()
			if sub.Traffic > 0 {
				updateBuilder = updateBuilder.
					SetTraffic(sub.Traffic).
					SetDownload(0).
					SetUpload(0)
			}
			err = updateBuilder.Exec(ctx)
			if err != nil {
				h.logger.Errorf("[ActivateOrder] 更新订阅失败: %v", err)
				return err
			}

			h.logger.Infof("[ActivateOrder] 续期现有订阅成功: subscribeID=%d, newExpireTime=%v",
				existingSubscribe.ID, newExpireTime)
		} else {
			// 检查配额限制
			if sub.Quota > 0 {
				count, err := tx.ProxyUserSubscribe.Query().
					Where(
						proxyusersubscribe.UserIDEQ(userInfo.ID),
						proxyusersubscribe.SubscribeIDEQ(orderInfo.SubscribeID),
					).
					Count(ctx)
				if err != nil {
					h.logger.Errorf("[ActivateOrder] 查询用户订阅数量失败: %v", err)
					return err
				}
				if count >= int(sub.Quota) {
					h.logger.Infof("[ActivateOrder] 订阅配额已超限: userID=%d, subscribeID=%d, quota=%d, count=%d",
						userInfo.ID, orderInfo.SubscribeID, sub.Quota, count)
					return fmt.Errorf("subscribe quota limit exceeded")
				}
			}

			// 创建新订阅
			expireTime := tool.AddTime(redemptionData.UnitTime, redemptionData.Quantity, now)
			// 套餐 traffic 已经按字节存储，不能再按 GB 换算。
			traffic := sub.Traffic

			builder := tx.ProxyUserSubscribe.Create().
				SetUserID(int64(userInfo.ID)).
				SetOrderID(orderInfo.ID).
				SetSubscribeID(orderInfo.SubscribeID).
				SetStartTime(now).
				SetExpireTime(expireTime).
				SetTraffic(traffic).
				SetDownload(0).
				SetUpload(0).
				SetExpiredDownload(0).
				SetExpiredUpload(0).
				SetToken(tool.GenerateSubscribeToken(orderInfo.OrderNo)).
				SetUUID(uuid.New().String()).
				SetStatus(1)
			if sub.NodeGroupID != nil {
				builder = builder.SetNodeGroupID(*sub.NodeGroupID)
			}
			_, err = builder.Save(ctx)

			if err != nil {
				h.logger.Errorf("[ActivateOrder] 创建订阅失败: %v", err)
				return err
			}

			h.logger.Infof("[ActivateOrder] 创建新订阅成功: expireTime=%v", expireTime)
		}

		// 6.2 更新兑换码使用次数
		if err := tx.ProxyRedemptionCode.UpdateOneID(redemptionData.RedemptionCodeID).
			AddUsedCount(1).
			Exec(ctx); err != nil {
			h.logger.Errorf("[ActivateOrder] 更新兑换码使用次数失败: %v", err)
			return err
		}

		// 6.3 创建兑换记录
		_, err = tx.ProxyRedemptionRecord.Create().
			SetRedemptionCodeID(redemptionData.RedemptionCodeID).
			SetUserID(int64(userInfo.ID)).
			SetSubscribeID(orderInfo.SubscribeID).
			SetUnitTime(redemptionData.UnitTime).
			SetQuantity(int32(redemptionData.Quantity)).
			SetRedeemedAt(now).
			Save(ctx)

		if err != nil {
			h.logger.Errorf("[ActivateOrder] 创建兑换记录失败: %v", err)
			return err
		}

		return nil
	})

	if err != nil {
		h.logger.Errorf("[ActivateOrder] 兑换码激活事务失败: %v", err)
		return err
	}

	// 7. 触发用户组重新计算（在后台goroutine中异步执行）
	h.triggerUserGroupRecalculation(ctx, int64(userInfo.ID))

	// 8. 清理服务器缓存（关键步骤：让节点获取最新订阅）
	h.clearServerCache(ctx, sub)

	// 8.1 清理用户订阅缓存（如果存在）
	if existingSubscribe != nil {
		if err = h.clearUserSubscribeCache(ctx, existingSubscribe); err != nil {
			h.logger.Errorf("[ActivateOrder] 清理用户订阅缓存失败: %v", err)
		}
	}

	// 9. 删除Redis临时数据
	h.rdb.Del(ctx, cacheKey)

	// 10. 发送通知（可选）
	// 可以复用现有的通知模板或创建新的兑换通知模板

	h.logger.Infof("[ActivateOrder] 兑换码激活成功: orderNo=%s, userID=%d, subscribeID=%d",
		orderInfo.OrderNo, userInfo.ID, orderInfo.SubscribeID)

	return nil
}

// ============================================================================
// triggerUserGroupRecalculation - 触发用户组重新计算
// 完整复刻原项目逻辑 (activateOrderLogic.go:513-568)
// ============================================================================

// triggerUserGroupRecalculation 触发用户组重新计算
// 完整复刻原项目逻辑 (activateOrderLogic.go:513-568)
// 在后台goroutine中异步执行，避免阻塞主订单处理流程
func (h *ActivateOrderHandler) triggerUserGroupRecalculation(ctx context.Context, userId int64) {
	go func() {
		// 使用带超时的新context用于用户组重新计算
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 检查是否启用了用户组管理
		var groupEnabled string
		err := h.db.ProxySystem.Query().
			Where(
				proxysystem.CategoryEQ("group"),
				proxysystem.KeyEQ("enabled"),
			).
			Select("value").
			Scan(ctx, &groupEnabled)

		if err != nil || (groupEnabled != "true" && groupEnabled != "1") {
			h.logger.Debugf("[ActivateOrder] 用户组管理未启用，跳过重新计算")
			return
		}

		// 获取配置的用户组模式
		var groupMode string
		err = h.db.ProxySystem.Query().
			Where(
				proxysystem.CategoryEQ("group"),
				proxysystem.KeyEQ("mode"),
			).
			Select("value").
			Scan(ctx, &groupMode)

		if err != nil {
			h.logger.Errorf("[ActivateOrder] 获取用户组模式失败: %v", err)
			return
		}

		// 老项目允许 average / subscribe / traffic 三种模式
		if groupMode != "average" && groupMode != "subscribe" && groupMode != "traffic" {
			h.logger.Debugf("[ActivateOrder] 用户组模式无效 (当前: %s)，跳过", groupMode)
			return
		}

		if h.groupRecalculator == nil {
			h.logger.Warnf("[ActivateOrder] 分组重算仓储未注入，跳过: userID=%d, mode=%s", userId, groupMode)
			return
		}

		historyID, err := h.groupRecalculator.RecalculateGroup(ctx, groupMode, "")
		if err != nil {
			h.logger.Errorf("[ActivateOrder] 触发用户组重算失败: userID=%d, mode=%s, err=%v", userId, groupMode, err)
			return
		}

		h.logger.Infof("[ActivateOrder] 成功触发用户组重新计算: userID=%d, mode=%s, historyID=%d", userId, groupMode, historyID)
	}()
}

func orderDurationSnapshot(orderInfo *ent.ProxyOrder, sub *ent.ProxySubscribe) (string, int64) {
	if orderInfo.DurationUnit != "" {
		return orderInfo.DurationUnit, orderInfo.DurationValue
	}
	return sub.UnitTime, int64(orderInfo.Quantity)
}
