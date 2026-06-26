package logic

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyorder"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribepriceoption"
	modellog "github.com/npanel-dev/NPanel-backend/internal/model/log"
	"github.com/npanel-dev/NPanel-backend/internal/service"
)

// CloseOrderRequest 关闭订单请求
type CloseOrderRequest struct {
	OrderNo string `json:"orderNo" validate:"required"`
}

// CloseOrderLogic 关闭订单业务逻辑
// 完整复刻原项目：server-master/internal/logic/public/order/closeOrderLogic.go
type CloseOrderLogic struct {
	ctx          context.Context
	db           *ent.Client
	logger       *log.Helper
	cacheService *service.CacheService
	orderInfo    *ent.ProxyOrder // 保存订单信息用于缓存清理
}

// NewCloseOrderLogic 创建关闭订单业务逻辑
func NewCloseOrderLogic(ctx context.Context, db *ent.Client, logger log.Logger, cacheService *service.CacheService) *CloseOrderLogic {
	return &CloseOrderLogic{
		ctx:          ctx,
		db:           db,
		logger:       log.NewHelper(logger),
		cacheService: cacheService,
	}
}

// CloseOrder 关闭订单
// 完整复刻原项目逻辑：server-master/internal/logic/public/order/closeOrderLogic.go:36-133
func (l *CloseOrderLogic) CloseOrder(req *CloseOrderRequest) error {
	// 1. Find order information by order number (复刻原项目 line 37-45)
	orderInfo, err := l.db.ProxyOrder.Query().
		Where(
			proxyorder.OrderNoEQ(req.OrderNo),
		).
		Only(l.ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			l.logger.Warnw("[CloseOrder] Order not found", "orderNo", req.OrderNo)
			return nil
		}
		l.logger.Errorw("[CloseOrder] Find order info failed", "error", err.Error(), "orderNo", req.OrderNo)
		return nil
	}

	// 租户ID已移除，现在使用单租户架构

	// 保存订单信息供缓存清理使用
	l.orderInfo = orderInfo

	// 2. If the order status is not 1, it means that the order has been closed or paid (复刻原项目 line 46-53)
	if orderInfo.Status != 1 {
		l.logger.Infow("[CloseOrder] Order status is not 1", "orderNo", req.OrderNo, "status", orderInfo.Status)
		return nil
	}

	// 3. 使用事务处理关闭订单的完整逻辑（复刻原项目 line 54-128）
	err = l.db.TX(l.ctx, func(tx *ent.Tx) error {
		// 3.1 update order status (复刻原项目 line 55-63)
		err := tx.ProxyOrder.UpdateOneID(orderInfo.ID).
			SetStatus(3). // 原项目使用status=3表示已关闭
			Exec(l.ctx)
		if err != nil {
			l.logger.Errorw("[CloseOrder] Update order status failed", "error", err.Error(), "orderNo", req.OrderNo)
			return err
		}

		if err := l.restoreReservedInventory(tx, orderInfo); err != nil {
			return err
		}

		// 3.2 If User ID is 0, it means that the order is a guest order and does not need to be refunded, the order can be deleted directly (复刻原项目 line 64-75)
		if orderInfo.UserID == 0 {
			err = tx.ProxyOrder.DeleteOneID(orderInfo.ID).Exec(l.ctx)
			if err != nil {
				l.logger.Errorw("[CloseOrder] Delete order failed", "error", err.Error(), "orderNo", req.OrderNo)
				return err
			}
			l.logger.Infow("[CloseOrder] Guest order deleted", "orderNo", req.OrderNo)
			return nil
		}

		// 3.3 refund deduction amount to user deduction balance (复刻原项目 line 76-126)
		if orderInfo.GiftAmount > 0 {
			// 3.3.1 Find user info (复刻原项目 line 77-85)
			userInfo, err := tx.ProxyUser.Get(l.ctx, orderInfo.UserID)
			if err != nil {
				l.logger.Errorw("[CloseOrder] Find user info failed", "error", err.Error(), "user_id", orderInfo.UserID)
				return err
			}

			// 3.3.2 Calculate new gift amount balance (复刻原项目 line 86)
			currentGiftAmount := int64(0)
			if userInfo.GiftAmount != nil {
				currentGiftAmount = *userInfo.GiftAmount
			}
			newGiftAmount := currentGiftAmount + orderInfo.GiftAmount

			// 3.3.3 Refund gift amount to user account (复刻原项目 line 87-95)
			err = tx.ProxyUser.UpdateOneID(userInfo.ID).
				SetGiftAmount(newGiftAmount).
				Exec(l.ctx)
			if err != nil {
				l.logger.Errorw("[CloseOrder] Refund deduction amount failed", "error", err.Error(), "uid", orderInfo.UserID, "deduction", orderInfo.GiftAmount)
				return err
			}

			// 3.3.4 Record the deduction refund log (复刻原项目 line 96-123)
			giftLog := modellog.Gift{
				Type:        modellog.GiftTypeIncrease, // GiftTypeIncrease = 341
				OrderNo:     orderInfo.OrderNo,
				SubscribeId: 0,
				Amount:      int64(orderInfo.GiftAmount),
				Balance:     int64(newGiftAmount),
				Remark:      "Order cancellation refund",
				Timestamp:   time.Now().UnixMilli(),
			}
			content, err := giftLog.Marshal()
			if err != nil {
				l.logger.Errorw("[CloseOrder] Marshal gift log failed", "error", err.Error(), "orderNo", req.OrderNo)
				return err
			}

			err = tx.ProxySystemLog.Create().
				SetType(int8(modellog.TypeGift)). // TypeGift = 34
				SetDate(time.Now().Format(time.DateOnly)).
				SetObjectID(userInfo.ID).
				SetContent(string(content)).
				Exec(l.ctx)
			if err != nil {
				l.logger.Errorw("[CloseOrder] Record cancellation refund log failed", "error", err.Error(), "uid", orderInfo.UserID, "deduction", orderInfo.GiftAmount)
				return err
			}

			// 3.3.5 update user cache (复刻原项目 line 125)
			err = l.updateUserCache(l.ctx, userInfo)
			if err != nil {
				l.logger.Errorw("[CloseOrder] Update user cache failed", "error", err.Error(), "uid", userInfo.ID)
				return err
			}

			l.logger.Infow("[CloseOrder] Gift amount refunded", "orderNo", req.OrderNo, "amount", orderInfo.GiftAmount, "oldBalance", currentGiftAmount, "newBalance", newGiftAmount)
		}

		return nil
	})

	if err != nil {
		l.logger.Errorw("[CloseOrder] Close order transaction failed", "error", err.Error(), "orderNo", req.OrderNo)
		return err
	}

	l.logger.Infow("[CloseOrder] Order closed successfully", "orderNo", req.OrderNo)

	return nil
}

