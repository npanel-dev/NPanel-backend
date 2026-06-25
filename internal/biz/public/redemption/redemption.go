package redemption

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyorder"
	"github.com/npanel-dev/NPanel-backend/ent/proxyredemptioncode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyredemptionrecord"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/redis/go-redis/v9"
)

// RedemptionRepo 兑换码仓库接口
type RedemptionRepo interface {
	GetDB() *ent.Client
	GetRedis() *redis.Client
}

// RedemptionUseCase 兑换码用例
type RedemptionUseCase struct {
	repo   RedemptionRepo
	logger *log.Helper
}

// NewRedemptionUseCase 创建兑换码用例
func NewRedemptionUseCase(repo RedemptionRepo, logger log.Logger) *RedemptionUseCase {
	return &RedemptionUseCase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// RedeemCodeResult 兑换结果
type RedeemCodeResult struct {
	OrderNo string
	Message string
}

const (
	orderTypeRedemption  int8 = 5
	orderStatusPaid      int8 = 2
	orderStatusFinished  int8 = 5
	subscribeStatusAlive int8 = 1
)

// RedeemCode 兑换兑换码
func (uc *RedemptionUseCase) RedeemCode(ctx context.Context, userID int64, code string) (*RedeemCodeResult, error) {
	db := uc.repo.GetDB()
	redis := uc.repo.GetRedis()

	// 使用Redis分布式锁防止并发重复兑换
	lockKey := fmt.Sprintf("redemption_lock:%d:%s", userID, code)
	lockSuccess, err := redis.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if !lockSuccess {
		return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	defer redis.Del(ctx, lockKey)

	now := time.Now()
	orderNo := tool.GenerateTradeNo()
	var subscribeID int64

	err = db.TX(ctx, func(tx *ent.Tx) error {
		// 查询兑换码
		redemptionCode, err := tx.ProxyRedemptionCode.Query().
			Where(proxyredemptioncode.CodeEQ(code)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
			}
			return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}

		// 检查兑换码是否启用
		if redemptionCode.Status != 1 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}

		// 检查兑换码是否还有剩余次数
		if redemptionCode.TotalCount > 0 && redemptionCode.UsedCount >= redemptionCode.TotalCount {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}

		// 检查用户是否已经兑换过此码
		existingRecord, err := tx.ProxyRedemptionRecord.Query().
			Where(
				proxyredemptionrecord.UserIDEQ(userID),
				proxyredemptionrecord.RedemptionCodeIDEQ(redemptionCode.ID),
			).
			First(ctx)
		if err == nil && existingRecord != nil {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		if err != nil && !ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}

		// 查询订阅套餐
		subscribePlan, err := tx.ProxySubscribe.Query().
			Where(proxysubscribe.IDEQ(redemptionCode.SubscribePlan)).
			Only(ctx)
		if err != nil {
			return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}

		// 检查订阅套餐是否可售
		if !subscribePlan.Sell {
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotAvailable)
		}
		subscribeID = subscribePlan.ID

		// 如果用户已有同套餐订阅，则兑换码走延期；否则创建新订阅并检查配额。
		existingSubscribe, err := tx.ProxyUserSubscribe.Query().
			Where(
				proxyusersubscribe.UserIDEQ(userID),
				proxyusersubscribe.SubscribeIDEQ(redemptionCode.SubscribePlan),
			).
			Order(ent.Desc(proxyusersubscribe.FieldID)).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}

		if ent.IsNotFound(err) && subscribePlan.Quota > 0 {
			count, err := tx.ProxyUserSubscribe.Query().
				Where(
					proxyusersubscribe.UserIDEQ(userID),
					proxyusersubscribe.SubscribeIDEQ(redemptionCode.SubscribePlan),
				).
				Count(ctx)
			if err != nil {
				return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
			}
			if int32(count) >= subscribePlan.Quota {
				return responsecode.NewKratosError(responsecode.ErrSubscribeQuotaLimit)
			}
		}

		// 判断是否首次购买
		orderCount, err := tx.ProxyOrder.Query().
			Where(
				proxyorder.UserIDEQ(userID),
				proxyorder.StatusIn(orderStatusPaid, orderStatusFinished),
			).
			Count(ctx)
		if err != nil {
			return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
		}
		isNew := orderCount == 0

		// 创建已完成兑换订单；订阅在同一事务内同步生效。
		order, err := tx.ProxyOrder.Create().
			SetUserID(userID).
			SetOrderNo(orderNo).
			SetType(orderTypeRedemption).
			SetQuantity(redemptionCode.Quantity).
			SetPrice(0).
			SetAmount(0).
			SetDiscount(0).
			SetGiftAmount(0).
			SetCoupon("").
			SetCouponDiscount(0).
			SetPaymentID(0).
			SetMethod("redemption").
			SetFeeAmount(0).
			SetCommission(0).
			SetStatus(orderStatusFinished).
			SetSubscribeID(redemptionCode.SubscribePlan).
			SetIsNew(isNew).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			return responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
		}

		if existingSubscribe != nil {
			expireBase := now
			if existingSubscribe.ExpireTime != nil && existingSubscribe.ExpireTime.After(now) {
				expireBase = *existingSubscribe.ExpireTime
			}
			newExpireTime := tool.AddTime(redemptionCode.UnitTime, int64(redemptionCode.Quantity), expireBase)

			update := tx.ProxyUserSubscribe.UpdateOneID(existingSubscribe.ID).
				SetOrderID(order.ID).
				SetExpireTime(newExpireTime).
				SetStatus(subscribeStatusAlive).
				ClearFinishedAt()
			if subscribePlan.Traffic > 0 {
				update.SetTraffic(subscribePlan.Traffic).
					SetDownload(0).
					SetUpload(0)
			}
			if _, err := update.Save(ctx); err != nil {
				return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
			}
		} else {
			expireTime := tool.AddTime(redemptionCode.UnitTime, int64(redemptionCode.Quantity), now)
			if _, err := tx.ProxyUserSubscribe.Create().
				SetOrderID(order.ID).
				SetUserID(userID).
				SetSubscribeID(redemptionCode.SubscribePlan).
				SetStartTime(now).
				SetExpireTime(expireTime).
				SetTraffic(subscribePlan.Traffic).
				SetDownload(0).
				SetUpload(0).
				SetExpiredDownload(0).
				SetExpiredUpload(0).
				SetToken(tool.GenerateSubscribeToken(orderNo)).
				SetUUID(uuid.New().String()).
				SetStatus(subscribeStatusAlive).
				Save(ctx); err != nil {
				return responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
			}
		}

		redemptionUpdate := tx.ProxyRedemptionCode.Update().
			Where(proxyredemptioncode.IDEQ(redemptionCode.ID))
		if redemptionCode.TotalCount > 0 {
			redemptionUpdate.Where(proxyredemptioncode.UsedCountLT(redemptionCode.TotalCount))
		}
		affected, err := redemptionUpdate.AddUsedCount(1).Save(ctx)
		if err != nil {
			return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
		}
		if affected == 0 {
			return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}

		if _, err := tx.ProxyRedemptionRecord.Create().
			SetRedemptionCodeID(redemptionCode.ID).
			SetUserID(userID).
			SetSubscribeID(redemptionCode.SubscribePlan).
			SetUnitTime(redemptionCode.UnitTime).
			SetQuantity(redemptionCode.Quantity).
			SetRedeemedAt(now).
			SetCreatedAt(now).
			Save(ctx); err != nil {
			return responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	uc.logger.Infof("Redemption completed: order_no=%s, user_id=%d, subscribe_id=%d", orderNo, userID, subscribeID)

	return &RedeemCodeResult{
		OrderNo: orderNo,
		Message: "兑换成功",
	}, nil
}
