package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/user/v1"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystemlog"
	"github.com/npanel-dev/NPanel-backend/ent/proxytrafficlog"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserdevice"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	userbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/user"
	logmodel "github.com/npanel-dev/NPanel-backend/internal/model/log"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/phone"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
)

type adminUserSubscribeDiscount struct {
	Quantity int64 `json:"quantity"`
	Discount int64 `json:"discount"`
}

type adminUserTrafficLimit struct {
	StatType     string `json:"stat_type"`
	StatValue    int64  `json:"stat_value"`
	TrafficUsage int64  `json:"traffic_usage"`
	SpeedLimit   int64  `json:"speed_limit"`
}

type adminUserSubscribeRepo struct {
	data   *Data
	logger *log.Helper
}

const unixMillisecondsThreshold int64 = 10_000_000_000

// resolveAdminUserSubscribeExpiry keeps the admin API compatible with both
// Unix seconds (the canonical API format) and legacy Unix milliseconds.
func resolveAdminUserSubscribeExpiry(expiredAt int64, now time.Time) (expireTime *time.Time, status int8, finishedAt *time.Time) {
	if expiredAt <= 0 {
		return nil, 1, nil
	}

	var expiry time.Time
	if expiredAt >= unixMillisecondsThreshold {
		expiry = time.UnixMilli(expiredAt)
	} else {
		expiry = time.Unix(expiredAt, 0)
	}
	if expiry.After(now) {
		return &expiry, 1, nil
	}

	finished := now
	return &expiry, 3, &finished
}

func (r *adminUserSubscribeRepo) clearUserSubscribeCaches(ctx context.Context, userSub *ent.ProxyUserSubscribe) error {
	if userSub == nil || r.data == nil || r.data.rdb == nil {
		return nil
	}

	cacheKeys := []string{
		fmt.Sprintf("cache:user:subscribe:user:%d", userSub.UserID),
		fmt.Sprintf("cache:user:subscribe:id:%d", userSub.ID),
		fmt.Sprintf("cache:subscribe:id:%d", userSub.SubscribeID),
		fmt.Sprintf("cache:subscribe:servers:%d", userSub.SubscribeID),
	}
	if userSub.Token != nil && *userSub.Token != "" {
		cacheKeys = append(cacheKeys, fmt.Sprintf("cache:user:subscribe:token:%s", *userSub.Token))
	}
	if err := r.data.rdb.Del(ctx, cacheKeys...).Err(); err != nil {
		return err
	}
	return ClearLegacyServerAllCaches(ctx, r.data.rdb)
}

// NewAdminUserSubscribeRepo creates a new admin user subscribe repository
func NewAdminUserSubscribeRepo(d *Data, logger log.Logger) userbiz.SubscribeRepo {
	return &adminUserSubscribeRepo{
		data:   d,
		logger: log.NewHelper(logger),
	}
}

func (r *adminUserSubscribeRepo) isSingleSubscribeModeEnabled(ctx context.Context) bool {
	enabled := r.data.conf != nil && r.data.conf.Subscribe != nil && r.data.conf.Subscribe.SingleModel

	values, err := loadSystemConfigMap(ctx, r.data.db, "subscribe")
	if err == nil {
		enabled = systemConfigBool(values, enabled, "SingleModel", "single_model")
	}

	return enabled
}