func (l *CloseOrderLogic) restoreReservedInventory(tx *ent.Tx, orderInfo *ent.ProxyOrder) error {
	if orderInfo.Type == 1 && orderInfo.SubscribeID > 0 {
		subscribeInfo, err := tx.ProxySubscribe.Query().
			Where(proxysubscribe.IDEQ(orderInfo.SubscribeID)).
			Only(l.ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				l.logger.Warnw("[CloseOrder] Subscribe not found, skip inventory restore", "subscribeID", orderInfo.SubscribeID, "orderNo", orderInfo.OrderNo)
			} else {
				l.logger.Errorw("[CloseOrder] Find subscribe failed", "error", err.Error(), "subscribeID", orderInfo.SubscribeID)
				return err
			}
		} else if subscribeInfo.Inventory != -1 {
			if err := tx.ProxySubscribe.UpdateOneID(subscribeInfo.ID).
				SetInventory(subscribeInfo.Inventory + 1).
				Exec(l.ctx); err != nil {
				l.logger.Errorw("[CloseOrder] Restore subscribe inventory failed", "error", err.Error(), "subscribeID", subscribeInfo.ID)
				return err
			}
		}
	}

	if (orderInfo.Type == 1 || orderInfo.Type == 2) && orderInfo.PriceOptionID > 0 {
		optionInfo, err := tx.ProxySubscribePriceOption.Query().
			Where(proxysubscribepriceoption.IDEQ(orderInfo.PriceOptionID)).
			Only(l.ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				l.logger.Warnw("[CloseOrder] Price option not found, skip inventory restore", "priceOptionID", orderInfo.PriceOptionID, "orderNo", orderInfo.OrderNo)
				return nil
			}
			l.logger.Errorw("[CloseOrder] Find price option failed", "error", err.Error(), "priceOptionID", orderInfo.PriceOptionID)
			return err
		}
		if optionInfo.Inventory != -1 {
			if err := tx.ProxySubscribePriceOption.UpdateOneID(optionInfo.ID).
				SetInventory(optionInfo.Inventory + 1).
				Exec(l.ctx); err != nil {
				l.logger.Errorw("[CloseOrder] Restore price option inventory failed", "error", err.Error(), "priceOptionID", optionInfo.ID)
				return err
			}
		}
	}
	return nil
}

// updateUserCache 更新用户缓存
// 复刻原项目的 UpdateUserCache 逻辑：server-master/internal/model/user/model.go:231-233
func (l *CloseOrderLogic) updateUserCache(ctx context.Context, user *ent.ProxyUser) error {
	// 原项目的 UpdateUserCache 实际上是调用 ClearUserCache
	// 在新的架构中，我们需要清理所有与用户相关的缓存

	// 使用缓存服务清理用户相关缓存
	if l.cacheService == nil {
		l.logger.Warnw("[CloseOrder] Cache service not available, skipping cache update",
			"userID", user.ID,
		)
		return nil
	}

	// 记录缓存清理操作
	l.logger.Infow("[CloseOrder] Clearing user cache",
		"userID", user.ID,
	)

	// 清理用户相关缓存
	err := l.cacheService.ClearUserCache(ctx, int64(user.ID))
	if err != nil {
		l.logger.Errorw("[CloseOrder] Failed to clear user cache",
			"userID", user.ID,
			"error", err.Error(),
		)
		return err
	}

	// 额外清理订单相关的缓存
	err = l.cacheService.ClearOrderCache(ctx, l.orderInfo.OrderNo)
	if err != nil {
		l.logger.Errorw("[CloseOrder] Failed to clear order cache",
			"orderNo", l.orderInfo.OrderNo,
			"error", err.Error(),
		)
		// 不阻断业务流程，因为用户缓存已经清理成功
	}

	l.logger.Infow("[CloseOrder] Cache cleared successfully",
		"userID", user.ID,
		"orderNo", l.orderInfo.OrderNo,
	)

	return nil
}
