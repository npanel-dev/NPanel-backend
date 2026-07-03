package server

import (
	adsv1 "github.com/npanel-dev/NPanel-backend/api/admin/ads/v1"
	announcementv1 "github.com/npanel-dev/NPanel-backend/api/admin/announcement/v1"
	applicationv1 "github.com/npanel-dev/NPanel-backend/api/admin/application/v1"
	authmethodv1 "github.com/npanel-dev/NPanel-backend/api/admin/authmethod/v1"
	adminv1 "github.com/npanel-dev/NPanel-backend/api/admin/console/v1"
	admincouponv1 "github.com/npanel-dev/NPanel-backend/api/admin/coupon/v1"
	admindocumentv1 "github.com/npanel-dev/NPanel-backend/api/admin/document/v1"
	maingroupv1 "github.com/npanel-dev/NPanel-backend/api/admin/group/v1"
	adminlogv1 "github.com/npanel-dev/NPanel-backend/api/admin/log/v1"
	adminmarketingv1 "github.com/npanel-dev/NPanel-backend/api/admin/marketing/v1"
	adminorderv1 "github.com/npanel-dev/NPanel-backend/api/admin/order/v1"
	adminpaymentv1 "github.com/npanel-dev/NPanel-backend/api/admin/payment/v1"
	adminredemptionv1 "github.com/npanel-dev/NPanel-backend/api/admin/redemption/v1"
	adminroutingv1 "github.com/npanel-dev/NPanel-backend/api/admin/routing/v1"
	adminserverv1 "github.com/npanel-dev/NPanel-backend/api/admin/server/v1"
	adminsubscribev1 "github.com/npanel-dev/NPanel-backend/api/admin/subscribe/v1"
	adminsystemv1 "github.com/npanel-dev/NPanel-backend/api/admin/system/v1"
	adminticketv1 "github.com/npanel-dev/NPanel-backend/api/admin/ticket/v1"
	admintoolv1 "github.com/npanel-dev/NPanel-backend/api/admin/tool/v1"
	adminuserv1 "github.com/npanel-dev/NPanel-backend/api/admin/user/v1"
	adminwithdrawalv1 "github.com/npanel-dev/NPanel-backend/api/admin/withdrawal/v1"
	authoauthv1 "github.com/npanel-dev/NPanel-backend/api/auth/oauth/v1"
	publicauthv1 "github.com/npanel-dev/NPanel-backend/api/public/auth/v1"
	publiccommonv1 "github.com/npanel-dev/NPanel-backend/api/public/common/v1"
	publicorderv1 "github.com/npanel-dev/NPanel-backend/api/public/order/v1"
	publicportalv1 "github.com/npanel-dev/NPanel-backend/api/public/portal/v1"
	publicticketv1 "github.com/npanel-dev/NPanel-backend/api/public/ticket/v1"
	publicuserv1 "github.com/npanel-dev/NPanel-backend/api/public/user/v1"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	appmiddleware "github.com/npanel-dev/NPanel-backend/internal/middleware"
	adsservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/ads"
	announcementservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/announcement"
	applicationservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/application"
	authmethodservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/authmethod"
	adminconsoleservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/console"
	admincouponservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/coupon"
	admindocumentservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/document"
	maingroupservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/group"
	adminlogservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/log"
	adminmarketingservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/marketing"
	adminorderservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/order"
	adminpaymentservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/payment"
	adminredemptionservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/redemption"
	adminroutingservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/routing"
	adminserverservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/server"
	adminsubscribeservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/subscribe"
	adminsystemservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/system"
	adminticketservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/ticket"
	admintoolservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/tool"
	adminuserservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/user"
	adminwithdrawalservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/withdrawal"
	authservice "github.com/npanel-dev/NPanel-backend/internal/service/auth"
	authoauthservice "github.com/npanel-dev/NPanel-backend/internal/service/auth/oauth"
	commonservice "github.com/npanel-dev/NPanel-backend/internal/service/common"
	publicorderservice "github.com/npanel-dev/NPanel-backend/internal/service/public"
	publicportalservice "github.com/npanel-dev/NPanel-backend/internal/service/public/portal"
	publicticketservice "github.com/npanel-dev/NPanel-backend/internal/service/public/ticket"
	publicuserservice "github.com/npanel-dev/NPanel-backend/internal/service/public/user"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server
func NewGRPCServer(c *conf.Server, authMiddleware *appmiddleware.ServiceContext, ads *adsservice.AdsService, announcement *announcementservice.AnnouncementService, application *applicationservice.SubscribeApplicationService, authmethod *authmethodservice.AuthMethodService, adminConsole *adminconsoleservice.ConsoleService, adminCoupon *admincouponservice.CouponService, adminDocument *admindocumentservice.DocumentService, adminLog *adminlogservice.LogService, adminMarketing *adminmarketingservice.MarketingService, adminOrder *adminorderservice.OrderService, adminPayment *adminpaymentservice.PaymentService, adminRouting *adminroutingservice.RoutingService, adminServer *adminserverservice.ServerService, adminSubscribe *adminsubscribeservice.SubscribeService, adminSystem *adminsystemservice.SystemService, adminTicket *adminticketservice.TicketService, adminRedemption *adminredemptionservice.RedemptionService, adminTool *admintoolservice.ToolService, adminWithdrawal *adminwithdrawalservice.WithdrawalService, adminGroup *maingroupservice.GroupService, adminUser *adminuserservice.UserService, adminUserAuthMethod *adminuserservice.UserAuthMethodService, adminUserDevice *adminuserservice.UserDeviceService, adminUserSubscribe *adminuserservice.UserSubscribeService, auth *authservice.AuthService, oauthSvc *authoauthservice.OAuthService, commonSvc *commonservice.CommonService, publicOrder *publicorderservice.PublicOrderService, publicPortal *publicportalservice.PortalService, publicTicket *publicticketservice.TicketService, publicUser *publicuserservice.UserService) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			authMiddleware.Auth(), // 对齐旧项目认证语义
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	adsv1.RegisterAdsServiceServer(srv, ads)
	announcementv1.RegisterAnnouncementServiceServer(srv, announcement)
	applicationv1.RegisterSubscribeApplicationServiceServer(srv, application)
	authmethodv1.RegisterAuthMethodServiceServer(srv, authmethod)
	adminv1.RegisterAdminConsoleServer(srv, adminConsole)
	admincouponv1.RegisterCouponServiceServer(srv, adminCoupon)
	admindocumentv1.RegisterDocumentServiceServer(srv, adminDocument)
	adminlogv1.RegisterLogServiceServer(srv, adminLog)
	adminmarketingv1.RegisterMarketingServiceServer(srv, adminMarketing)
	adminorderv1.RegisterOrderServiceServer(srv, adminOrder)
	adminpaymentv1.RegisterPaymentServiceServer(srv, adminPayment)
	adminroutingv1.RegisterRoutingServiceServer(srv, adminRouting)
	adminserverv1.RegisterServerServiceServer(srv, adminServer)
	adminsubscribev1.RegisterSubscribeServer(srv, adminSubscribe)
	adminsystemv1.RegisterSystemServiceServer(srv, adminSystem)
	adminticketv1.RegisterTicketServer(srv, adminTicket)
	adminredemptionv1.RegisterRedemptionServer(srv, adminRedemption)
	admintoolv1.RegisterToolServer(srv, adminTool)
	adminwithdrawalv1.RegisterWithdrawalServiceServer(srv, adminWithdrawal)
	maingroupv1.RegisterGroupServer(srv, adminGroup)
	// Admin User模块服务注册
	adminuserv1.RegisterUserServiceServer(srv, adminUser)
	adminuserv1.RegisterUserAuthMethodServiceServer(srv, adminUserAuthMethod)
	adminuserv1.RegisterUserDeviceServiceServer(srv, adminUserDevice)
	adminuserv1.RegisterUserSubscribeServiceServer(srv, adminUserSubscribe)
	// Auth模块服务注册
	publicauthv1.RegisterAuthServer(srv, auth)
	// Auth OAuth模块服务注册
	authoauthv1.RegisterOAuthServer(srv, oauthSvc)
	// Common模块服务注册
	publiccommonv1.RegisterCommonServer(srv, commonSvc)
	// Public Order模块服务注册
	publicorderv1.RegisterPublicOrderServer(srv, publicOrder)
	// Public Portal模块服务注册
	publicportalv1.RegisterPortalServer(srv, publicPortal)
	// Public Ticket模块服务注册
	publicticketv1.RegisterTicketServer(srv, publicTicket)
	// Public User模块服务注册
	publicuserv1.RegisterPublicUserServer(srv, publicUser)
	return srv
}
