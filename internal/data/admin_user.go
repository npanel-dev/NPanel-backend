package data

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/user/v1"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystemlog"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserdevice"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	userbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/user"
	logmodel "github.com/npanel-dev/NPanel-backend/internal/model/log"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/phone"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
)

type adminUserRepo struct {
	data   *Data
	logger *log.Helper
}

// NewAdminUserRepo 创建用户数据仓库
func NewAdminUserRepo(data *Data, logger log.Logger) userbiz.UserRepo {
	return &adminUserRepo{
		data:   data,
		logger: log.NewHelper(logger),
	}
}

// CreateUser 创建用户
func (r *adminUserRepo) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (int64, error) {
	req.Email = strings.TrimSpace(req.Email)
	req.Telephone = strings.TrimSpace(req.Telephone)
	req.TelephoneAreaCode = strings.TrimSpace(req.TelephoneAreaCode)
	req.RefererUser = strings.TrimSpace(req.RefererUser)

	if req.Telephone == "" {
		req.TelephoneAreaCode = ""
	} else if req.TelephoneAreaCode == "" {
		return 0, responsecode.NewKratosError(responsecode.ErrTelephoneAreaCodeIsEmpty)
	} else if !phone.Check(req.TelephoneAreaCode, req.Telephone) {
		return 0, responsecode.NewKratosError(responsecode.ErrTelephoneError)
	}

	// 生成推荐码（如果未提供）
	referCode := req.ReferCode
	if referCode == "" {
		referCode = uuidx.UserInviteCode(time.Now().UnixMicro())
	}

	// 密码处理（如果未提供，默认使用邮箱）
	password := req.Password
	if password == "" {
		password = req.Email
	}
	encodedPwd := tool.EncodePassWord(password)

	// 查找推荐人ID
	var refererID *int64
	if req.RefererUser != "" {
		// 通过推荐人邮箱查找认证方法
		authMethod, err := r.data.db.ProxyUserAuthMethod.Query().
			Where(
				proxyuserauthmethod.AuthTypeEQ("email"),
				proxyuserauthmethod.AuthIdentifierEQ(req.RefererUser),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return 0, responsecode.NewKratosError(responsecode.ErrUserNotExist)
			}
			return 0, err
		}
		refererID = new(int64)
		*refererID = authMethod.UserID
	}

	// 检查邮箱是否已存在
	if req.Email != "" {
		exists, err := r.data.db.ProxyUserAuthMethod.Query().
			Where(
				proxyuserauthmethod.AuthTypeEQ("email"),
				proxyuserauthmethod.AuthIdentifierEQ(req.Email),
			).
			Exist(ctx)
		if err != nil {
			return 0, err
		}
		if exists {
			return 0, responsecode.NewKratosError(responsecode.ErrEmailExist)
		}
	}

	// 检查手机号是否已存在
	if req.TelephoneAreaCode != "" && req.Telephone != "" {
		phone := fmt.Sprintf("%s-%s", req.TelephoneAreaCode, req.Telephone)
		exists, err := r.data.db.ProxyUserAuthMethod.Query().
			Where(
				proxyuserauthmethod.AuthTypeEQ("mobile"),
				proxyuserauthmethod.AuthIdentifierEQ(phone),
			).
			Exist(ctx)
		if err != nil {
			return 0, err
		}
		if exists {
			return 0, responsecode.NewKratosError(responsecode.ErrTelephoneExist)
		}
	}

	// 使用事务创建用户和认证方法
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return 0, err
	}

	// Numeric fields are already int64 in proto
	balance := req.Balance
	commission := req.Commission
	giftAmount := req.GiftAmount

	// 创建用户
	builder := tx.ProxyUser.Create().
		SetPassword(encodedPwd).
		SetReferCode(referCode).
		SetBalance(balance).
		SetCommission(commission).
		SetGiftAmount(giftAmount).
		SetReferralPercentage(int8(req.ReferralPercentage)).
		SetOnlyFirstPurchase(req.OnlyFirstPurchase).
		SetIsAdmin(req.IsAdmin)

	if refererID != nil {
		builder.SetRefererID(*refererID)
	}

	user, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	// 创建邮箱认证方法
	if req.Email != "" {
		err = tx.ProxyUserAuthMethod.Create().
			SetUserID(user.ID).
			SetAuthType("email").
			SetAuthIdentifier(req.Email).
			SetVerified(false).
			Exec(ctx)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}

	// 创建手机号认证方法
	if req.TelephoneAreaCode != "" && req.Telephone != "" {
		phone := fmt.Sprintf("%s-%s", req.TelephoneAreaCode, req.Telephone)
		err = tx.ProxyUserAuthMethod.Create().
			SetUserID(user.ID).
			SetAuthType("mobile").
			SetAuthIdentifier(phone).
			SetVerified(false).
			Exec(ctx)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int64(user.ID), nil
}

