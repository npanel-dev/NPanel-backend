package user

import (
	"context"
)

// UserRepo Public User数据仓库接口
type UserRepo interface {
	// QueryUserInfo 查询用户信息
	// 从context获取当前用户
	// 返回用户信息，AuthIdentifier需要脱敏
	// AuthMethods需要按照email、mobile优先级排序
	QueryUserInfo(ctx context.Context, userID int) (*UserInfo, error)

	// GetTawkIdentity 获取Tawk访客身份信息和Secure Mode签名
	GetTawkIdentity(ctx context.Context, userID int) (*TawkIdentity, error)

	// GetLoginLog 获取登录日志
	// 从proxy_system_log表查询type=6(Login)的日志
	GetLoginLog(ctx context.Context, userID int, page, size int) ([]*LoginLog, int32, error)

	// QueryUserBalanceLog 查询用户余额日志
	// 从proxy_system_log表查询type=8(Balance)的日志
	// 固定查询全部，page=1, size=99999
	QueryUserBalanceLog(ctx context.Context, userID int) ([]*BalanceLog, int32, error)

	// QueryUserCommissionLog 查询用户佣金日志
	// 从proxy_system_log表查询type=9(Commission)的日志
	QueryUserCommissionLog(ctx context.Context, userID int, page, size int) ([]*CommissionLog, int32, error)

	// QueryUserAffiliate 查询用户推荐数量
	// 查询referer_id=当前用户的用户数量
	// 查询commission日志求和
	QueryUserAffiliate(ctx context.Context, userID int) (registers int64, totalCommission int64, err error)

	// QueryUserAffiliateList 查询用户推荐列表
	// 查询referer_id=当前用户的用户列表
	// 需要查询每个用户的AuthMethod并脱敏
	QueryUserAffiliateList(ctx context.Context, userID int, page, size int) ([]*UserAffiliate, int32, error)

	// GetOAuthMethods 获取OAuth方法
	// 查询当前用户的所有AuthMethods
	GetOAuthMethods(ctx context.Context, userID int) ([]*AuthMethod, error)

	// QueryUserSubscribe 查询用户订阅
	// 查询status in (0,1,2,3)的订阅
	// 需要解析Discount字段并计算ResetTime
	QueryUserSubscribe(ctx context.Context, userID int) ([]*UserSubscribe, int32, error)

	// GetSubscribeLog 获取订阅日志
	// 从proxy_system_log查询type=20(Subscribe)的日志
	GetSubscribeLog(ctx context.Context, userID int, page, size int) ([]*UserSubscribeLog, int32, error)

	// ResetUserSubscribeToken 重置订阅令牌
	// 验证UserSubscribeId属于当前用户
	// 生成新Token和UUID
	// 清除缓存
	ResetUserSubscribeToken(ctx context.Context, userID, userSubscribeID int) error

	// PreUnsubscribe 预退订
	// 调用CalculateRemainingAmount计算退款金额
	PreUnsubscribe(ctx context.Context, userID, id int) (deductionAmount int64, err error)

	// Unsubscribe 退订
	// 验证订阅状态 status in (0,1,2)
	// 使用事务处理退款逻辑
	// 清除缓存
	Unsubscribe(ctx context.Context, userID, id int) error

	// UpdateUserNotify 更新通知设置
	// 更新用户的4个通知开关
	UpdateUserNotify(ctx context.Context, userID int, enableLoginNotify, enableBalanceNotify, enableSubscribeNotify, enableTradeNotify *bool) error

	// UpdateUserPassword 更新密码
	// 使用加密函数加密密码
	UpdateUserPassword(ctx context.Context, userID int, password string) error

	// BindTelegram 绑定Telegram
	// 从JWT session获取，生成Telegram Bot URL
	// 过期时间5分钟
	BindTelegram(ctx context.Context, session string, botName string) (url string, expiredAt int64, err error)

	// UnbindTelegram 解绑Telegram
	// 查询telegram AuthMethod并删除
	// 发送Telegram通知（需要telegram bot）
	UnbindTelegram(ctx context.Context, userID int) error

	// BindOAuth 绑定OAuth
	// 根据method（google/apple/github/facebook）生成OAuth URL
	// 生成state code并存入Redis（5分钟过期）
	BindOAuth(ctx context.Context, method, redirect string) (string, error)

	// BindOAuthCallback OAuth回调
	// 验证state code
	// 交换token获取用户信息
	// 创建AuthMethod记录
	BindOAuthCallback(ctx context.Context, userID int, method string, callback string) error

	// UnbindOAuth 解绑OAuth
	// 验证method不能为email/mobile
	// 删除对应AuthMethod
	UnbindOAuth(ctx context.Context, userID int, method string) error

	// VerifyEmail 验证邮箱
	// 从Redis验证code
	// 设置AuthMethod.Verified=true
	VerifyEmail(ctx context.Context, userID int, email, code string) error

	// UpdateBindMobile 更新绑定手机
	// 验证手机验证码
	// 检查手机号是否已被其他用户绑定
	// 创建或更新mobile AuthMethod
	UpdateBindMobile(ctx context.Context, userID int, areaCode, mobile, code string) error

	// UpdateBindEmail 更新绑定邮箱
	// 检查邮箱是否已被其他用户绑定
	// 创建或更新email AuthMethod（Verified=false）
	UpdateBindEmail(ctx context.Context, userID int, email string) error

	// DeviceWSConnect 设备WebSocket连接
	// 升级WebSocket连接并添加设备到管理器
	DeviceWSConnect(ctx context.Context) error

	// GetDeviceList 获取设备列表
	// 查询用户的所有设备
	GetDeviceList(ctx context.Context, userID int) ([]*UserDevice, int32, error)

	// UnbindDevice 解绑设备
	// 验证设备属于当前用户并删除设备
	UnbindDevice(ctx context.Context, userID, deviceID int) error

	// GetDeviceOnlineStatistics 获取设备在线统计
	// 获取设备连接统计信息
	GetDeviceOnlineStatistics(ctx context.Context, userID int) (*DeviceOnlineStatistics, error)

	// UpdateUserSubscribeNote 更新用户订阅备注
	UpdateUserSubscribeNote(ctx context.Context, userID int, userSubscribeID int64, note string) error

	// UpdateUserRules 更新用户规则
	UpdateUserRules(ctx context.Context, userID int, rules []string) error

	// DeleteCurrentUserAccount 删除当前用户账号
	DeleteCurrentUserAccount(ctx context.Context, userID int, sessionID string) error

	// GetUserTrafficStats 获取用户流量统计
	GetUserTrafficStats(ctx context.Context, userID int, userSubscribeID int64, days int) (*TrafficStats, error)
}