// GetUserSubscribe 获取用户订阅列表
func (r *adminUserSubscribeRepo) GetUserSubscribe(ctx context.Context, req *v1.GetUserSubscribeRequest) ([]*ent.ProxyUserSubscribe, int32, error) {
	query := r.data.db.ProxyUserSubscribe.Query()

	// 用户ID过滤（可选）
	if req.UserId > 0 {
		query = query.Where(proxyusersubscribe.UserIDEQ(req.UserId))
	}

	now := time.Now()
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	query = query.Where(
		proxyusersubscribe.StatusIn(0, 1, 2, 3, 4),
		proxyusersubscribe.Or(
			proxyusersubscribe.ExpireTimeGT(now),
			proxyusersubscribe.FinishedAtGTE(sevenDaysAgo),
			proxyusersubscribe.ExpireTimeEQ(time.UnixMilli(0)),
			proxyusersubscribe.ExpireTimeIsNil(),
		),
	)

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("Failed to count user subscribes: %v", err)
		return nil, 0, err
	}

	// 分页查询
	list, err := query.
		Order(ent.Desc(proxyusersubscribe.FieldCreatedAt)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribes: %v", err)
		return nil, 0, err
	}

	return list, int32(total), nil
}

// CreateUserSubscribe 创建用户订阅
func (r *adminUserSubscribeRepo) CreateUserSubscribe(ctx context.Context, req *v1.CreateUserSubscribeRequest) (int64, error) {
	userID := req.UserId
	if userID <= 0 {
		return 0, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 验证用户是否存在
	userExists, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(userID),
		).
		Exist(ctx)

	if err != nil {
		r.logger.Errorf("Failed to check user existence: %v", err)
		return 0, err
	}

	if !userExists {
		return 0, responsecode.NewKratosError(responsecode.ErrUserNotExist)
	}

	if r.isSingleSubscribeModeEnabled(ctx) {
		count, err := r.data.db.ProxyUserSubscribe.Query().
			Where(proxyusersubscribe.UserIDEQ(userID)).
			Count(ctx)
		if err != nil {
			r.logger.Errorf("Failed to count user subscribes: %v", err)
			return 0, err
		}
		if count >= 1 {
			return 0, responsecode.NewKratosError(responsecode.ErrSingleSubscribeModeExceedsLimit)
		}
	}

	subscribeID := req.SubscribeId
	if subscribeID <= 0 {
		return 0, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 验证订阅套餐是否存在
	subscribePlan, err := r.data.db.ProxySubscribe.Query().
		Where(
			proxysubscribe.IDEQ(subscribeID),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return 0, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		r.logger.Errorf("Failed to query subscribe plan: %v", err)
		return 0, err
	}

	trafficVal := req.Traffic
	if trafficVal == 0 {
		trafficVal = subscribePlan.Traffic
	}

	// 生成Token和UUID
	tokenStr := uuidx.SubscribeToken(fmt.Sprintf("adminCreate:%d", time.Now().UnixMilli()))
	subscribeUUID := uuid.New().String()

	startTime := time.Now()
	expireTime, status, finishedAt := resolveAdminUserSubscribeExpiry(req.ExpiredAt, startTime)

	create := r.data.db.ProxyUserSubscribe.Create().
		SetUserID(userID).
		SetOrderID(0).
		SetSubscribeID(subscribeID).
		SetStartTime(startTime).
		SetTraffic(trafficVal).
		SetDownload(0).
		SetUpload(0).
		SetNodeGroupID(getInt64ValueFromPointer(subscribePlan.NodeGroupID)).
		SetGroupLocked(false).
		SetToken(tokenStr).
		SetUUID(subscribeUUID).
		SetStatus(status)
	if expireTime != nil {
		create = create.SetExpireTime(*expireTime)
	}
	if finishedAt != nil {
		create = create.SetFinishedAt(*finishedAt)
	}

	created, err := create.Save(ctx)

	if err != nil {
		r.logger.Errorf("Failed to create user subscribe: %v", err)
		return 0, err
	}

	return created.ID, nil
}

// UpdateUserSubscribe 更新用户订阅
func (r *adminUserSubscribeRepo) UpdateUserSubscribe(ctx context.Context, req *v1.UpdateUserSubscribeRequest) error {
	id := req.UserSubscribeId
	if id <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 查找用户订阅
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.IDEQ(id),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		r.logger.Errorf("Failed to query user subscribe: %v", err)
		return err
	}

	now := time.Now()
	expireTime, status, finishedAt := resolveAdminUserSubscribeExpiry(req.ExpiredAt, now)

	update := userSub.Update().
		SetTraffic(req.Traffic).
		SetDownload(req.Download).
		SetUpload(req.Upload).
		SetStatus(status).
		SetUpdatedAt(now)
	if expireTime != nil {
		update.SetExpireTime(*expireTime)
	} else {
		update.ClearExpireTime()
	}
	if finishedAt != nil {
		update.SetFinishedAt(*finishedAt)
	} else {
		update.ClearFinishedAt()
	}
	if req.SubscribeId > 0 {
		update.SetSubscribeID(req.SubscribeId)
	}

	err = update.Exec(ctx)

	if err != nil {
		r.logger.Errorf("Failed to update user subscribe: %v", err)
		return err
	}

	if err := r.clearUserSubscribeCaches(ctx, userSub); err != nil {
		r.logger.Errorf("Failed to clear user subscribe caches: %v", err)
		return err
	}

	return nil
}