// DeleteUser 删除用户
func (r *adminUserRepo) DeleteUser(ctx context.Context, userID int) error {
	// Demo模式保护
	isDemo := strings.ToLower(os.Getenv("NPANEL_MODE")) == "demo"
	if userID == 2 && isDemo {
		return responsecode.NewKratosError(503) // Demo mode restriction
	}

	// 对齐旧项目：用户删除是软删除，且不存在时静默成功。
	_, err := r.data.db.ProxyUser.Update().
		Where(
			proxyuser.IDEQ(int64(userID)),
			proxyuser.DeletedAtIsNil(),
			proxyuser.Or(proxyuser.IsDelIsNil(), proxyuser.IsDelEQ(1)),
		).
		SetDeletedAt(time.Now()).
		SetIsDel(0).
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

// BatchDeleteUser 批量删除用户
func (r *adminUserRepo) BatchDeleteUser(ctx context.Context, userIDs []int) (int64, error) {
	// Demo模式保护
	isDemo := strings.ToLower(os.Getenv("NPANEL_MODE")) == "demo"
	if isDemo {
		for _, id := range userIDs {
			if id == 2 {
				return 0, responsecode.NewKratosError(503) // Demo mode restriction
			}
		}
	}

	// Convert []int to []int64 for the query
	int64IDs := make([]int64, len(userIDs))
	for i, id := range userIDs {
		int64IDs[i] = int64(id)
	}

	deleted, err := r.data.db.ProxyUser.Update().
		Where(
			proxyuser.IDIn(int64IDs...),
			proxyuser.DeletedAtIsNil(),
			proxyuser.Or(proxyuser.IsDelIsNil(), proxyuser.IsDelEQ(1)),
		).
		SetDeletedAt(time.Now()).
		SetIsDel(0).
		Save(ctx)
	if err != nil {
		return 0, err
	}

	return int64(deleted), nil
}

// GetUserByID 根据ID获取用户
func (r *adminUserRepo) GetUserByID(ctx context.Context, userID int) (*ent.ProxyUser, error) {
	user, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(int64(userID)),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrUserNotExist)
		}
		return nil, err
	}

	return user, nil
}