// UserInfo 用户信息
type UserInfo struct {
	ID                    int64
	Avatar                string
	Balance               int64
	Commission            int64
	ReferralPercentage    int32
	OnlyFirstPurchase     bool
	GiftAmount            int64
	Telegram              int64
	ReferCode             string
	RefererID             int64
	Enable                bool
	IsAdmin               bool
	EnableBalanceNotify   bool
	EnableLoginNotify     bool
	EnableSubscribeNotify bool
	EnableTradeNotify     bool
	AuthMethods           []*AuthMethod
	UserDevices           []*UserDevice
	Rules                 []string
	CreatedAt             int64
	UpdatedAt             int64
	DeletedAt             int64
	IsDel                 bool
}

// TawkIdentity Tawk访客身份信息
type TawkIdentity struct {
	Name   string
	Email  string
	UserID string
	Hash   string
}

// AuthMethod 认证方法
type AuthMethod struct {
	AuthType       string
	AuthIdentifier string
	Verified       bool
}

// LoginLog 登录日志
type LoginLog struct {
	ID        int64
	UserID    int64
	LoginIP   string
	UserAgent string
	Success   bool
	Timestamp int64
}

// BalanceLog 余额日志
type BalanceLog struct {
	Type      int32
	UserID    int64
	Amount    int64
	OrderNo   string
	Balance   int64
	Timestamp int64
}