// DeleteUserSubscribe 删除用户订阅
func (r *adminUserSubscribeRepo) DeleteUserSubscribe(ctx context.Context, id int64) error {
	// 查找用户订阅（用于后续清除缓存）
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.IDEQ(id),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		r.logger.Errorf("Failed to query user subscribe: %v", err)
		return err
	}

	// 删除用户订阅
	err = r.data.db.ProxyUserSubscribe.DeleteOne(userSub).Exec(ctx)
	if err != nil {
		r.logger.Errorf("Failed to delete user subscribe: %v", err)
		return err
	}

	if err := r.clearUserSubscribeCaches(ctx, userSub); err != nil {
		r.logger.Errorf("Failed to clear user subscribe caches: %v", err)
		return err
	}

	return nil
}

// GetUserSubscribeById 根据ID获取用户订阅详情（包含套餐信息）
func (r *adminUserSubscribeRepo) GetUserSubscribeById(ctx context.Context, id int64) (*v1.UserSubscribeDetail, error) {
	// 查询用户订阅
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.IDEQ(id),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		r.logger.Errorf("Failed to query user subscribe: %v", err)
		return nil, err
	}

	userInfo, err := r.data.db.ProxyUser.Query().
		Where(proxyuser.IDEQ(userSub.UserID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrUserNotExist)
		}
		r.logger.Errorf("Failed to query user info: %v", err)
		return nil, err
	}

	authMethods, err := r.data.db.ProxyUserAuthMethod.Query().
		Where(proxyuserauthmethod.UserIDEQ(userSub.UserID)).
		Order(ent.Desc(proxyuserauthmethod.FieldAuthType)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query user auth methods: %v", err)
		return nil, err
	}

	userDevices, err := r.data.db.ProxyUserDevice.Query().
		Where(proxyuserdevice.UserIDEQ(userSub.UserID)).
		All(ctx)
	if err != nil {
		r.logger.Errorf("Failed to query user devices: %v", err)
		return nil, err
	}

	subscribePlan, err := r.data.db.ProxySubscribe.Query().
		Where(proxysubscribe.IDEQ(userSub.SubscribeID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		r.logger.Errorf("Failed to query subscribe plan: %v", err)
		return nil, err
	}

	protoAuthMethods := make([]*v1.UserAuthMethod, 0, len(authMethods))
	telegram := getInt64ValueFromPointer(userInfo.Telegram)
	for _, item := range authMethods {
		authIdentifier := item.AuthIdentifier
		if item.AuthType == "mobile" {
			authIdentifier = phone.FormatToInternational(authIdentifier)
		}
		if item.AuthType == "telegram" && telegram == 0 {
			if parsed, parseErr := strconv.ParseInt(item.AuthIdentifier, 10, 64); parseErr == nil {
				telegram = parsed
			}
		}
		protoAuthMethods = append(protoAuthMethods, &v1.UserAuthMethod{
			AuthType:       item.AuthType,
			AuthIdentifier: authIdentifier,
			Verified:       item.Verified,
		})
	}

	protoUserDevices := make([]*v1.UserDevice, 0, len(userDevices))
	for _, item := range userDevices {
		protoUserDevices = append(protoUserDevices, &v1.UserDevice{
			Id:         item.ID,
			Ip:         getStringValue(item.IP),
			Identifier: getStringValue(item.Identifier),
			UserAgent:  getStringValue(item.UserAgent),
			Online:     item.Online,
			Enabled:    item.Enabled,
			CreatedAt:  item.CreatedAt.Unix(),
			UpdatedAt:  item.UpdatedAt.Unix(),
		})
	}

	detail := &v1.UserSubscribeDetail{
		Id:          int64(userSub.ID),
		UserId:      int64(userSub.UserID),
		OrderId:     int64(userSub.OrderID),
		SubscribeId: int64(userSub.SubscribeID),
		NodeGroupId: int64(userSub.NodeGroupID),
		GroupLocked: userSub.GroupLocked,
		StartTime:   userSub.StartTime.Unix(),
		Traffic:     getInt64ValueFromPointer(userSub.Traffic),
		Download:    getInt64ValueFromPointer(userSub.Download),
		Upload:      getInt64ValueFromPointer(userSub.Upload),
		Token:       getStringValue(userSub.Token),
		Status:      uint32(getInt8Value(userSub.Status)),
		CreatedAt:   userSub.CreatedAt.Unix(),
		UpdatedAt:   userSub.UpdatedAt.Unix(),
		User: &v1.User{
			Id:                    userInfo.ID,
			Avatar:                getStringValue(userInfo.Avatar),
			Balance:               getInt64ValueFromPointer(userInfo.Balance),
			ReferCode:             getStringValue(userInfo.ReferCode),
			RefererId:             getInt64ValueFromPointer(userInfo.RefererID),
			Commission:            getInt64ValueFromPointer(userInfo.Commission),
			ReferralPercentage:    uint32(userInfo.ReferralPercentage),
			OnlyFirstPurchase:     userInfo.OnlyFirstPurchase,
			GiftAmount:            getInt64ValueFromPointer(userInfo.GiftAmount),
			Enable:                userInfo.Enable,
			IsAdmin:               userInfo.IsAdmin,
			EnableBalanceNotify:   userInfo.EnableBalanceNotify,
			EnableLoginNotify:     userInfo.EnableLoginNotify,
			EnableSubscribeNotify: userInfo.EnableSubscribeNotify,
			EnableTradeNotify:     userInfo.EnableTradeNotify,
			AuthMethods:           protoAuthMethods,
			UserDevices:           protoUserDevices,
			Rules:                 parseUserRulesValue(userInfo.Rules),
			CreatedAt:             userInfo.CreatedAt.Unix(),
			UpdatedAt:             userInfo.UpdatedAt.Unix(),
			Telegram:              telegram,
			DeletedAt:             timePointerToUnix(userInfo.DeletedAt),
			IsDel:                 userInfo.IsDel != nil && *userInfo.IsDel == 0,
		},
		Subscribe: &v1.Subscribe{
			Id:                subscribePlan.ID,
			Name:              subscribePlan.Name,
			Language:          subscribePlan.Language,
			Description:       getStringValue(subscribePlan.Description),
			UnitPrice:         subscribePlan.UnitPrice,
			UnitTime:          subscribePlan.UnitTime,
			Discount:          parseAdminUserSubscribeDiscounts(subscribePlan.Discount),
			Replacement:       subscribePlan.Replacement,
			Inventory:         subscribePlan.Inventory,
			Traffic:           subscribePlan.Traffic,
			SpeedLimit:        subscribePlan.SpeedLimit,
			DeviceLimit:       subscribePlan.DeviceLimit,
			Quota:             subscribePlan.Quota,
			Nodes:             parseInt64CSV(subscribePlan.Nodes),
			NodeTags:          parseStringCSV(subscribePlan.NodeTags),
			NodeGroupIds:      subscribePlan.NodeGroupIds,
			NodeGroupId:       getInt64ValueFromPointer(subscribePlan.NodeGroupID),
			TrafficLimit:      parseAdminUserTrafficLimits(subscribePlan.TrafficLimit),
			Show:              subscribePlan.Show,
			Sell:              subscribePlan.Sell,
			Sort:              int32(subscribePlan.Sort),
			DeductionRatio:    derefInt32(subscribePlan.DeductionRatio),
			AllowDeduction:    subscribePlan.AllowDeduction,
			ResetCycle:        derefInt32(subscribePlan.ResetCycle),
			RenewalReset:      subscribePlan.RenewalReset,
			ShowOriginalPrice: subscribePlan.ShowOriginalPrice,
			CreatedAt:         subscribePlan.CreatedAt.Unix(),
			UpdatedAt:         subscribePlan.UpdatedAt.Unix(),
		},
	}
	if userSub.ExpireTime != nil {
		detail.ExpireTime = userSub.ExpireTime.Unix()
	}
	return detail, nil
}