// GetUserList 获取用户列表
func (r *adminUserRepo) GetUserList(ctx context.Context, page, size int32, search string, userID, subscribeID, userSubscribeID *int64, unscoped bool, shortCode string) ([]*ent.ProxyUser, int32, error) {
	query := r.data.db.ProxyUser.Query()
	if !unscoped {
		query = query.Where(
			proxyuser.DeletedAtIsNil(),
			proxyuser.Or(proxyuser.IsDelIsNil(), proxyuser.IsDelEQ(1)),
		)
	}

	// 按用户ID过滤
	if userID != nil && *userID > 0 {
		query = query.Where(proxyuser.IDEQ(*userID))
	}

	// 按搜索关键字过滤（邮箱、手机号或推荐码）
	if search != "" {
		matchedIDs := map[int64]struct{}{}

		authMethods, err := r.data.db.ProxyUserAuthMethod.Query().
			Where(
				proxyuserauthmethod.AuthIdentifierContains(search),
			).
			All(ctx)
		if err != nil {
			return nil, 0, err
		}

		for _, am := range authMethods {
			matchedIDs[am.UserID] = struct{}{}
		}

		referQuery := r.data.db.ProxyUser.Query().
			Where(
				proxyuser.ReferCodeContains(search),
			)
		if !unscoped {
			referQuery = referQuery.Where(
				proxyuser.DeletedAtIsNil(),
				proxyuser.Or(proxyuser.IsDelIsNil(), proxyuser.IsDelEQ(1)),
			)
		}

		referUsers, err := referQuery.
			All(ctx)
		if err != nil {
			return nil, 0, err
		}
		for _, user := range referUsers {
			matchedIDs[user.ID] = struct{}{}
		}

		if len(matchedIDs) == 0 {
			return []*ent.ProxyUser{}, 0, nil
		}

		filterUserIDs := make([]int64, 0, len(matchedIDs))
		for id := range matchedIDs {
			filterUserIDs = append(filterUserIDs, id)
		}
		query = query.Where(proxyuser.IDIn(filterUserIDs...))
	}

	if userSubscribeID != nil && *userSubscribeID > 0 {
		userSubs, err := r.data.db.ProxyUserSubscribe.Query().
			Where(
				proxyusersubscribe.IDEQ(*userSubscribeID),
				proxyusersubscribe.StatusIn(0, 1),
			).
			All(ctx)
		if err != nil {
			return nil, 0, err
		}
		if len(userSubs) == 0 {
			return []*ent.ProxyUser{}, 0, nil
		}
		filterUserIDs := make([]int64, 0, len(userSubs))
		seen := make(map[int64]struct{}, len(userSubs))
		for _, userSub := range userSubs {
			if _, ok := seen[userSub.UserID]; ok {
				continue
			}
			seen[userSub.UserID] = struct{}{}
			filterUserIDs = append(filterUserIDs, userSub.UserID)
		}
		query = query.Where(proxyuser.IDIn(filterUserIDs...))
	}

	if subscribeID != nil && *subscribeID > 0 {
		userSubs, err := r.data.db.ProxyUserSubscribe.Query().
			Where(
				proxyusersubscribe.SubscribeIDEQ(*subscribeID),
				proxyusersubscribe.StatusIn(0, 1),
			).
			All(ctx)
		if err != nil {
			return nil, 0, err
		}
		if len(userSubs) == 0 {
			return []*ent.ProxyUser{}, 0, nil
		}
		filterUserIDs := make([]int64, 0, len(userSubs))
		seen := make(map[int64]struct{}, len(userSubs))
		for _, userSub := range userSubs {
			if _, ok := seen[userSub.UserID]; ok {
				continue
			}
			seen[userSub.UserID] = struct{}{}
			filterUserIDs = append(filterUserIDs, userSub.UserID)
		}
		query = query.Where(proxyuser.IDIn(filterUserIDs...))
	}

	if shortCode != "" {
		devices, err := r.data.db.ProxyUserDevice.Query().
			Where(proxyuserdevice.ShortCodeContains(shortCode)).
			All(ctx)
		if err != nil {
			return nil, 0, err
		}
		if len(devices) == 0 {
			return []*ent.ProxyUser{}, 0, nil
		}
		filterUserIDs := make([]int64, 0, len(devices))
		seen := make(map[int64]struct{}, len(devices))
		for _, device := range devices {
			if _, ok := seen[device.UserID]; ok {
				continue
			}
			seen[device.UserID] = struct{}{}
			filterUserIDs = append(filterUserIDs, device.UserID)
		}
		query = query.Where(proxyuser.IDIn(filterUserIDs...))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	users, err := query.
		Order(func(s *sql.Selector) {
			s.OrderBy(sql.Desc(proxyuser.FieldID))
		}).
		Offset(int((page - 1) * size)).
		Limit(int(size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, int32(total), nil
}

// UpdateUserBasicInfo 更新用户基本信息
func (r *adminUserRepo) UpdateUserBasicInfo(ctx context.Context, req *v1.UpdateUserBasicInfoRequest) error {
	userID := req.UserId
	if userID <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// 先查询用户当前信息
	userInfo, err := r.data.db.ProxyUser.Query().
		Where(
			proxyuser.IDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrUserNotExist)
		}
		return err
	}

	isDemo := strings.ToLower(os.Getenv("NPANEL_MODE")) == "demo"

	// 头像大小验证（如果提供）
	if req.Avatar != "" && !tool.IsValidImageSize(req.Avatar, 1024) {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	currentBalance := int64(0)
	if userInfo.Balance != nil {
		currentBalance = int64(*userInfo.Balance)
	}
	balance := req.Balance
	if currentBalance != balance {
		change := balance - currentBalance
		balanceLog := logmodel.Balance{
			Type:      logmodel.BalanceTypeAdjust,
			Amount:    change,
			OrderNo:   "",
			Balance:   balance,
			Timestamp: time.Now().UnixMilli(),
		}
		content, _ := balanceLog.Marshal()

		err = r.data.db.ProxySystemLog.Create().
			SetType(int8(logmodel.TypeBalance)).
			SetDate(time.Now().Format(time.DateOnly)).
			SetObjectID(userID).
			SetContent(string(content)).
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("Failed to insert balance log: %v", err)
		}
	}

	currentGiftAmount := int64(0)
	if userInfo.GiftAmount != nil {
		currentGiftAmount = int64(*userInfo.GiftAmount)
	}
	giftAmount := req.GiftAmount
	if currentGiftAmount != giftAmount {
		change := giftAmount - currentGiftAmount
		if change != 0 {
			var changeType uint16
			if currentGiftAmount < giftAmount {
				changeType = logmodel.GiftTypeIncrease
			} else {
				changeType = logmodel.GiftTypeReduce
			}
			giftLog := logmodel.Gift{
				Type:      changeType,
				Amount:    change,
				Balance:   giftAmount,
				Remark:    "Admin adjustment",
				Timestamp: time.Now().UnixMilli(),
			}
			content, _ := giftLog.Marshal()

			err = r.data.db.ProxySystemLog.Create().
				SetType(int8(logmodel.TypeGift)).
				SetDate(time.Now().Format(time.DateOnly)).
				SetObjectID(userID).
				SetContent(string(content)).
				Exec(ctx)
			if err != nil {
				r.logger.Errorf("Failed to insert gift log: %v", err)
			}
		}
	}

	currentCommission := int64(0)
	if userInfo.Commission != nil {
		currentCommission = int64(*userInfo.Commission)
	}
	commission := req.Commission
	if commission != currentCommission {
		commissionLog := logmodel.Commission{
			Type:      logmodel.CommissionTypeAdjust,
			Amount:    commission - currentCommission,
			Timestamp: time.Now().UnixMilli(),
		}
		content, _ := commissionLog.Marshal()

		err = r.data.db.ProxySystemLog.Create().
			SetType(int8(logmodel.TypeCommission)).
			SetDate(time.Now().Format(time.DateOnly)).
			SetObjectID(userID).
			SetContent(string(content)).
			Exec(ctx)
		if err != nil {
			r.logger.Errorf("Failed to insert commission log: %v", err)
		}
	}

	// 构建更新
	builder := r.data.db.ProxyUser.UpdateOneID(userID).
		SetUpdatedAt(time.Now())

	// Demo模式密码保护
	if req.Password != "" {
		if userInfo.ID == 2 && isDemo {
			return responsecode.NewKratosError(503) // Demo mode restriction
		}
		encodedPwd := tool.EncodePassWord(req.Password)
		builder.SetPassword(encodedPwd)
	}

	// Telegram field is int64 in proto
	var telegramID int64
	if req.Telegram != 0 {
		telegramID = req.Telegram
	}

	// 无条件更新所有数值字段（允许设置为0）
	builder.SetBalance(balance)
	builder.SetCommission(commission)
	builder.SetGiftAmount(giftAmount)
	if req.RefererId > 0 {
		builder.SetRefererID(req.RefererId)
	}
	builder.SetReferralPercentage(int8(req.ReferralPercentage))
	builder.SetOnlyFirstPurchase(req.OnlyFirstPurchase)
	builder.SetEnable(req.Enable)
	builder.SetIsAdmin(req.IsAdmin)
	if telegramID > 0 {
		builder.SetTelegram(telegramID)
	}

	// 字符串字段：仅在非空时更新（避免清空重要字段）
	if req.Avatar != "" {
		builder.SetAvatar(req.Avatar)
	}
	if req.ReferCode != "" {
		builder.SetReferCode(req.ReferCode)
	}

	// 执行更新
	err = builder.Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrUserNotExist)
		}
		return err
	}

	return nil
}