// CommissionLog 佣金日志
type CommissionLog struct {
	Type      int32
	UserID    int64
	Amount    int64
	OrderNo   string
	Timestamp int64
}

// UserAffiliate 用户推荐信息
type UserAffiliate struct {
	Avatar       string
	Identifier   string
	RegisteredAt int64
	Enable       bool
}

// UserSubscribe 用户订阅信息
type UserSubscribe struct {
	ID          int64
	UserID      int64
	OrderID     int64
	SubscribeID int64
	Subscribe   *Subscribe
	NodeGroupID int64
	GroupLocked bool
	StartTime   int64
	ExpireTime  int64
	FinishedAt  int64
	ResetTime   int64
	Traffic     int64
	Download    int64
	Upload      int64
	Token       string
	Status      int32
	Short       string
	CreatedAt   int64
	UpdatedAt   int64
}

// Subscribe 订阅套餐信息
type Subscribe struct {
	ID                int64
	Name              string
	Language          string
	Description       string
	UnitPrice         int64
	UnitTime          string
	Replacement       int64
	Inventory         int64
	Traffic           int64
	SpeedLimit        int64
	DeviceLimit       int64
	Quota             int64
	Nodes             []int64
	NodeTags          []string
	NodeGroupIDs      []int64
	NodeGroupID       int64
	TrafficLimit      []*TrafficLimit
	Show              bool
	Sell              bool
	Sort              int64
	DeductionRatio    int64
	AllowDeduction    bool
	ResetCycle        int64
	RenewalReset      bool
	ShowOriginalPrice bool
	PriceOptions      []SubscribePriceOption
	Discount          []*SubscribeDiscount
	CreatedAt         int64
	UpdatedAt         int64
}

type SubscribePriceOption struct {
	ID            int64
	SubscribeID   int64
	Code          string
	Type          string
	Name          string
	DurationUnit  string
	DurationValue int64
	Price         int64
	OriginalPrice int64
	Inventory     int64
	Show          bool
	Sell          bool
	IsDefault     bool
	Sort          int64
	CreatedAt     int64
	UpdatedAt     int64
}

// SubscribeDiscount 订阅折扣信息
type SubscribeDiscount struct {
	Quantity int64
	Discount float64
}

type TrafficLimit struct {
	StatType     string
	StatValue    int64
	TrafficUsage int64
	SpeedLimit   int64
}

// UserSubscribeLog 用户订阅日志
type UserSubscribeLog struct {
	ID              int64
	UserID          int64
	UserSubscribeID int64
	Token           string
	IP              string
	UserAgent       string
	Timestamp       int64
}

// UserDevice 用户设备信息
type UserDevice struct {
	ID         int64
	IP         string
	Identifier string
	UserAgent  string
	Online     bool
	Enabled    bool
	CreatedAt  int64
	UpdatedAt  int64
}

// WeeklyStat 每周统计
type WeeklyStat struct {
	Day     int32
	DayName string
	Hours   float64
}

// ConnectionRecords 连接记录
type ConnectionRecords struct {
	CurrentContinuousDays   int64
	HistoryContinuousDays   int64
	LongestSingleConnection int64
}

// DeviceOnlineStatistics 设备在线统计
type DeviceOnlineStatistics struct {
	WeeklyStats       []*WeeklyStat
	ConnectionRecords *ConnectionRecords
}

type DailyTrafficStats struct {
	Date     string
	Upload   int64
	Download int64
	Total    int64
}

type TrafficStats struct {
	List          []*DailyTrafficStats
	TotalUpload   int64
	TotalDownload int64
	TotalTraffic  int64
}

// UserUseCase Public User用例
type UserUseCase struct {
	repo UserRepo
}