// GetUserSubscribeDevices 获取用户订阅设备列表
func (r *adminUserSubscribeRepo) GetUserSubscribeDevices(ctx context.Context, req *v1.GetUserSubscribeDevicesRequest) ([]*ent.ProxyUserDevice, int32, error) {
	query := r.data.db.ProxyUserDevice.Query()
	if req.UserId > 0 {
		query = query.Where(proxyuserdevice.UserIDEQ(req.UserId))
	}
	if req.SubscribeId > 0 {
		query = query.Where(proxyuserdevice.SubscribeIDEQ(req.SubscribeId))
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("Failed to count user subscribe devices: %v", err)
		return nil, 0, err
	}

	list, err := query.
		Order(ent.Desc(proxyuserdevice.FieldCreatedAt)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribe devices: %v", err)
		return nil, 0, err
	}

	return list, int32(total), nil
}

// GetUserSubscribeLogs 获取用户订阅日志
func (r *adminUserSubscribeRepo) GetUserSubscribeLogs(ctx context.Context, req *v1.GetUserSubscribeLogsRequest) ([]*ent.ProxySystemLog, int32, error) {
	query := r.data.db.ProxySystemLog.Query().
		Where(
			proxysystemlog.TypeEQ(int8(logmodel.TypeSubscribe)), // Type = 20
		)

	if req.Date != "" {
		query = query.Where(proxysystemlog.DateEQ(req.Date))
	}

	userSubscribeIDs := make([]int64, 0)
	if req.UserSubscribeId > 0 {
		userSubscribeIDs = append(userSubscribeIDs, req.UserSubscribeId)
	}

	if req.UserId > 0 || req.SubscribeId > 0 {
		userSubQuery := r.data.db.ProxyUserSubscribe.Query()
		if req.UserId > 0 {
			userSubQuery = userSubQuery.Where(proxyusersubscribe.UserIDEQ(req.UserId))
		}
		if req.SubscribeId > 0 {
			userSubQuery = userSubQuery.Where(proxyusersubscribe.SubscribeIDEQ(req.SubscribeId))
		}
		userSubs, err := userSubQuery.All(ctx)
		if err != nil {
			r.logger.Errorf("Failed to query user subscribes for subscribe logs: %v", err)
			return nil, 0, err
		}
		for _, item := range userSubs {
			userSubscribeIDs = append(userSubscribeIDs, item.ID)
		}
	}

	if len(userSubscribeIDs) > 0 {
		query = query.Where(proxysystemlog.ObjectIDIn(userSubscribeIDs...))
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("Failed to count user subscribe logs: %v", err)
		return nil, 0, err
	}

	// 分页查询
	list, err := query.
		Order(ent.Desc(proxysystemlog.FieldCreatedAt)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query user subscribe logs: %v", err)
		return nil, 0, err
	}

	return list, int32(total), nil
}