// UpdateUserNotifySettings 更新用户通知设置
func (r *adminUserRepo) UpdateUserNotifySettings(ctx context.Context, req *v1.UpdateUserNotifySettingsRequest) error {
	userID := req.UserId
	if userID <= 0 {
		return responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	err := r.data.db.ProxyUser.UpdateOneID(userID).
		SetEnableBalanceNotify(req.EnableBalanceNotify).
		SetEnableLoginNotify(req.EnableLoginNotify).
		SetEnableSubscribeNotify(req.EnableSubscribeNotify).
		SetEnableTradeNotify(req.EnableTradeNotify).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrUserNotExist)
		}
		return err
	}

	return nil
}

// GetUserLoginLogs 获取用户登录日志
func (r *adminUserRepo) GetUserLoginLogs(ctx context.Context, page, size int32, userID *int64, date string) ([]*ent.ProxySystemLog, int32, error) {
	query := r.data.db.ProxySystemLog.Query().
		Where(
			proxysystemlog.TypeEQ(int8(logmodel.TypeLogin)), // 登录日志类型: 30
		)

	// 按用户ID过滤
	if userID != nil && *userID > 0 {
		query = query.Where(proxysystemlog.ObjectIDEQ(*userID))
	}

	// 按日期过滤
	if date != "" {
		query = query.Where(proxysystemlog.DateEQ(date))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	logs, err := query.
		Order(func(s *sql.Selector) {
			s.OrderBy(sql.Desc(proxysystemlog.FieldCreatedAt))
		}).
		Offset(int((page - 1) * size)).
		Limit(int(size)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return logs, int32(total), nil
}

// parseDateRange 解析日期范围（YYYY-MM-DD）
func parseDateRange(date string) (time.Time, time.Time, error) {
	startTime, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	endTime := startTime.Add(24 * time.Hour).Add(-time.Second)
	return startTime, endTime, nil
}
