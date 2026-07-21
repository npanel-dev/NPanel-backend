package responsecode

// 兼容 npanel-server 的响应码表
// 保持当前 responsecode 实现方式，但返回码与旧项目完全一致

const (
	// ==== 成功码（与旧项目一致，统一返回 200） ====

	// 用户管理相关
	UserCreated           = 200 // 用户创建成功
	UserUpdated           = 200 // 用户更新成功
	UserDeleted           = 200 // 用户删除成功
	UserListRetrieved     = 200 // 用户列表获取成功
	UserDetailRetrieved   = 200 // 用户详情获取成功
	PasswordUpdated       = 200 // 密码更新成功
	NotifySettingsUpdated = 200 // 通知设置更新成功

	// 订阅管理相关 (2006003100-2006003199)

	SubscribeCreated         = 200 // 订阅创建成功
	SubscribeUpdated         = 200 // 订阅更新成功
	SubscribeDeleted         = 200 // 订阅删除成功
	SubscribeListRetrieved   = 200 // 订阅列表获取成功
	SubscribeDetailRetrieved = 200 // 订阅详情获取成功
	SubscribeSorted          = 200 // 订阅排序成功

	// 订单管理相关 (2006003200-2006003299)
	OrderCreated         = 200 // 订单创建成功
	OrderUpdated         = 200 // 订单更新成功
	OrderDeleted         = 200 // 订单删除成功
	OrderListRetrieved   = 200 // 订单列表获取成功
	OrderDetailRetrieved = 200 // 订单详情获取成功
	OrderCancelled       = 200 // 订单取消成功
	OrderCompleted       = 200 // 订单完成成功

	// 支付管理相关 (2006003300-2006003399)
	PaymentCreated         = 200 // 支付方式创建成功
	PaymentUpdated         = 200 // 支付方式更新成功
	PaymentDeleted         = 200 // 支付方式删除成功
	PaymentListRetrieved   = 200 // 支付方式列表获取成功
	PaymentDetailRetrieved = 200 // 支付方式详情获取成功
	PaymentToggled         = 200 // 支付方式状态切换成功

	// 服务器管理相关 (2006003400-2006003499)
	ServerCreated         = 200 // 服务器创建成功
	ServerUpdated         = 200 // 服务器更新成功
	ServerDeleted         = 200 // 服务器删除成功
	ServerListRetrieved   = 200 // 服务器列表获取成功
	ServerDetailRetrieved = 200 // 服务器详情获取成功
	ServerToggled         = 200 // 服务器状态切换成功
	ServerSortUpdated     = 200 // 服务器排序更新成功

	// 节点管理相关 (2006003500-2006003599)
	NodeCreated         = 200 // 节点创建成功
	NodeUpdated         = 200 // 节点更新成功
	NodeDeleted         = 200 // 节点删除成功
	NodeListRetrieved   = 200 // 节点列表获取成功
	NodeDetailRetrieved = 200 // 节点详情获取成功
	NodeToggled         = 200 // 节点状态切换成功
	NodeSortUpdated     = 200 // 节点排序更新成功

	// 优惠券管理相关 (2006003600-2006003699)
	CouponCreated         = 200 // 优惠券创建成功
	CouponUpdated         = 200 // 优惠券更新成功
	CouponDeleted         = 200 // 优惠券删除成功
	CouponListRetrieved   = 200 // 优惠券列表获取成功
	CouponDetailRetrieved = 200 // 优惠券详情获取成功
	CouponValidated       = 200 // 优惠券验证成功
	CouponUsed            = 200 // 优惠券使用成功
	DiscountCalculated    = 200 // 折扣计算成功

	// 工单管理相关 (2006003700-2006003799)
	TicketCreated         = 200 // 工单创建成功
	TicketUpdated         = 200 // 工单更新成功
	TicketDeleted         = 200 // 工单删除成功
	TicketListRetrieved   = 200 // 工单列表获取成功
	TicketDetailRetrieved = 200 // 工单详情获取成功
	TicketStatusUpdated   = 200 // 工单状态更新成功

	// 系统日志管理相关 (2006003800-2006003899)
	SystemLogCreated         = 200 // 系统日志创建成功
	SystemLogDeleted         = 200 // 系统日志删除成功
	SystemLogListRetrieved   = 200 // 系统日志列表获取成功
	SystemLogDetailRetrieved = 200 // 系统日志详情获取成功

	// 认证管理相关 (2006070-2006079)
	UserCheckSuccess     = 200 // 用户检查成功
	UserLoginSuccess     = 200 // 用户登录成功
	UserRegisterSuccess  = 200 // 用户注册成功
	PasswordResetSuccess = 200 // 密码重置成功

	// Public User 相关 (2006080-2006101)
	UserInfoQuerySuccess       = 200 // 查询用户信息成功
	LoginLogQuerySuccess       = 200 // 查询登录日志成功
	BalanceLogQuerySuccess     = 200 // 查询余额日志成功
	CommissionLogQuerySuccess  = 200 // 查询佣金日志成功
	AffiliateQuerySuccess      = 200 // 查询推荐成功
	AffiliateListQuerySuccess  = 200 // 查询推荐列表成功
	OAuthMethodsQuerySuccess   = 200 // 查询OAuth方法成功
	UserSubscribeQuerySuccess  = 200 // 查询用户订阅成功
	SubscribeLogQuerySuccess   = 200 // 查询订阅日志成功
	SubscribeTokenResetSuccess = 200 // 重置订阅令牌成功
	PreUnsubscribeSuccess      = 200 // 预退订成功
	UnsubscribeSuccess         = 200 // 退订成功
	NotifyUpdateSuccess        = 200 // 更新通知设置成功
	PasswordUpdateSuccess      = 200 // 更新密码成功
	TelegramBindSuccess        = 200 // 绑定Telegram成功
	TelegramUnbindSuccess      = 200 // 解绑Telegram成功
	OAuthBindSuccess           = 200 // 绑定OAuth成功
	OAuthCallbackSuccess       = 200 // OAuth回调成功
	OAuthUnbindSuccess         = 200 // 解绑OAuth成功
	EmailVerifySuccess         = 200 // 验证邮箱成功
	MobileBindSuccess          = 200 // 绑定手机成功
	EmailBindSuccess           = 200 // 绑定邮箱成功

	// 设备管理相关 (2006268-2006270)
	UserDeviceListQuerySuccess       = 200 // 查询设备列表成功
	UserDeviceUnbindSuccess          = 200 // 解绑设备成功
	UserDeviceStatisticsQuerySuccess = 200 // 获取设备在线统计成功

	// Public Order 相关 (2006102-2006109)
	OrderCloseSuccess       = 200 // 关闭订单成功
	OrderDetailQuerySuccess = 200 // 查询订单详情成功
	OrderListQuerySuccess   = 200 // 查询订单列表成功
	OrderPreCreateSuccess   = 200 // 预创建订单成功
	PurchaseSuccess         = 200 // 购买成功
	RechargeSuccess         = 200 // 充值成功
	RenewalSuccess          = 200 // 续费成功
	TrafficResetSuccess     = 200 // 重置流量成功

	// Auth OAuth 相关 (2006110-2006112)
	OAuthLoginSuccess    = 200 // OAuth登录成功
	OAuthTokenGetSuccess = 200 // 获取OAuth令牌成功
	AppleCallbackSuccess = 200 // Apple回调成功

	// Public Ticket 相关 (2006113-2006117)
	UserTicketCreateSuccess       = 200 // 创建工单成功
	UserTicketListQuerySuccess    = 200 // 查询工单列表成功
	UserTicketDetailQuerySuccess  = 200 // 查询工单详情成功
	UserTicketStatusUpdateSuccess = 200 // 更新工单状态成功
	UserTicketFollowCreateSuccess = 200 // 创建工单跟进成功

	// Public Common 相关 (2006118-2006126)
	GetAdsSuccess                = 200 // 获取广告列表成功
	GetClientSuccess             = 200 // 获取客户端列表成功
	GetPrivacyPolicySuccess      = 200 // 获取隐私政策成功
	GetTosSuccess                = 200 // 获取服务条款成功
	GetGlobalConfigSuccess       = 200 // 获取全局配置成功
	GetStatSuccess               = 200 // 获取统计数据成功
	SendEmailCodeSuccess         = 200 // 发送邮箱验证码成功
	SendSmsCodeSuccess           = 200 // 发送短信验证码成功
	CheckVerificationCodeSuccess = 200 // 验证码校验成功

	// Admin Log 相关 (2006127-2006141)
	FilterBalanceLogSuccess              = 200 // 查询余额日志成功
	FilterCommissionLogSuccess           = 200 // 查询佣金日志成功
	FilterEmailLogSuccess                = 200 // 查询邮件日志成功
	FilterGiftLogSuccess                 = 200 // 查询礼品日志成功
	FilterLoginLogSuccess                = 200 // 查询登录日志成功
	GetMessageLogListSuccess             = 200 // 查询消息日志列表成功
	FilterMobileLogSuccess               = 200 // 查询短信日志成功
	FilterRegisterLogSuccess             = 200 // 查询注册日志成功
	FilterServerTrafficLogSuccess        = 200 // 查询服务器流量日志成功
	FilterSubscribeLogSuccess            = 200 // 查询订阅日志成功
	FilterResetSubscribeLogSuccess       = 200 // 查询重置订阅日志成功
	FilterUserSubscribeTrafficLogSuccess = 200 // 查询用户订阅流量日志成功
	FilterTrafficLogDetailsSuccess       = 200 // 查询流量日志详情成功
	GetLogSettingSuccess                 = 200 // 获取日志设置成功
	UpdateLogSettingSuccess              = 200 // 更新日志设置成功

	// Admin Ticket 相关 (2006142-2006145)
	AdminUpdateTicketStatusSuccess = 200 // 更新工单状态成功
	AdminGetTicketSuccess          = 200 // 获取工单详情成功
	AdminCreateTicketFollowSuccess = 200 // 创建工单跟进成功
	AdminGetTicketListSuccess      = 200 // 获取工单列表成功

	// Admin Order 相关 (2006146-2006148)
	AdminCreateOrderSuccess       = 200 // 创建订单成功
	AdminGetOrderListSuccess      = 200 // 获取订单列表成功
	AdminUpdateOrderStatusSuccess = 200 // 更新订单状态成功

	// Admin Console 相关 (2006149-2006152)
	QueryRevenueStatisticsSuccess = 200 // 查询营收统计成功
	QueryUserStatisticsSuccess    = 200 // 查询用户统计成功
	QueryTicketWaitReplySuccess   = 200 // 查询待回复工单成功
	QueryServerTotalDataSuccess   = 200 // 查询服务器总数据成功

	// Public Portal 相关 (2006153-2006158)
	GetSubscriptionSuccess            = 200 // 获取订阅列表成功
	PrePurchaseOrderSuccess           = 200 // 预购买订单成功
	PortalPurchaseSuccess             = 200 // 门户购买成功
	GetAvailablePaymentMethodsSuccess = 200 // 获取可用支付方式成功
	PurchaseCheckoutSuccess           = 200 // 购买结账成功
	QueryPurchaseOrderSuccess         = 200 // 查询购买订单成功

	// Public Subscribe 相关 (2006159)
	SubscribeQuerySuccess = 200 // 查询订阅列表成功

	// Public Announcement 相关 (2006160)
	AnnouncementQuerySuccess = 200 // 查询公告列表成功

	// Public Document 相关 (2006161)
	DocumentQuerySuccess = 200 // 查询文档成功

	// Admin Ads 相关 (2006162-2006166)
	AdminGetAdsListSuccess = 200 // 获取广告列表成功
	AdminGetAdsSuccess     = 200 // 获取广告详情成功
	AdminCreateAdsSuccess  = 200 // 创建广告成功
	AdminUpdateAdsSuccess  = 200 // 更新广告成功
	AdminDeleteAdsSuccess  = 200 // 删除广告成功

	// Admin Announcement 相关 (2006167-2006171)
	AdminCreateAnnouncementSuccess = 200 // 创建公告成功
	AdminUpdateAnnouncementSuccess = 200 // 更新公告成功
	AdminGetAnnouncementSuccess    = 200 // 获取公告详情成功
	AdminListAnnouncementsSuccess  = 200 // 获取公告列表成功
	AdminDeleteAnnouncementSuccess = 200 // 删除公告成功

	// Admin Application 相关 (2006172-2006176)
	AdminCreateSubscribeApplicationSuccess  = 200 // 创建订阅应用配置成功
	AdminPreviewSubscribeTemplateSuccess    = 200 // 预览订阅模板成功
	AdminUpdateSubscribeApplicationSuccess  = 200 // 更新订阅应用配置成功
	AdminDeleteSubscribeApplicationSuccess  = 200 // 删除订阅应用配置成功
	AdminGetSubscribeApplicationListSuccess = 200 // 获取订阅应用配置列表成功

	// Admin Coupon 相关 (2006177-2006181)
	AdminCreateCouponSuccess      = 200 // 创建优惠券成功
	AdminUpdateCouponSuccess      = 200 // 更新优惠券成功
	AdminDeleteCouponSuccess      = 200 // 删除优惠券成功
	AdminBatchDeleteCouponSuccess = 200 // 批量删除优惠券成功
	AdminGetCouponListSuccess     = 200 // 获取优惠券列表成功

	// Admin RechargePromotion 相关 (2006208-2006212)  // 充值促销
	AdminCreateRechargePromotionSuccess       = 200 // 创建充值促销活动成功
	AdminUpdateRechargePromotionSuccess       = 200 // 更新充值促销活动成功
	AdminDeleteRechargePromotionSuccess       = 200 // 删除充值促销活动成功
	AdminGetRechargePromotionListSuccess      = 200 // 获取充值促销活动列表成功
	AdminUpdateRechargePromotionStatusSuccess = 200 // 更新充值促销活动状态成功

	// Admin Payment 相关 (2006182-2006186)
	AdminCreatePaymentMethodSuccess  = 200 // 创建支付方式成功
	AdminUpdatePaymentMethodSuccess  = 200 // 更新支付方式成功
	AdminDeletePaymentMethodSuccess  = 200 // 删除支付方式成功
	AdminGetPaymentMethodListSuccess = 200 // 获取支付方式列表成功
	AdminGetPaymentPlatformSuccess   = 200 // 获取支付平台成功

	// Admin Document 相关 (2006187-2006192)
	AdminCreateDocumentSuccess      = 200 // 创建文档成功
	AdminUpdateDocumentSuccess      = 200 // 更新文档成功
	AdminDeleteDocumentSuccess      = 200 // 删除文档成功
	AdminBatchDeleteDocumentSuccess = 200 // 批量删除文档成功
	AdminGetDocumentListSuccess     = 200 // 获取文档列表成功
	AdminGetDocumentDetailSuccess   = 200 // 获取文档详情成功

	// Admin AuthMethod 相关 (2006193-2006199)
	AdminGetAuthMethodConfigSuccess    = 200 // 获取认证方法配置成功
	AdminUpdateAuthMethodConfigSuccess = 200 // 更新认证方法配置成功
	AdminGetEmailPlatformSuccess       = 200 // 获取邮件平台列表成功
	AdminGetSmsPlatformSuccess         = 200 // 获取短信平台列表成功
	AdminGetAuthMethodListSuccess      = 200 // 获取认证方法列表成功
	AdminTestEmailSendSuccess          = 200 // 测试邮件发送成功
	AdminTestSmsSendSuccess            = 200 // 测试短信发送成功

	// Admin Marketing 相关 (2006200-2006207)
	AdminCreateBatchSendEmailTaskSuccess    = 200 // 创建批量发送邮件任务成功
	AdminGetBatchSendEmailTaskListSuccess   = 200 // 获取批量发送邮件任务列表成功
	AdminStopBatchSendEmailTaskSuccess      = 200 // 停止批量发送邮件任务成功
	AdminGetPreSendEmailCountSuccess        = 200 // 获取预发送邮件数量成功
	AdminGetBatchSendEmailTaskStatusSuccess = 200 // 获取批量发送邮件任务状态成功
	AdminCreateQuotaTaskSuccess             = 200 // 创建配额任务成功
	AdminQueryQuotaTaskPreCountSuccess      = 200 // 查询配额任务预计数量成功
	AdminQueryQuotaTaskListSuccess          = 200 // 查询配额任务列表成功

	// Admin Subscribe 相关 (2006208-2006219)
	AdminCreateSubscribeSuccess           = 200 // 创建订阅套餐成功
	AdminUpdateSubscribeSuccess           = 200 // 更新订阅套餐成功
	AdminDeleteSubscribeSuccess           = 200 // 删除订阅套餐成功
	AdminBatchDeleteSubscribeSuccess      = 200 // 批量删除订阅套餐成功
	AdminGetSubscribeDetailsSuccess       = 200 // 获取订阅套餐详情成功
	AdminGetSubscribeListSuccess          = 200 // 获取订阅套餐列表成功
	AdminSubscribeSortSuccess             = 200 // 订阅套餐排序成功
	AdminCreateSubscribeGroupSuccess      = 200 // 创建订阅组成功
	AdminUpdateSubscribeGroupSuccess      = 200 // 更新订阅组成功
	AdminDeleteSubscribeGroupSuccess      = 200 // 删除订阅组成功
	AdminBatchDeleteSubscribeGroupSuccess = 200 // 批量删除订阅组成功
	AdminGetSubscribeGroupListSuccess     = 200 // 获取订阅组列表成功

	// Admin Server 相关 (2006220-2006234)
	AdminCreateServerSuccess         = 200 // 创建服务器成功
	AdminUpdateServerSuccess         = 200 // 更新服务器成功
	AdminDeleteServerSuccess         = 200 // 删除服务器成功
	AdminFilterServerListSuccess     = 200 // 获取服务器列表成功
	AdminGetServerProtocolsSuccess   = 200 // 获取服务器协议成功
	AdminCreateNodeSuccess           = 200 // 创建节点成功
	AdminUpdateNodeSuccess           = 200 // 更新节点成功
	AdminDeleteNodeSuccess           = 200 // 删除节点成功
	AdminFilterNodeListSuccess       = 200 // 获取节点列表成功
	AdminToggleNodeStatusSuccess     = 200 // 切换节点状态成功
	AdminQueryNodeTagSuccess         = 200 // 查询节点标签成功
	AdminHasMigrateServerNodeSuccess = 200 // 检查服务器节点迁移成功
	AdminMigrateServerNodeSuccess    = 200 // 迁移服务器节点成功
	AdminResetSortWithServerSuccess  = 200 // 重置服务器排序成功
	AdminResetSortWithNodeSuccess    = 200 // 重置节点排序成功

	// Admin User 相关 (2006235-2006243)
	AdminCreateUserSuccess               = 200 // 创建用户成功
	AdminDeleteUserSuccess               = 200 // 删除用户成功
	AdminBatchDeleteUserSuccess          = 200 // 批量删除用户成功
	AdminCurrentUserSuccess              = 200 // 获取当前用户成功
	AdminGetUserDetailSuccess            = 200 // 获取用户详情成功
	AdminGetUserListSuccess              = 200 // 获取用户列表成功
	AdminUpdateUserBasicInfoSuccess      = 200 // 更新用户基本信息成功
	AdminUpdateUserNotifySettingsSuccess = 200 // 更新用户通知设置成功
	AdminGetUserLoginLogsSuccess         = 200 // 获取用户登录日志成功

	// Admin System 相关 (2006244-2006267)
	AdminGetCurrencyConfigSuccess         = 200 // 获取货币配置成功
	AdminUpdateCurrencyConfigSuccess      = 200 // 更新货币配置成功
	AdminGetInviteConfigSuccess           = 200 // 获取邀请配置成功
	AdminUpdateInviteConfigSuccess        = 200 // 更新邀请配置成功
	AdminGetNodeConfigSuccess             = 200 // 获取节点配置成功
	AdminUpdateNodeConfigSuccess          = 200 // 更新节点配置成功
	AdminGetPrivacyPolicyConfigSuccess    = 200 // 获取隐私政策配置成功
	AdminUpdatePrivacyPolicyConfigSuccess = 200 // 更新隐私政策配置成功
	AdminGetRegisterConfigSuccess         = 200 // 获取注册配置成功
	AdminUpdateRegisterConfigSuccess      = 200 // 更新注册配置成功
	AdminGetSiteConfigSuccess             = 200 // 获取站点配置成功
	AdminUpdateSiteConfigSuccess          = 200 // 更新站点配置成功
	AdminGetSubscribeConfigSuccess        = 200 // 获取订阅配置成功
	AdminUpdateSubscribeConfigSuccess     = 200 // 更新订阅配置成功
	AdminGetTosConfigSuccess              = 200 // 获取服务条款配置成功
	AdminUpdateTosConfigSuccess           = 200 // 更新服务条款配置成功
	AdminGetVerifyCodeConfigSuccess       = 200 // 获取验证码配置成功
	AdminUpdateVerifyCodeConfigSuccess    = 200 // 更新验证码配置成功
	AdminGetVerifyConfigSuccess           = 200 // 获取验证配置成功
	AdminUpdateVerifyConfigSuccess        = 200 // 更新验证配置成功
	AdminGetNodeMultiplierSuccess         = 200 // 获取节点倍率成功
	AdminPreViewNodeMultiplierSuccess     = 200 // 预览节点倍率成功
	AdminSetNodeMultiplierSuccess         = 200 // 设置节点倍率成功
	AdminSettingTelegramBotSuccess        = 200 // 设置Telegram机器人成功

	// Admin Redemption 相关 (2006280-2006286)
	AdminCreateRedemptionCodeSuccess       = 200 // 创建兑换码成功
	AdminUpdateRedemptionCodeSuccess       = 200 // 更新兑换码成功
	AdminToggleRedemptionCodeStatusSuccess = 200 // 切换兑换码状态成功
	AdminDeleteRedemptionCodeSuccess       = 200 // 删除兑换码成功
	AdminBatchDeleteRedemptionCodeSuccess  = 200 // 批量删除兑换码成功
	AdminGetRedemptionCodeListSuccess      = 200 // 获取兑换码列表成功
	AdminGetRedemptionRecordListSuccess    = 200 // 获取兑换记录列表成功

	// Admin Tool 相关 (2006287-2006290)
	AdminGetSystemLogSuccess    = 200 // 获取系统日志成功
	AdminRestartSystemSuccess   = 200 // 重启系统成功
	AdminGetVersionSuccess      = 200 // 获取版本信息成功
	AdminQueryIPLocationSuccess = 200 // 查询IP地理位置成功

	// Admin Group 相关 (2006291-2006309)
	AdminGetUserGroupListSuccess         = 200 // 获取用户组列表成功
	AdminCreateUserGroupSuccess          = 200 // 创建用户组成功
	AdminUpdateUserGroupSuccess          = 200 // 更新用户组成功
	AdminDeleteUserGroupSuccess          = 200 // 删除用户组成功
	AdminUpdateUserUserGroupSuccess      = 200 // 更新用户的用户组成功
	AdminGetNodeGroupListSuccess         = 200 // 获取节点组列表成功
	AdminCreateNodeGroupSuccess          = 200 // 创建节点组成功
	AdminUpdateNodeGroupSuccess          = 200 // 更新节点组成功
	AdminDeleteNodeGroupSuccess          = 200 // 删除节点组成功
	AdminGetGroupConfigSuccess           = 200 // 获取分组配置成功
	AdminUpdateGroupConfigSuccess        = 200 // 更新分组配置成功
	AdminRecalculateGroupSuccess         = 200 // 重新计算分组成功
	AdminGetRecalculationStatusSuccess   = 200 // 获取重新计算状态成功
	AdminGetGroupHistorySuccess          = 200 // 获取分组历史成功
	AdminGetGroupHistoryDetailSuccess    = 200 // 获取分组历史详情成功
	AdminExportGroupResultSuccess        = 200 // 导出分组结果成功
	AdminMigrateUsersSuccess             = 200 // 迁移用户成功
	AdminPreviewUserNodesSuccess         = 200 // 预览用户节点成功
	AdminResetGroupsSuccess              = 200 // 重置所有分组成功
	AdminGetSubscribeGroupMappingSuccess = 200 // 获取订阅组映射成功

	// ==== 业务错误码（与旧项目一致） ====

	// 参数验证错误
	ErrInvalidUserID        = 400 // 无效的用户ID
	ErrInvalidOrderID       = 400 // 无效的订单ID
	ErrInvalidSubscribeID   = 400 // 无效的订阅ID
	ErrInvalidPaymentID     = 400 // 无效的支付ID
	ErrInvalidServerID      = 400 // 无效的服务器ID
	ErrInvalidNodeID        = 400 // 无效的节点ID
	ErrInvalidCouponCode    = 400 // 无效的优惠券码
	ErrMissingRequiredParam = 400 // 缺少必需参数
	ErrInvalidParamFormat   = 400 // 参数格式错误

	// 数据不存在错误
	ErrUserNotFound                 = 20002 // 用户不存在
	ErrOrderNotFound                = 61001 // 订单不存在
	ErrSubscribeNotFound            = 60002 // 订阅不存在
	ErrPaymentNotFound              = 61002 // 支付方式不存在
	ErrServerNotFound               = 30002 // 服务器不存在
	ErrNodeNotFound                 = 30002 // 节点不存在
	ErrCouponNotFound               = 50001 // 优惠券不存在
	ErrDeviceNotFound               = 90017 // 设备不存在
	ErrAuthMethodNotFound           = 400   // 认证方法不存在
	ErrAnnouncementNotFound         = 400   // 公告不存在
	ErrDocumentNotFound             = 400   // 文档不存在
	ErrAdsNotFound                  = 400   // 广告不存在
	ErrSystemNotFound               = 400   // 系统配置不存在
	ErrSubscribeApplicationNotFound = 400   // 订阅申请不存在
	ErrServerGroupNotFound          = 30004 // 服务器组不存在
	ErrTicketNotFound               = 400   // 工单不存在
	ErrSystemLogNotFound            = 400   // 系统日志不存在
	ErrTaskNotFound                 = 400   // 任务不存在
	ErrRedemptionCodeNotFound       = 400   // 兑换码不存在
	ErrUserGroupNotFound            = 400   // 用户组不存在

	// 业务逻辑错误
	ErrInvalidTaskType     = 400 // 无效的任务类型
	ErrInvalidTaskStatus   = 400 // 无效的任务状态
	ErrTaskCannotBeStopped = 400 // 任务无法停止

	// 数据冲突错误
	ErrUserAlreadyExists                 = 20001 // 用户已存在
	ErrOrderAlreadyExists                = 400   // 订单已存在
	ErrSubscribeAlreadyExists            = 60003 // 订阅已存在
	ErrPaymentAlreadyExists              = 400   // 支付方式已存在
	ErrServerAlreadyExists               = 30001 // 服务器已存在
	ErrNodeAlreadyExists                 = 30001 // 节点已存在
	ErrCouponAlreadyExists               = 400   // 优惠券已存在
	ErrDuplicateEmail                    = 90011 // 邮箱已存在
	ErrDuplicateUsername                 = 20001 // 用户名已存在
	ErrAnnouncementAlreadyExists         = 400   // 公告已存在
	ErrDocumentAlreadyExists             = 400   // 文档已存在
	ErrSystemAlreadyExists               = 400   // 系统配置已存在
	ErrAuthMethodAlreadyExists           = 400   // 认证方法已存在
	ErrSubscribeApplicationAlreadyExists = 400   // 订阅申请已存在

	// 业务逻辑错误
	ErrOrderCannotCancel                  = 61003 // 订单不能取消
	ErrOrderCannotComplete                = 61003 // 订单不能完成
	ErrOrderCannotClose                   = 61003 // 订单不能关闭
	ErrCouponExpired                      = 50005 // 优惠券已过期
	ErrCouponNotAvailable                 = 50003 // 优惠券不可用
	ErrCouponUsedUp                       = 50002 // 优惠券已用完
	ErrCouponUserLimitExceeded            = 50004 // 优惠券用户使用次数超限
	ErrInsufficientBalance                = 20005 // 余额不足
	ErrUserCommissionNotEnough            = 20010 // 佣金不足
	ErrOrderPaymentFailed                 = 61003 // 订单支付失败
	ErrDeviceLimitExceeded                = 400   // 设备数量超限
	ErrSubscribeExpired                   = 60001 // 订阅已过期
	ErrTrafficExceeded                    = 61005 // 流量超限
	ErrSubscribeInUse                     = 60004 // 订阅套餐正在使用中
	ErrInvalidOrderStatus                 = 61003 // 无效的订单状态
	ErrInvalidParameter                   = 400   // 无效的参数
	ErrTitleRequired                      = 400   // 标题不能为空
	ErrTypeRequired                       = 400   // 类型不能为空
	ErrInvalidTimeRange                   = 400   // 时间范围无效
	ErrInvalidTicketStatus                = 400   // 无效的工单状态
	ErrInvalidTicketPriority              = 400   // 无效的工单优先级
	ErrUnsupportedPlatform                = 400   // 不支持的支付平台
	ErrStopRegister                       = 20006 // 停止注册
	ErrTelegramNotBound                   = 20007 // Telegram 未绑定
	ErrUserNotBindOauth                   = 20008 // 用户未绑定 OAuth 方法
	ErrInviteCodeError                    = 20009 // 邀请码错误
	ErrRegisterIPLimit                    = 20011 // 注册 IP 限制
	ErrNodeGroupExist                     = 30003 // 节点组已存在
	ErrNodeGroupNotEmpty                  = 30005 // 节点组不为空
	ErrTooManyRequests                    = 401   // 请求过于频繁
	ErrInvalidCiphertext                  = 40006 // 无效密文
	ErrSecretIsEmpty                      = 40007 // Secret 为空
	ErrSubscribeNotAvailable              = 60002 // 订阅不可用
	ErrSingleSubscribeModeExceedsLimit    = 60005 // 单个订阅模式超过限制
	ErrSubscribeQuotaLimit                = 60006 // 订阅配额限制
	ErrSubscribeOutOfStock                = 60007 // 订阅已售罄
	ErrInsufficientOfPeriod               = 61004 // 周期数不足
	ErrExistAvailableTraffic              = 61005 // 存在可用流量
	ErrVerifyCodeError                    = 70001 // 验证码错误
	ErrQueueEnqueueError                  = 80001 // 队列入队错误
	ErrDebugModeError                     = 90001 // 调试模式已启用
	ErrSmsNotEnabled                      = 90003 // 电话登录未启用
	ErrEmailNotEnabled                    = 90004 // 邮件功能未启用或 SMTP 未配置
	ErrGetAuthenticatorError              = 90005 // 不支持的登录方法
	ErrAuthenticatorNotSupportedError     = 90006 // 认证器不支持此方法
	ErrTelephoneAreaCodeIsEmpty           = 90007 // 电话区号为空
	ErrPasswordIsEmpty                    = 90008 // 密码为空
	ErrAreaCodeIsEmpty                    = 90009 // 区号为空
	ErrPasswordOrVerificationCodeRequired = 90010 // 需要密码或验证码
	ErrDeviceExist                        = 90013 // 设备已存在
	ErrTelephoneError                     = 90014 // 电话号码错误
	ErrTodaySendCountExceedsLimit         = 90015 // 今日发送次数超限
	ErrInvalidEmail                       = 90016 // 邮箱格式不合法
	ErrUseridNotMatch                     = 90018 // 用户 ID 不匹配

	// ==== 权限/请求错误码（与旧项目一致） ====

	// 认证错误
	ErrMissingAuthToken     = 40002 // 缺少认证令牌
	ErrInvalidAuthToken     = 40003 // 无效的认证令牌
	ErrAuthTokenExpired     = 40004 // 认证令牌已过期
	ErrInvalidCredentials   = 20003 // 无效的凭证
	ErrUserNotAuthenticated = 40002 // 用户未认证
	ErrPasswordIncorrect    = 20003 // 密码错误
	ErrAccountLocked        = 20004 // 账户已锁定
	ErrAccountDisabled      = 20004 // 账户已禁用

	// 授权错误
	ErrPermissionDenied       = 40008 // 权限被拒绝
	ErrInsufficientPermission = 40008 // 权限不足
	ErrResourceAccessDenied   = 40008 // 资源访问被拒绝
	ErrOperationNotAllowed    = 40008 // 操作不被允许
	ErrNotResourceOwner       = 40008 // 非资源所有者
	ErrInvalidAccess          = 40005 // 无效访问

	// ==== 系统错误码（与旧项目一致） ====

	// 数据库错误
	ErrDatabaseConnection  = 500   // 数据库连接失败
	ErrDatabaseQuery       = 10001 // 数据库查询失败
	ErrDatabaseUpdate      = 10002 // 数据库更新失败
	ErrDatabaseInsert      = 10003 // 数据库插入失败
	ErrDatabaseDelete      = 10004 // 数据库删除失败
	ErrDatabaseTransaction = 500   // 数据库事务失败
	ErrDatabaseConstraint  = 500   // 数据库约束错误
	ErrDatabaseTimeout     = 500   // 数据库超时

	// 缓存错误
	ErrCacheConnection  = 500 // 缓存连接失败
	ErrCacheGet         = 500 // 缓存获取失败
	ErrCacheSet         = 500 // 缓存设置失败
	ErrCacheDelete      = 500 // 缓存删除失败
	ErrCacheExpired     = 500 // 缓存已过期
	ErrCacheSerialize   = 500 // 缓存序列化失败
	ErrCacheDeserialize = 500 // 缓存反序列化失败

	// 内部服务错误
	ErrInternalError        = 500 // 内部错误
	ErrServiceUnavailable   = 500 // 服务不可用
	ErrServiceTimeout       = 500 // 服务超时
	ErrConfigurationError   = 500 // 配置错误
	ErrInitializationFailed = 500 // 初始化失败
	ErrResourceExhausted    = 500 // 资源耗尽

	// 外部服务错误
	ErrIPGeolocationFailed = 500   // IP地理位置查询失败
	ErrPaymentGatewayError = 500   // 支付网关错误
	ErrEmailSendFailed     = 500   // 邮件发送失败
	ErrSMSSendFailed       = 90002 // 短信发送失败
	ErrThirdPartyAPIError  = 500   // 第三方API错误
)