// GetUserSubscribeResetTrafficLogs 获取用户订阅重置流量日志
func (r *adminUserSubscribeRepo) GetUserSubscribeResetTrafficLogs(ctx context.Context, req *v1.GetUserSubscribeResetTrafficLogsRequest) ([]*ent.ProxySystemLog, int32, error) {
	query := r.data.db.ProxySystemLog.Query().
		Where(
			proxysystemlog.TypeEQ(int8(logmodel.TypeResetSubscribe)), // Type = 23
		)

	if req.UserSubscribeId > 0 {
		query = query.Where(proxysystemlog.ObjectIDEQ(req.UserSubscribeId))
	}
	if req.Date != "" {
		query = query.Where(proxysystemlog.DateEQ(req.Date))
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("Failed to count reset traffic logs: %v", err)
		return nil, 0, err
	}

	// 分页查询
	list, err := query.
		Order(ent.Desc(proxysystemlog.FieldCreatedAt)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query reset traffic logs: %v", err)
		return nil, 0, err
	}

	return list, int32(total), nil
}

// GetUserSubscribeTrafficLogs 获取用户订阅流量日志
func (r *adminUserSubscribeRepo) GetUserSubscribeTrafficLogs(ctx context.Context, req *v1.GetUserSubscribeTrafficLogsRequest) ([]*ent.ProxyTrafficLog, int32, error) {
	query := r.data.db.ProxyTrafficLog.Query()

	if req.UserSubscribeId > 0 {
		userSub, err := r.data.db.ProxyUserSubscribe.Query().
			Where(proxyusersubscribe.IDEQ(req.UserSubscribeId)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return []*ent.ProxyTrafficLog{}, 0, nil
			}
			r.logger.Errorf("Failed to query user subscribe for traffic logs: %v", err)
			return nil, 0, err
		}
		query = query.Where(
			proxytrafficlog.UserIDEQ(userSub.UserID),
			proxytrafficlog.SubscribeIDEQ(userSub.SubscribeID),
		)
	}

	// 可选：过滤特定用户
	if req.UserId > 0 {
		query = query.Where(proxytrafficlog.UserIDEQ(req.UserId))
	}

	if req.SubscribeId > 0 {
		query = query.Where(proxytrafficlog.SubscribeIDEQ(req.SubscribeId))
	}
	if req.StartTime > 0 {
		query = query.Where(proxytrafficlog.TimestampGTE(time.UnixMilli(req.StartTime)))
	}
	if req.EndTime > 0 {
		query = query.Where(proxytrafficlog.TimestampLTE(time.UnixMilli(req.EndTime)))
	}
	if req.Date != "" {
		startTime, endTime, err := parseDateRange(req.Date)
		if err != nil {
			return nil, 0, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
		}
		query = query.Where(
			proxytrafficlog.TimestampGTE(startTime),
			proxytrafficlog.TimestampLTE(endTime),
		)
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		r.logger.Errorf("Failed to count traffic logs: %v", err)
		return nil, 0, err
	}

	// 分页查询
	list, err := query.
		Order(ent.Desc(proxytrafficlog.FieldTimestamp)).
		Offset(int((req.Page - 1) * req.Size)).
		Limit(int(req.Size)).
		All(ctx)

	if err != nil {
		r.logger.Errorf("Failed to query traffic logs: %v", err)
		return nil, 0, err
	}

	return list, int32(total), nil
}