// NewUserUseCase 创建Public User用例
func NewUserUseCase(repo UserRepo) *UserUseCase {
	return &UserUseCase{repo: repo}
}

// QueryUserInfo 查询用户信息
func (uc *UserUseCase) QueryUserInfo(ctx context.Context, userID int) (*UserInfo, error) {
	return uc.repo.QueryUserInfo(ctx, userID)
}

// GetTawkIdentity 获取Tawk访客身份信息和Secure Mode签名
func (uc *UserUseCase) GetTawkIdentity(ctx context.Context, userID int) (*TawkIdentity, error) {
	return uc.repo.GetTawkIdentity(ctx, userID)
}

// GetLoginLog 获取登录日志
func (uc *UserUseCase) GetLoginLog(ctx context.Context, userID int, page, size int) ([]*LoginLog, int32, error) {
	return uc.repo.GetLoginLog(ctx, userID, page, size)
}

// QueryUserBalanceLog 查询用户余额日志
func (uc *UserUseCase) QueryUserBalanceLog(ctx context.Context, userID int) ([]*BalanceLog, int32, error) {
	return uc.repo.QueryUserBalanceLog(ctx, userID)
}

// QueryUserCommissionLog 查询用户佣金日志
func (uc *UserUseCase) QueryUserCommissionLog(ctx context.Context, userID int, page, size int) ([]*CommissionLog, int32, error) {
	return uc.repo.QueryUserCommissionLog(ctx, userID, page, size)
}

// QueryUserAffiliate 查询用户推荐数量
func (uc *UserUseCase) QueryUserAffiliate(ctx context.Context, userID int) (registers int64, totalCommission int64, err error) {
	return uc.repo.QueryUserAffiliate(ctx, userID)
}

// QueryUserAffiliateList 查询用户推荐列表
func (uc *UserUseCase) QueryUserAffiliateList(ctx context.Context, userID int, page, size int) ([]*UserAffiliate, int32, error) {
	return uc.repo.QueryUserAffiliateList(ctx, userID, page, size)
}

// GetOAuthMethods 获取OAuth方法
func (uc *UserUseCase) GetOAuthMethods(ctx context.Context, userID int) ([]*AuthMethod, error) {
	return uc.repo.GetOAuthMethods(ctx, userID)
}

// QueryUserSubscribe 查询用户订阅
func (uc *UserUseCase) QueryUserSubscribe(ctx context.Context, userID int) ([]*UserSubscribe, int32, error) {
	return uc.repo.QueryUserSubscribe(ctx, userID)
}

// GetSubscribeLog 获取订阅日志
func (uc *UserUseCase) GetSubscribeLog(ctx context.Context, userID int, page, size int) ([]*UserSubscribeLog, int32, error) {
	return uc.repo.GetSubscribeLog(ctx, userID, page, size)
}

// ResetUserSubscribeToken 重置订阅令牌
func (uc *UserUseCase) ResetUserSubscribeToken(ctx context.Context, userID, userSubscribeID int) error {
	return uc.repo.ResetUserSubscribeToken(ctx, userID, userSubscribeID)
}

// PreUnsubscribe 预退订
func (uc *UserUseCase) PreUnsubscribe(ctx context.Context, userID, id int) (int64, error) {
	return uc.repo.PreUnsubscribe(ctx, userID, id)
}

// Unsubscribe 退订
func (uc *UserUseCase) Unsubscribe(ctx context.Context, userID, id int) error {
	return uc.repo.Unsubscribe(ctx, userID, id)
}

// UpdateUserNotify 更新通知设置
func (uc *UserUseCase) UpdateUserNotify(ctx context.Context, userID int, enableLoginNotify, enableBalanceNotify, enableSubscribeNotify, enableTradeNotify *bool) error {
	return uc.repo.UpdateUserNotify(ctx, userID, enableLoginNotify, enableBalanceNotify, enableSubscribeNotify, enableTradeNotify)
}