// CodeMessages 响应码消息映射表
var CodeMessages = map[int]string{
	200: "Success",
}

// getCodeMessage 获取响应码对应的消息
func getCodeMessage(code int) string {
	if code == 200 {
		return "Success"
	}
	if message, exists := CodeMessages[code]; exists {
		return message
	}
	switch code {
	case 500:
		return "Internal Server Error"
	case 10001:
		return "Database query error"
	case 10002:
		return "Database update error"
	case 10003:
		return "Database insert error"
	case 10004:
		return "Database deleted error"
	case 20001:
		return "User already exists"
	case 20002:
		return "User does not exist"
	case 20003:
		return "User password error"
	case 20004:
		return "User disabled"
	case 20005:
		return "Insufficient balance"
	case 20006:
		return "Stop register"
	case 20007:
		return "Telegram not bound"
	case 20008:
		return "User not bind oauth method"
	case 20009:
		return "Invite code error"
	case 20010:
		return "User commission not enough"
	case 20011:
		return "Too many registrations"
	case 30001:
		return "Node already exists"
	case 30002:
		return "Node does not exist"
	case 30003:
		return "Node group already exists"
	case 30004:
		return "Node group does not exist"
	case 30005:
		return "Node group is not empty"
	case 400:
		return "Param Error"
	case 401:
		return "Too Many Requests"
	case 40002:
		return "User token is empty"
	case 40003:
		return "User token is invalid"
	case 40004:
		return "User token is expired"
	case 40005:
		return "Invalid access"
	case 40006:
		return "Invalid ciphertext"
	case 40007:
		return "Secret is empty"
	case 40008:
		return "Permission denied"
	case 50001:
		return "Coupon does not exist"
	case 50002:
		return "Coupon has already been used"
	case 50003:
		return "Coupon does not match the order or conditions"
	case 50004:
		return "Coupon has insufficient remaining uses"
	case 50005:
		return "Coupon is expired"
	case 60001:
		return "Subscribe is expired"
	case 60002:
		return "Subscribe is not available"
	case 60003:
		return "User has subscription"
	case 60004:
		return "Subscribe is used"
	case 60005:
		return "Single subscribe mode exceeds limit"
	case 60006:
		return "Subscribe quota limit"
	case 60007:
		return "Subscribe out of stock"
	case 61001:
		return "Order does not exist"
	case 61002:
		return "Payment method not found"
	case 61003:
		return "Order status error"
	case 61004:
		return "Insufficient number of period"
	case 61005:
		return "Exist available traffic"
	case 70001:
		return "Verify code error"
	case 80001:
		return "Queue enqueue error"
	case 90001:
		return "Debug mode is enabled"
	case 90002:
		return "Send sms error"
	case 90003:
		return "Telephone login is not enabled"
	case 90004:
		return "Email is not enabled or SMTP is not configured"
	case 90005:
		return "Unsupported login method"
	case 90006:
		return "The authenticator does not support this method"
	case 90007:
		return "Telephone area code is empty"
	case 90008:
		return "password is empty"
	case 90009:
		return "Area code is empty"
	case 90010:
		return "Password or verification code required"
	case 90011:
		return "Email already exists"
	case 90012:
		return "Telephone already exists"
	case 90013:
		return "device exists"
	case 90014:
		return "telephone number error"
	case 90015:
		return "This account has reached the limit of sending times today"
	case 90016:
		return "Invalid email address"
	case 90017:
		return "Device does not exist"
	case 90018:
		return "Userid not match"
	default:
		return "Internal Server Error"
	}
}