// ResetUserSubscribeToken 重置用户订阅令牌
func (r *adminUserSubscribeRepo) ResetUserSubscribeToken(ctx context.Context, userSubscribeID int64) error {
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.IDEQ(userSubscribeID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	newToken := uuidx.SubscribeToken(fmt.Sprintf("AdminUpdate:%d", time.Now().UnixMilli()))
	if err = r.data.db.ProxyUserSubscribe.UpdateOneID(userSub.ID).
		SetToken(newToken).
		SetUpdatedAt(time.Now()).
		Exec(ctx); err != nil {
		return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	return nil
}

// ToggleUserSubscribeStatus 切换用户订阅状态
func (r *adminUserSubscribeRepo) ToggleUserSubscribeStatus(ctx context.Context, userSubscribeID int64) error {
	userSub, err := r.data.db.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.IDEQ(userSubscribeID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	currentStatus := int8(0)
	if userSub.Status != nil {
		currentStatus = *userSub.Status
	}

	var nextStatus int8
	switch currentStatus {
	case 2:
		nextStatus = 5
	case 5:
		nextStatus = 2
	default:
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	if err = r.data.db.ProxyUserSubscribe.UpdateOneID(userSub.ID).
		SetStatus(nextStatus).
		SetUpdatedAt(time.Now()).
		Exec(ctx); err != nil {
		return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	return nil
}

// ResetUserSubscribeTraffic 重置用户订阅流量
func (r *adminUserSubscribeRepo) ResetUserSubscribeTraffic(ctx context.Context, userSubscribeID int64) error {
	_, err := r.data.db.ProxyUserSubscribe.Query().
		Where(proxyusersubscribe.IDEQ(userSubscribeID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrSubscribeNotFound)
		}
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	if err = r.data.db.ProxyUserSubscribe.UpdateOneID(userSubscribeID).
		SetDownload(0).
		SetUpload(0).
		SetUpdatedAt(time.Now()).
		Exec(ctx); err != nil {
		return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	return nil
}

// 辅助函数：获取int64指针的值，nil返回0
func getInt64ValueFromPointer(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// 辅助函数：获取int32指针的值并转为int64，nil返回0
func getInt64ValueFromInt32Pointer(p *int32) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}

// 辅助函数：获取int32指针的值，nil返回0
func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// 辅助函数：获取int64指针的值并格式化为字符串，nil返回"0"
func formatInt64Value(p *int64) string {
	if p == nil {
		return "0"
	}
	return strconv.FormatInt(*p, 10)
}

// 辅助函数：获取string指针的值，nil返回空字符串
func getStringValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func parseUserRulesValue(raw *string) []string {
	if raw == nil || *raw == "" {
		return nil
	}
	var rules []string
	if err := json.Unmarshal([]byte(*raw), &rules); err == nil {
		return rules
	}
	return []string{*raw}
}

func timePointerToUnix(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

// 辅助函数：获取int8指针的值，nil返回0
func getInt8Value(p *int8) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// 辅助函数：获取uint8指针的值，nil返回0
func getUint8Value(p *uint8) uint8 {
	if p == nil {
		return 0
	}
	return *p
}

func parseAdminUserSubscribeDiscounts(raw *string) []*v1.SubscribeDiscount {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var items []adminUserSubscribeDiscount
	if err := json.Unmarshal([]byte(*raw), &items); err != nil {
		return nil
	}
	result := make([]*v1.SubscribeDiscount, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.SubscribeDiscount{
			Quantity: item.Quantity,
			Discount: item.Discount,
		})
	}
	return result
}

func parseAdminUserTrafficLimits(raw *string) []*v1.TrafficLimit {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var items []adminUserTrafficLimit
	if err := json.Unmarshal([]byte(*raw), &items); err != nil {
		return nil
	}
	result := make([]*v1.TrafficLimit, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.TrafficLimit{
			StatType:     item.StatType,
			StatValue:    item.StatValue,
			TrafficUsage: item.TrafficUsage,
			SpeedLimit:   int32(item.SpeedLimit),
		})
	}
	return result
}

func parseInt64CSV(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if value, err := strconv.ParseInt(part, 10, 64); err == nil {
			result = append(result, value)
		}
	}
	return result
}

func parseStringCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
