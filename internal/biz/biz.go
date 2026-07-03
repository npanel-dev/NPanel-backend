package biz

import (
	"github.com/google/wire"

	ads "github.com/npanel-dev/NPanel-backend/internal/biz/admin/ads"
	announcement "github.com/npanel-dev/NPanel-backend/internal/biz/admin/announcement"
	application "github.com/npanel-dev/NPanel-backend/internal/biz/admin/application"
	authmethod "github.com/npanel-dev/NPanel-backend/internal/biz/admin/authmethod"
	adminconsole "github.com/npanel-dev/NPanel-backend/internal/biz/admin/console"
	admincoupon "github.com/npanel-dev/NPanel-backend/internal/biz/admin/coupon"
	admindocument "github.com/npanel-dev/NPanel-backend/internal/biz/admin/document"
	maingroup "github.com/npanel-dev/NPanel-backend/internal/biz/admin/group"
	adminlog "github.com/npanel-dev/NPanel-backend/internal/biz/admin/log"
	adminmarketing "github.com/npanel-dev/NPanel-backend/internal/biz/admin/marketing"
	adminorder "github.com/npanel-dev/NPanel-backend/internal/biz/admin/order"
	adminpayment "github.com/npanel-dev/NPanel-backend/internal/biz/admin/payment"
	adminredemption "github.com/npanel-dev/NPanel-backend/internal/biz/admin/redemption"
	adminrouting "github.com/npanel-dev/NPanel-backend/internal/biz/admin/routing"
	adminserver "github.com/npanel-dev/NPanel-backend/internal/biz/admin/server"
	adminsubscribe "github.com/npanel-dev/NPanel-backend/internal/biz/admin/subscribe"
	adminsystem "github.com/npanel-dev/NPanel-backend/internal/biz/admin/system"
	adminticket "github.com/npanel-dev/NPanel-backend/internal/biz/admin/ticket"
	admintool "github.com/npanel-dev/NPanel-backend/internal/biz/admin/tool"
	adminuser "github.com/npanel-dev/NPanel-backend/internal/biz/admin/user"
	adminwithdrawal "github.com/npanel-dev/NPanel-backend/internal/biz/admin/withdrawal"
	"github.com/npanel-dev/NPanel-backend/internal/biz/auth"
	authoauth "github.com/npanel-dev/NPanel-backend/internal/biz/auth/oauth"
	"github.com/npanel-dev/NPanel-backend/internal/biz/common"
	publicorder "github.com/npanel-dev/NPanel-backend/internal/biz/public"
	publicannouncement "github.com/npanel-dev/NPanel-backend/internal/biz/public/announcement"
	publicdocument "github.com/npanel-dev/NPanel-backend/internal/biz/public/document"
	publicpayment "github.com/npanel-dev/NPanel-backend/internal/biz/public/payment"
	publicportal "github.com/npanel-dev/NPanel-backend/internal/biz/public/portal"
	publicredemption "github.com/npanel-dev/NPanel-backend/internal/biz/public/redemption"
	publicsubscribe "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscribe"
	subscription "github.com/npanel-dev/NPanel-backend/internal/biz/public/subscription"
	publicticket "github.com/npanel-dev/NPanel-backend/internal/biz/public/ticket"
	publicuser "github.com/npanel-dev/NPanel-backend/internal/biz/public/user"
	publicwithdrawal "github.com/npanel-dev/NPanel-backend/internal/biz/public/withdrawal"
	// Server模块用例
	server "github.com/npanel-dev/NPanel-backend/internal/biz/server"
)

// ProviderSet is biz providers
var ProviderSet = wire.NewSet(
	ads.NewAdsUsecase,
	announcement.NewAnnouncementUsecase,
	application.NewSubscribeApplicationUsecase,
	authmethod.NewAuthMethodUsecase,
	adminconsole.NewConsoleUsecase,
	admincoupon.NewCouponUseCase,
	admindocument.NewDocumentUsecase,
	adminlog.NewSystemLogUsecase,
	adminlog.NewTrafficLogUsecase,
	adminlog.NewLogSettingUsecase,
	adminmarketing.NewMarketingUsecase,
	adminorder.NewOrderUseCase,
	adminpayment.NewPaymentUsecase,
	adminserver.NewServerUsecase,
	adminrouting.NewRoutingUsecase,
	adminserver.NewNodeUsecase,
	adminserver.NewMigrationUsecase,
	adminsubscribe.NewSubscribeUseCase,
	adminsystem.NewSystemUsecase,
	adminticket.NewTicketUseCase,
	adminredemption.NewRedemptionUseCase,
	admintool.NewToolUseCase,
	adminwithdrawal.NewWithdrawalUsecase,
	maingroup.NewGroupUseCase,
	// Admin User模块用例
	adminuser.NewUserUsecase,
	adminuser.NewAuthMethodUsecase,
	adminuser.NewDeviceUsecase,
	adminuser.NewSubscribeUsecase,
	// Auth模块用例
	auth.NewAuthUsecase,
	// Auth OAuth模块用例
	authoauth.NewOAuthUseCase,
	// Common模块用例
	common.NewCommonUsecase,
	// Public Order模块用例
	publicorder.NewOrderUsecase,
	// Public Announcement模块用例
	publicannouncement.NewAnnouncementUseCase,
	// Public Document模块用例
	publicdocument.NewDocumentUseCase,
	// Public Payment模块用例
	publicpayment.NewPaymentUseCase,
	// Public Portal模块用例
	publicportal.NewPortalUseCase,
	// Public Redemption模块用例
	publicredemption.NewRedemptionUseCase,
	// Public Subscribe模块用例
	publicsubscribe.NewSubscribeUseCase,
	// Public Subscription模块用例（订阅配置生成）
	subscription.NewSubscriptionUseCase,
	// Public Ticket模块用例
	publicticket.NewTicketUseCase,
	// Public User模块用例
	publicuser.NewUserUseCase,
	// Public Withdrawal模块用例
	publicwithdrawal.NewWithdrawalUsecase,
	// Server模块用例
	server.NewServerNodeUsecase,
)