// UpdateUserPassword 更新密码
func (uc *UserUseCase) UpdateUserPassword(ctx context.Context, userID int, password string) error {
	return uc.repo.UpdateUserPassword(ctx, userID, password)
}

// BindTelegram 绑定Telegram
func (uc *UserUseCase) BindTelegram(ctx context.Context, session string, botName string) (string, int64, error) {
	return uc.repo.BindTelegram(ctx, session, botName)
}

// UnbindTelegram 解绑Telegram
func (uc *UserUseCase) UnbindTelegram(ctx context.Context, userID int) error {
	return uc.repo.UnbindTelegram(ctx, userID)
}

// BindOAuth 绑定OAuth
func (uc *UserUseCase) BindOAuth(ctx context.Context, method, redirect string) (string, error) {
	return uc.repo.BindOAuth(ctx, method, redirect)
}

// BindOAuthCallback OAuth回调
func (uc *UserUseCase) BindOAuthCallback(ctx context.Context, userID int, method string, callback string) error {
	return uc.repo.BindOAuthCallback(ctx, userID, method, callback)
}

// UnbindOAuth 解绑OAuth
func (uc *UserUseCase) UnbindOAuth(ctx context.Context, userID int, method string) error {
	return uc.repo.UnbindOAuth(ctx, userID, method)
}

// VerifyEmail 验证邮箱
func (uc *UserUseCase) VerifyEmail(ctx context.Context, userID int, email, code string) error {
	return uc.repo.VerifyEmail(ctx, userID, email, code)
}

// UpdateBindMobile 更新绑定手机
func (uc *UserUseCase) UpdateBindMobile(ctx context.Context, userID int, areaCode, mobile, code string) error {
	return uc.repo.UpdateBindMobile(ctx, userID, areaCode, mobile, code)
}

// UpdateBindEmail 更新绑定邮箱
func (uc *UserUseCase) UpdateBindEmail(ctx context.Context, userID int, email string) error {
	return uc.repo.UpdateBindEmail(ctx, userID, email)
}

// DeviceWSConnect 设备WebSocket连接
func (uc *UserUseCase) DeviceWSConnect(ctx context.Context) error {
	return uc.repo.DeviceWSConnect(ctx)
}

// GetDeviceList 获取设备列表
func (uc *UserUseCase) GetDeviceList(ctx context.Context, userID int) ([]*UserDevice, int32, error) {
	return uc.repo.GetDeviceList(ctx, userID)
}

// UnbindDevice 解绑设备
func (uc *UserUseCase) UnbindDevice(ctx context.Context, userID, deviceID int) error {
	return uc.repo.UnbindDevice(ctx, userID, deviceID)
}

// GetDeviceOnlineStatistics 获取设备在线统计
func (uc *UserUseCase) GetDeviceOnlineStatistics(ctx context.Context, userID int) (*DeviceOnlineStatistics, error) {
	return uc.repo.GetDeviceOnlineStatistics(ctx, userID)
}

// UpdateUserSubscribeNote 更新用户订阅备注
func (uc *UserUseCase) UpdateUserSubscribeNote(ctx context.Context, userID int, userSubscribeID int64, note string) error {
	return uc.repo.UpdateUserSubscribeNote(ctx, userID, userSubscribeID, note)
}

// UpdateUserRules 更新用户规则
func (uc *UserUseCase) UpdateUserRules(ctx context.Context, userID int, rules []string) error {
	return uc.repo.UpdateUserRules(ctx, userID, rules)
}

// DeleteCurrentUserAccount 删除当前用户账号
func (uc *UserUseCase) DeleteCurrentUserAccount(ctx context.Context, userID int, sessionID string) error {
	return uc.repo.DeleteCurrentUserAccount(ctx, userID, sessionID)
}

// GetUserTrafficStats 获取用户流量统计
func (uc *UserUseCase) GetUserTrafficStats(ctx context.Context, userID int, userSubscribeID int64, days int) (*TrafficStats, error) {
	return uc.repo.GetUserTrafficStats(ctx, userID, userSubscribeID, days)
}
