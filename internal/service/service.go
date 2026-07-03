package service

import (
	"github.com/google/wire"

	ads "github.com/npanel-dev/NPanel-backend/internal/service/admin/ads"
	announcement "github.com/npanel-dev/NPanel-backend/internal/service/admin/announcement"
	application "github.com/npanel-dev/NPanel-backend/internal/service/admin/application"
	authmethod "github.com/npanel-dev/NPanel-backend/internal/service/admin/authmethod"
	adminconsole "github.com/npanel-dev/NPanel-backend/internal/service/admin/console"
	admincoupon "github.com/npanel-dev/NPanel-backend/internal/service/admin/coupon"
	admindocument "github.com/npanel-dev/NPanel-backend/internal/service/admin/document"
	maingroup "github.com/npanel-dev/NPanel-backend/internal/service/admin/group"
	adminlog "github.com/npanel-dev/NPanel-backend/internal/service/admin/log"
	adminmarketing "github.com/npanel-dev/NPanel-backend/internal/service/admin/marketing"
	adminorder "github.com/npanel-dev/NPanel-backend/internal/service/admin/order"
	adminpayment "github.com/npanel-dev/NPanel-backend/internal/service/admin/payment"
	adminredemption "github.com/npanel-dev/NPanel-backend/internal/service/admin/redemption"
	adminrouting "github.com/npanel-dev/NPanel-backend/internal/service/admin/routing"
	adminserver "github.com/npanel-dev/NPanel-backend/internal/service/admin/server"
	adminsubscribe "github.com/npanel-dev/NPanel-backend/internal/service/admin/subscribe"
	adminsystem "github.com/npanel-dev/NPanel-backend/internal/service/admin/system"
	adminticket "github.com/npanel-dev/NPanel-backend/internal/service/admin/ticket"
	admintool "github.com/npanel-dev/NPanel-backend/internal/service/admin/tool"
	adminuser "github.com/npanel-dev/NPanel-backend/internal/service/admin/user"
	adminwithdrawal "github.com/npanel-dev/NPanel-backend/internal/service/admin/withdrawal"
	"github.com/npanel-dev/NPanel-backend/internal/service/auth"
	authoauth "github.com/npanel-dev/NPanel-backend/internal/service/auth/oauth"
	"github.com/npanel-dev/NPanel-backend/internal/service/common"
	publicorder "github.com/npanel-dev/NPanel-backend/internal/service/public"
	publicannouncement "github.com/npanel-dev/NPanel-backend/internal/service/public/announcement"
	publicdocument "github.com/npanel-dev/NPanel-backend/internal/service/public/document"
	publicpayment "github.com/npanel-dev/NPanel-backend/internal/service/public/payment"
	publicportal "github.com/npanel-dev/NPanel-backend/internal/service/public/portal"
	publicredemption "github.com/npanel-dev/NPanel-backend/internal/service/public/redemption"
	publicsubscribe "github.com/npanel-dev/NPanel-backend/internal/service/public/subscribe"
	publicsubscription "github.com/npanel-dev/NPanel-backend/internal/service/public/subscription"
	publicticket "github.com/npanel-dev/NPanel-backend/internal/service/public/ticket"
	publicuser "github.com/npanel-dev/NPanel-backend/internal/service/public/user"
	// Server模块服务
	"github.com/npanel-dev/NPanel-backend/internal/service/server"
)

// ProviderSet is service providers
var ProviderSet = wire.NewSet(
	ads.NewAdsService,
	announcement.NewAnnouncementService,
	application.NewSubscribeApplicationService,
	authmethod.NewAuthMethodService,
	adminconsole.NewConsoleService,
	admincoupon.NewCouponService,
	admindocument.NewDocumentService,
	adminlog.NewLogService,
	adminmarketing.NewMarketingService,
	adminorder.NewOrderService,
	adminpayment.NewPaymentService,
	adminrouting.NewRoutingService,
	adminserver.NewServerService,
	adminsubscribe.NewSubscribeService,
	adminsystem.NewSystemService,
	adminticket.NewTicketService,
	adminredemption.NewRedemptionService,
	admintool.NewToolService,
	adminwithdrawal.NewWithdrawalService,
	maingroup.NewGroupService,
	// Admin User模块服务
	adminuser.NewUserService,
	adminuser.NewUserAuthMethodService,
	adminuser.NewUserDeviceService,
	adminuser.NewUserSubscribeService,
	// Auth模块服务
	auth.NewAuthService,
	// Auth OAuth模块服务
	authoauth.NewOAuthService,
	// Common模块服务
	common.NewCommonService,
	// Public Order模块服务
	publicorder.NewPublicOrderService,
	// Public Announcement模块服务
	publicannouncement.NewAnnouncementService,
	// Public Document模块服务
	publicdocument.NewDocumentService,
	// Public Payment模块服务
	publicpayment.NewPaymentService,
	// Public Portal模块服务
	publicportal.NewPortalService,
	// Public Subscribe模块服务
	publicsubscribe.NewSubscribeService,
	// Public Subscription模块服务（订阅配置生成）
	publicsubscription.NewPublicSubscriptionService,
	// Public Redemption模块服务
	publicredemption.NewRedemptionService,
	// Public Ticket模块服务
	publicticket.NewTicketService,
	// Public User模块服务
	publicuser.NewUserService,
	// Server模块服务
	server.NewServerService,
)
