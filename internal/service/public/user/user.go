package user

import (
	"context"
	"encoding/json"
	"strconv"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/npanel-dev/NPanel-backend/api/public/user/v1"
	userBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/user"
	withdrawalBiz "github.com/npanel-dev/NPanel-backend/internal/biz/public/withdrawal"
	"github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// UserService Public User服务实现
type UserService struct {
	v1.UnimplementedPublicUserServer
	uc           *userBiz.UserUseCase
	withdrawalUc *withdrawalBiz.WithdrawalUsecase
}

// NewUserService 创建Public User服务
func NewUserService(uc *userBiz.UserUseCase, withdrawalUc *withdrawalBiz.WithdrawalUsecase) *UserService {
	return &UserService{
		uc:           uc,
		withdrawalUc: withdrawalUc,
	}
}

func parseStringID(id string) (int64, error) {
	v, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func encodeProtoValue(value *structpb.Value) (string, error) {
	if value == nil {
		return "null", nil
	}
	payload, err := json.Marshal(value.AsInterface())
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// QueryUserInfo 查询用户信息
func (s *UserService) QueryUserInfo(ctx context.Context, req *emptypb.Empty) (*v1.User, error) {
	userID := middleware.GetUserID(ctx)

	userInfo, err := s.uc.QueryUserInfo(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	authMethods := make([]*v1.UserAuthMethod, 0, len(userInfo.AuthMethods))
	for _, method := range userInfo.AuthMethods {
		authMethods = append(authMethods, &v1.UserAuthMethod{
			AuthType:       method.AuthType,
			AuthIdentifier: method.AuthIdentifier,
			Verified:       method.Verified,
		})
	}

	userDevices := make([]*v1.UserDevice, 0, len(userInfo.UserDevices))
	for _, item := range userInfo.UserDevices {
		userDevices = append(userDevices, &v1.UserDevice{
			Id:         item.ID,
			Ip:         item.IP,
			Identifier: item.Identifier,
			UserAgent:  item.UserAgent,
			Online:     item.Online,
			Enabled:    item.Enabled,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
		})
	}

	return &v1.User{
		Id:                    userInfo.ID,
		Avatar:                userInfo.Avatar,
		Balance:               userInfo.Balance,
		Commission:            userInfo.Commission,
		ReferralPercentage:    userInfo.ReferralPercentage,
		OnlyFirstPurchase:     userInfo.OnlyFirstPurchase,
		GiftAmount:            userInfo.GiftAmount,
		Telegram:              userInfo.Telegram,
		ReferCode:             userInfo.ReferCode,
		RefererId:             userInfo.RefererID,
		Enable:                userInfo.Enable,
		IsAdmin:               userInfo.IsAdmin,
		EnableBalanceNotify:   userInfo.EnableBalanceNotify,
		EnableLoginNotify:     userInfo.EnableLoginNotify,
		EnableSubscribeNotify: userInfo.EnableSubscribeNotify,
		EnableTradeNotify:     userInfo.EnableTradeNotify,
		AuthMethods:           authMethods,
		UserDevices:           userDevices,
		Rules:                 userInfo.Rules,
		CreatedAt:             userInfo.CreatedAt,
		UpdatedAt:             userInfo.UpdatedAt,
		DeletedAt:             userInfo.DeletedAt,
		IsDel:                 userInfo.IsDel,
	}, nil
}

// GetTawkIdentity 获取Tawk访客身份签名
func (s *UserService) GetTawkIdentity(ctx context.Context, req *emptypb.Empty) (*v1.TawkIdentityReply, error) {
	userID := middleware.GetUserID(ctx)

	identity, err := s.uc.GetTawkIdentity(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	return &v1.TawkIdentityReply{
		Name:   identity.Name,
		Email:  identity.Email,
		UserId: identity.UserID,
		Hash:   identity.Hash,
	}, nil
}

// GetLoginLog 获取登录日志
func (s *UserService) GetLoginLog(ctx context.Context, req *v1.GetLoginLogRequest) (*v1.GetLoginLogReply, error) {
	userID := middleware.GetUserID(ctx)

	logs, total, err := s.uc.GetLoginLog(ctx, int(userID), int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.UserLoginLog, 0, len(logs))
	for _, log := range logs {
		list = append(list, &v1.UserLoginLog{
			Id:        log.ID,
			UserId:    log.UserID,
			LoginIp:   log.LoginIP,
			UserAgent: log.UserAgent,
			Success:   log.Success,
			Timestamp: log.Timestamp,
		})
	}

	return &v1.GetLoginLogReply{List: list, Total: total}, nil
}

// QueryUserBalanceLog 查询用户余额日志
func (s *UserService) QueryUserBalanceLog(ctx context.Context, req *emptypb.Empty) (*v1.QueryUserBalanceLogReply, error) {
	userID := middleware.GetUserID(ctx)

	logs, total, err := s.uc.QueryUserBalanceLog(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.BalanceLog, 0, len(logs))
	for _, log := range logs {
		list = append(list, &v1.BalanceLog{
			Type:      log.Type,
			UserId:    log.UserID,
			Amount:    log.Amount,
			OrderNo:   log.OrderNo,
			Balance:   log.Balance,
			Timestamp: log.Timestamp,
		})
	}

	return &v1.QueryUserBalanceLogReply{List: list, Total: total}, nil
}

// QueryUserCommissionLog 查询用户佣金日志
func (s *UserService) QueryUserCommissionLog(ctx context.Context, req *v1.QueryUserCommissionLogRequest) (*v1.QueryUserCommissionLogReply, error) {
	userID := middleware.GetUserID(ctx)

	logs, total, err := s.uc.QueryUserCommissionLog(ctx, int(userID), int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.CommissionLog, 0, len(logs))
	for _, log := range logs {
		list = append(list, &v1.CommissionLog{
			Type:      log.Type,
			UserId:    log.UserID,
			Amount:    log.Amount,
			OrderNo:   log.OrderNo,
			Timestamp: log.Timestamp,
		})
	}

	return &v1.QueryUserCommissionLogReply{List: list, Total: total}, nil
}

// QueryUserAffiliate 查询用户推荐数量
func (s *UserService) QueryUserAffiliate(ctx context.Context, req *emptypb.Empty) (*v1.QueryUserAffiliateCountReply, error) {
	userID := middleware.GetUserID(ctx)

	registers, totalCommission, err := s.uc.QueryUserAffiliate(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	return &v1.QueryUserAffiliateCountReply{Registers: registers, TotalCommission: totalCommission}, nil
}

// QueryUserAffiliateList 查询用户推荐列表
func (s *UserService) QueryUserAffiliateList(ctx context.Context, req *v1.QueryUserAffiliateListRequest) (*v1.QueryUserAffiliateListReply, error) {
	userID := middleware.GetUserID(ctx)

	affiliates, total, err := s.uc.QueryUserAffiliateList(ctx, int(userID), int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.UserAffiliate, 0, len(affiliates))
	for _, affiliate := range affiliates {
		list = append(list, &v1.UserAffiliate{
			Avatar:       affiliate.Avatar,
			Identifier:   affiliate.Identifier,
			RegisteredAt: affiliate.RegisteredAt,
			Enable:       affiliate.Enable,
		})
	}

	return &v1.QueryUserAffiliateListReply{List: list, Total: total}, nil
}

// GetOAuthMethods 获取OAuth方法
func (s *UserService) GetOAuthMethods(ctx context.Context, req *emptypb.Empty) (*v1.GetOAuthMethodsReply, error) {
	userID := middleware.GetUserID(ctx)

	methods, err := s.uc.GetOAuthMethods(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.UserAuthMethod, 0, len(methods))
	for _, method := range methods {
		list = append(list, &v1.UserAuthMethod{
			AuthType:       method.AuthType,
			AuthIdentifier: method.AuthIdentifier,
			Verified:       method.Verified,
		})
	}

	return &v1.GetOAuthMethodsReply{Methods: list}, nil
}

// QueryUserSubscribe 查询用户订阅
func (s *UserService) QueryUserSubscribe(ctx context.Context, req *emptypb.Empty) (*v1.QueryUserSubscribeReply, error) {
	userID := middleware.GetUserID(ctx)

	list, total, err := s.uc.QueryUserSubscribe(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	subscribeList := make([]*v1.UserSubscribe, 0, len(list))
	for _, item := range list {
		sub := &v1.UserSubscribe{
			Id:          item.ID,
			UserId:      item.UserID,
			OrderId:     item.OrderID,
			SubscribeId: item.SubscribeID,
			StartTime:   item.StartTime,
			ExpireTime:  item.ExpireTime,
			FinishedAt:  item.FinishedAt,
			ResetTime:   item.ResetTime,
			Traffic:     item.Traffic,
			Download:    item.Download,
			Upload:      item.Upload,
			Token:       item.Token,
			Status:      item.Status,
			Short:       item.Short,
			NodeGroupId: item.NodeGroupID,
			GroupLocked: item.GroupLocked,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}

		if item.Subscribe != nil {
			sub.Subscribe = &v1.Subscribe{
				Id:                item.Subscribe.ID,
				Name:              item.Subscribe.Name,
				Language:          item.Subscribe.Language,
				Description:       item.Subscribe.Description,
				UnitPrice:         item.Subscribe.UnitPrice,
				UnitTime:          item.Subscribe.UnitTime,
				Replacement:       item.Subscribe.Replacement,
				Inventory:         int32(item.Subscribe.Inventory),
				Traffic:           item.Subscribe.Traffic,
				SpeedLimit:        int32(item.Subscribe.SpeedLimit),
				DeviceLimit:       int32(item.Subscribe.DeviceLimit),
				Quota:             int32(item.Subscribe.Quota),
				Nodes:             item.Subscribe.Nodes,
				NodeTags:          item.Subscribe.NodeTags,
				NodeGroupIds:      item.Subscribe.NodeGroupIDs,
				NodeGroupId:       item.Subscribe.NodeGroupID,
				Show:              item.Subscribe.Show,
				Sell:              item.Subscribe.Sell,
				Sort:              int32(item.Subscribe.Sort),
				DeductionRatio:    int32(item.Subscribe.DeductionRatio),
				AllowDeduction:    item.Subscribe.AllowDeduction,
				ResetCycle:        int32(item.Subscribe.ResetCycle),
				RenewalReset:      item.Subscribe.RenewalReset,
				ShowOriginalPrice: item.Subscribe.ShowOriginalPrice,
				PriceOptions:      convertUserSubscribePriceOptions(item.Subscribe.PriceOptions),
				CreatedAt:         item.Subscribe.CreatedAt,
				UpdatedAt:         item.Subscribe.UpdatedAt,
			}

			for _, limit := range item.Subscribe.TrafficLimit {
				sub.Subscribe.TrafficLimit = append(sub.Subscribe.TrafficLimit, &v1.TrafficLimit{
					StatType:     limit.StatType,
					StatValue:    limit.StatValue,
					TrafficUsage: limit.TrafficUsage,
					SpeedLimit:   int32(limit.SpeedLimit),
				})
			}
			for _, discount := range item.Subscribe.Discount {
				sub.Subscribe.Discount = append(sub.Subscribe.Discount, &v1.SubscribeDiscount{
					Quantity: discount.Quantity,
					Discount: discount.Discount,
				})
			}
		}

		subscribeList = append(subscribeList, sub)
	}

	return &v1.QueryUserSubscribeReply{List: subscribeList, Total: total}, nil
}

func convertUserSubscribePriceOptions(items []userBiz.SubscribePriceOption) []*v1.SubscribePriceOption {
	if len(items) == 0 {
		return []*v1.SubscribePriceOption{}
	}
	result := make([]*v1.SubscribePriceOption, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.SubscribePriceOption{
			Id:            item.ID,
			SubscribeId:   item.SubscribeID,
			Code:          item.Code,
			Type:          item.Type,
			Name:          item.Name,
			DurationUnit:  item.DurationUnit,
			DurationValue: item.DurationValue,
			Price:         item.Price,
			OriginalPrice: item.OriginalPrice,
			Inventory:     int32(item.Inventory),
			Show:          item.Show,
			Sell:          item.Sell,
			IsDefault:     item.IsDefault,
			Sort:          int32(item.Sort),
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return result
}

// GetSubscribeLog 获取订阅日志
func (s *UserService) GetSubscribeLog(ctx context.Context, req *v1.GetSubscribeLogRequest) (*v1.GetSubscribeLogReply, error) {
	userID := middleware.GetUserID(ctx)

	logs, total, err := s.uc.GetSubscribeLog(ctx, int(userID), int(req.Page), int(req.Size))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.UserSubscribeLog, 0, len(logs))
	for _, log := range logs {
		list = append(list, &v1.UserSubscribeLog{
			Id:              log.ID,
			UserId:          log.UserID,
			UserSubscribeId: log.UserSubscribeID,
			Token:           log.Token,
			Ip:              log.IP,
			UserAgent:       log.UserAgent,
			Timestamp:       log.Timestamp,
		})
	}

	return &v1.GetSubscribeLogReply{List: list, Total: total}, nil
}

// ResetUserSubscribeToken 重置订阅令牌
func (s *UserService) ResetUserSubscribeToken(ctx context.Context, req *v1.ResetUserSubscribeTokenRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.ResetUserSubscribeToken(ctx, int(userID), int(req.UserSubscribeId))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// PreUnsubscribe 预退订
func (s *UserService) PreUnsubscribe(ctx context.Context, req *v1.PreUnsubscribeRequest) (*v1.PreUnsubscribeReply, error) {
	userID := middleware.GetUserID(ctx)

	deductionAmount, err := s.uc.PreUnsubscribe(ctx, int(userID), int(req.Id))
	if err != nil {
		return nil, err
	}
	return &v1.PreUnsubscribeReply{DeductionAmount: deductionAmount}, nil
}

// Unsubscribe 退订
func (s *UserService) Unsubscribe(ctx context.Context, req *v1.UnsubscribeRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	err := s.uc.Unsubscribe(ctx, int(userID), int(req.Id))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// UpdateUserNotify 更新通知设置
func (s *UserService) UpdateUserNotify(ctx context.Context, req *v1.UpdateUserNotifyRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.UpdateUserNotify(ctx, int(userID), req.EnableLoginNotify, req.EnableBalanceNotify, req.EnableSubscribeNotify, req.EnableTradeNotify)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// UpdateUserPassword 更新密码
func (s *UserService) UpdateUserPassword(ctx context.Context, req *v1.UpdateUserPasswordRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.UpdateUserPassword(ctx, int(userID), req.Password)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// BindTelegram 绑定Telegram
func (s *UserService) BindTelegram(ctx context.Context, req *emptypb.Empty) (*v1.BindTelegramReply, error) {
	session := middleware.GetSessionID(ctx)
	botName := ""

	url, expiredAt, err := s.uc.BindTelegram(ctx, session, botName)
	if err != nil {
		return nil, err
	}

	return &v1.BindTelegramReply{Url: url, ExpiredAt: expiredAt}, nil
}

// UnbindTelegram 解绑Telegram
func (s *UserService) UnbindTelegram(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.UnbindTelegram(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// BindOAuth 绑定OAuth
func (s *UserService) BindOAuth(ctx context.Context, req *v1.BindOAuthRequest) (*v1.BindOAuthReply, error) {
	redirect, err := s.uc.BindOAuth(ctx, req.Method, req.Redirect)
	if err != nil {
		return nil, err
	}

	return &v1.BindOAuthReply{Redirect: redirect}, nil
}

// BindOAuthCallback OAuth回调
func (s *UserService) BindOAuthCallback(ctx context.Context, req *v1.BindOAuthCallbackRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	callback, err := encodeProtoValue(req.Callback)
	if err != nil {
		return nil, err
	}
	err = s.uc.BindOAuthCallback(ctx, int(userID), req.Method, callback)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// UnbindOAuth 解绑OAuth
func (s *UserService) UnbindOAuth(ctx context.Context, req *v1.UnbindOAuthRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.UnbindOAuth(ctx, int(userID), req.Method)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// VerifyEmail 验证邮箱
func (s *UserService) VerifyEmail(ctx context.Context, req *v1.VerifyEmailRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.VerifyEmail(ctx, int(userID), req.Email, req.Code)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// UpdateBindMobile 更新绑定手机
func (s *UserService) UpdateBindMobile(ctx context.Context, req *v1.UpdateBindMobileRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.UpdateBindMobile(ctx, int(userID), req.AreaCode, req.Mobile, req.Code)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// UpdateBindEmail 更新绑定邮箱
func (s *UserService) UpdateBindEmail(ctx context.Context, req *v1.UpdateBindEmailRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	err := s.uc.UpdateBindEmail(ctx, int(userID), req.Email)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// DeviceWSConnect 设备WebSocket连接
func (s *UserService) DeviceWSConnect(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.uc.DeviceWSConnect(ctx)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetDeviceList 获取设备列表
func (s *UserService) GetDeviceList(ctx context.Context, req *emptypb.Empty) (*v1.GetDeviceListReply, error) {
	userID := middleware.GetUserID(ctx)

	list, total, err := s.uc.GetDeviceList(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	deviceList := make([]*v1.UserDevice, 0, len(list))
	for _, device := range list {
		deviceList = append(deviceList, &v1.UserDevice{
			Id:         device.ID,
			Ip:         device.IP,
			Identifier: device.Identifier,
			UserAgent:  device.UserAgent,
			Online:     device.Online,
			Enabled:    device.Enabled,
			CreatedAt:  device.CreatedAt,
			UpdatedAt:  device.UpdatedAt,
		})
	}

	return &v1.GetDeviceListReply{List: deviceList, Total: total}, nil
}

// UnbindDevice 解绑设备
func (s *UserService) UnbindDevice(ctx context.Context, req *v1.UnbindDeviceRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	err := s.uc.UnbindDevice(ctx, int(userID), int(req.Id))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetDeviceOnlineStatistics 获取设备在线统计
func (s *UserService) GetDeviceOnlineStatistics(ctx context.Context, req *emptypb.Empty) (*v1.GetDeviceOnlineStatisticsReply, error) {
	userID := middleware.GetUserID(ctx)

	stats, err := s.uc.GetDeviceOnlineStatistics(ctx, int(userID))
	if err != nil {
		return nil, err
	}

	weeklyStats := make([]*v1.WeeklyStat, 0, len(stats.WeeklyStats))
	for _, stat := range stats.WeeklyStats {
		weeklyStats = append(weeklyStats, &v1.WeeklyStat{
			Day:     stat.Day,
			DayName: stat.DayName,
			Hours:   stat.Hours,
		})
	}

	connectionRecords := &v1.ConnectionRecords{
		CurrentContinuousDays:   stats.ConnectionRecords.CurrentContinuousDays,
		HistoryContinuousDays:   stats.ConnectionRecords.HistoryContinuousDays,
		LongestSingleConnection: stats.ConnectionRecords.LongestSingleConnection,
	}

	return &v1.GetDeviceOnlineStatisticsReply{
		WeeklyStats:       weeklyStats,
		ConnectionRecords: connectionRecords,
	}, nil
}

// CommissionWithdraw 佣金提现
func (s *UserService) CommissionWithdraw(ctx context.Context, req *v1.CommissionWithdrawRequest) (*v1.WithdrawalLog, error) {
	userID := middleware.GetUserID(ctx)
	withdrawal, err := s.withdrawalUc.CommissionWithdraw(ctx, int64(userID), &withdrawalBiz.CommissionWithdrawRequest{
		Amount:  req.Amount,
		Content: req.Content,
		Method:  req.Method,
	})
	if err != nil {
		return nil, err
	}

	return convertWithdrawalLog(withdrawal), nil
}

func (s *UserService) TransferCommissionToBalance(ctx context.Context, req *v1.TransferCommissionToBalanceRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	if err := s.withdrawalUc.TransferCommissionToBalance(ctx, int64(userID), req.Amount); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// QueryWithdrawalLog 查询提现日志
func (s *UserService) QueryWithdrawalLog(ctx context.Context, req *v1.QueryWithdrawalLogRequest) (*v1.QueryWithdrawalLogReply, error) {
	userID := middleware.GetUserID(ctx)

	withdrawals, total, err := s.withdrawalUc.QueryWithdrawalLog(ctx, int64(userID), int32(req.Page), int32(req.Size))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.WithdrawalLog, 0, len(withdrawals))
	for _, w := range withdrawals {
		list = append(list, convertWithdrawalLog(w))
	}
	return &v1.QueryWithdrawalLogReply{List: list, Total: total}, nil
}

func convertWithdrawalLog(w *withdrawalBiz.Withdrawal) *v1.WithdrawalLog {
	item := &v1.WithdrawalLog{
		Id:        w.ID,
		UserId:    w.UserID,
		Amount:    w.Amount,
		Content:   w.Content,
		Status:    int32(w.Status),
		Reason:    w.Reason,
		CreatedAt: w.CreatedAt.UnixMilli(),
		UpdatedAt: w.UpdatedAt.UnixMilli(),
		Method:    w.Method,
	}
	if w.ProcessedAt != nil {
		item.ProcessedAt = w.ProcessedAt.UnixMilli()
	}
	return item
}

// UpdateUserSubscribeNote 更新用户订阅备注
func (s *UserService) UpdateUserSubscribeNote(ctx context.Context, req *v1.UpdateUserSubscribeNoteRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	if len(req.Note) > 500 {
		return nil, strconv.ErrSyntax
	}

	if err := s.uc.UpdateUserSubscribeNote(ctx, int(userID), req.UserSubscribeId, req.Note); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// UpdateUserRules 更新用户规则
func (s *UserService) UpdateUserRules(ctx context.Context, req *v1.UpdateUserRulesRequest) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)

	if len(req.Rules) == 0 {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	if err := s.uc.UpdateUserRules(ctx, int(userID), req.Rules); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// DeleteCurrentUserAccount 删除当前用户账号
func (s *UserService) DeleteCurrentUserAccount(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	userID := middleware.GetUserID(ctx)
	sessionID := middleware.GetSessionID(ctx)

	if err := s.uc.DeleteCurrentUserAccount(ctx, int(userID), sessionID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetUserTrafficStats 获取用户流量统计
func (s *UserService) GetUserTrafficStats(ctx context.Context, req *v1.GetUserTrafficStatsRequest) (*v1.GetUserTrafficStatsReply, error) {
	userID := middleware.GetUserID(ctx)

	if req.UserSubscribeId == "" || (req.Days != 7 && req.Days != 30) {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	userSubscribeID, err := parseStringID(req.UserSubscribeId)
	if err != nil {
		return nil, err
	}

	stats, err := s.uc.GetUserTrafficStats(ctx, int(userID), userSubscribeID, int(req.Days))
	if err != nil {
		return nil, err
	}

	list := make([]*v1.DailyTrafficStats, 0, len(stats.List))
	for _, item := range stats.List {
		list = append(list, &v1.DailyTrafficStats{
			Date:     item.Date,
			Upload:   item.Upload,
			Download: item.Download,
			Total:    int32(item.Total),
		})
	}

	return &v1.GetUserTrafficStatsReply{
		List:          list,
		TotalUpload:   stats.TotalUpload,
		TotalDownload: stats.TotalDownload,
		TotalTraffic:  stats.TotalTraffic,
	}, nil
}
