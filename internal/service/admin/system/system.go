package system

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	pb "github.com/npanel-dev/NPanel-backend/api/admin/system/v1"
	systembiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/system"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

type SystemService struct {
	pb.UnimplementedSystemServiceServer

	uc  *systembiz.SystemUsecase
	log *log.Helper
}

func NewSystemService(uc *systembiz.SystemUsecase, logger log.Logger) *SystemService {
	return &SystemService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

func (s *SystemService) GetSystemModule(ctx context.Context, req *pb.GetSystemModuleRequest) (*pb.GetSystemModuleReply, error) {
	module, err := s.uc.GetSystemModule(ctx)
	if err != nil {
		s.log.Errorf("Failed to get system module: %v", err)
		return nil, err
	}

	return &pb.GetSystemModuleReply{
		Code:    200,
		Message: "success",
		Data: &pb.SystemModule{
			ServiceName:    module.ServiceName,
			ServiceVersion: module.ServiceVersion,
			Secret:         module.Secret,
		},
	}, nil
}

// GetCurrencyConfig 获取货币配置
func (s *SystemService) GetCurrencyConfig(ctx context.Context, req *pb.GetCurrencyConfigRequest) (*pb.GetCurrencyConfigReply, error) {

	config, err := s.uc.GetCurrencyConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get currency config: %v", err)
		return nil, err
	}

	return &pb.GetCurrencyConfigReply{
		Code:    responsecode.AdminGetCurrencyConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetCurrencyConfigSuccess],
		Data: &pb.CurrencyConfig{
			AccessKey:      config.AccessKey,
			CurrencyUnit:   config.CurrencyUnit,
			CurrencySymbol: config.CurrencySymbol,
		},
	}, nil
}

// UpdateCurrencyConfig 更新货币配置
func (s *SystemService) UpdateCurrencyConfig(ctx context.Context, req *pb.UpdateCurrencyConfigRequest) (*pb.UpdateCurrencyConfigReply, error) {

	config := &systembiz.CurrencyConfig{
		AccessKey:      req.AccessKey,
		CurrencyUnit:   req.CurrencyUnit,
		CurrencySymbol: req.CurrencySymbol,
	}

	if err := s.uc.UpdateCurrencyConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update currency config: %v", err)
		return nil, err
	}

	return &pb.UpdateCurrencyConfigReply{
		Code:    responsecode.AdminUpdateCurrencyConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateCurrencyConfigSuccess],
		Data:    &pb.UpdateCurrencyConfigData{Success: true},
	}, nil
}

// GetInviteConfig 获取邀请配置
func (s *SystemService) GetInviteConfig(ctx context.Context, req *pb.GetInviteConfigRequest) (*pb.GetInviteConfigReply, error) {

	config, err := s.uc.GetInviteConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get invite config: %v", err)
		return nil, err
	}

	return &pb.GetInviteConfigReply{
		Code:    responsecode.AdminGetInviteConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetInviteConfigSuccess],
		Data: &pb.InviteConfig{
			ForcedInvite:        config.ForcedInvite,
			ReferralPercentage:  int64(config.ReferralPercentage),
			OnlyFirstPurchase:   config.OnlyFirstPurchase,
			WithdrawalMinAmount: config.WithdrawalMinAmount,
			WithdrawalMethods:   config.WithdrawalMethods,
		},
	}, nil
}

// UpdateInviteConfig 更新邀请配置
func (s *SystemService) UpdateInviteConfig(ctx context.Context, req *pb.UpdateInviteConfigRequest) (*pb.UpdateInviteConfigReply, error) {

	config := &systembiz.InviteConfig{
		ForcedInvite:        req.ForcedInvite,
		ReferralPercentage:  int(req.ReferralPercentage),
		OnlyFirstPurchase:   req.OnlyFirstPurchase,
		WithdrawalMinAmount: req.WithdrawalMinAmount,
		WithdrawalMethods:   req.WithdrawalMethods,
	}

	if err := s.uc.UpdateInviteConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update invite config: %v", err)
		return nil, err
	}

	return &pb.UpdateInviteConfigReply{
		Code:    responsecode.AdminUpdateInviteConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateInviteConfigSuccess],
		Data:    &pb.UpdateInviteConfigData{Success: true},
	}, nil
}

// GetNodeConfig 获取节点配置
func (s *SystemService) GetNodeConfig(ctx context.Context, req *pb.GetNodeConfigRequest) (*pb.GetNodeConfigReply, error) {

	config, err := s.uc.GetNodeConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get node config: %v", err)
		return nil, err
	}

	// Convert biz NodeConfig to pb NodeConfig
	return &pb.GetNodeConfigReply{
		Code:    responsecode.AdminGetNodeConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetNodeConfigSuccess],
		Data:    convertBizNodeConfigToPb(config),
	}, nil
}

// UpdateNodeConfig 更新节点配置
func (s *SystemService) UpdateNodeConfig(ctx context.Context, req *pb.UpdateNodeConfigRequest) (*pb.UpdateNodeConfigReply, error) {

	config := convertPbNodeConfigToBiz(req)

	if err := s.uc.UpdateNodeConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update node config: %v", err)
		return nil, err
	}

	return &pb.UpdateNodeConfigReply{
		Code:    responsecode.AdminUpdateNodeConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateNodeConfigSuccess],
		Data:    &pb.UpdateNodeConfigData{Success: true},
	}, nil
}

// GetPrivacyPolicyConfig 获取隐私政策配置
func (s *SystemService) GetPrivacyPolicyConfig(ctx context.Context, req *pb.GetPrivacyPolicyConfigRequest) (*pb.GetPrivacyPolicyConfigReply, error) {

	config, err := s.uc.GetPrivacyPolicyConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get privacy policy config: %v", err)
		return nil, err
	}

	return &pb.GetPrivacyPolicyConfigReply{
		Code:    responsecode.AdminGetPrivacyPolicyConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetPrivacyPolicyConfigSuccess],
		Data: &pb.PrivacyPolicyConfig{
			PrivacyPolicy: config.PrivacyPolicy,
		},
	}, nil
}

// UpdatePrivacyPolicyConfig 更新隐私政策配置
func (s *SystemService) UpdatePrivacyPolicyConfig(ctx context.Context, req *pb.UpdatePrivacyPolicyConfigRequest) (*pb.UpdatePrivacyPolicyConfigReply, error) {

	config := &systembiz.PrivacyPolicyConfig{
		PrivacyPolicy: req.PrivacyPolicy,
	}

	if err := s.uc.UpdatePrivacyPolicyConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update privacy policy config: %v", err)
		return nil, err
	}

	return &pb.UpdatePrivacyPolicyConfigReply{
		Code:    responsecode.AdminUpdatePrivacyPolicyConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdatePrivacyPolicyConfigSuccess],
		Data:    &pb.UpdatePrivacyPolicyConfigData{Success: true},
	}, nil
}

// convertBizNodeConfigToPb converts biz NodeConfig to pb NodeConfig
func convertBizNodeConfigToPb(config *systembiz.NodeConfig) *pb.NodeConfig {
	pbConfig := &pb.NodeConfig{
		NodeSecret:             config.NodeSecret,
		NodePullInterval:       int64(config.NodePullInterval),
		NodePushInterval:       int64(config.NodePushInterval),
		TrafficReportThreshold: int64(config.TrafficReportThreshold),
		IpStrategy:             config.IPStrategy,
		DeviceAdmissionEnabled: config.DeviceAdmissionEnabled,
		DeviceCountMode:        config.DeviceCountMode,
	}

	// Parse JSON fields
	if config.DNS != "" {
		var dnsList []systembiz.NodeDNS
		if err := json.Unmarshal([]byte(config.DNS), &dnsList); err == nil {
			for _, dns := range dnsList {
				pbConfig.Dns = append(pbConfig.Dns, &pb.NodeDNS{
					Proto:   dns.Proto,
					Address: dns.Address,
					Domains: dns.Domains,
				})
			}
		}
	}

	if config.Block != "" {
		var blockList []string
		if err := json.Unmarshal([]byte(config.Block), &blockList); err == nil {
			pbConfig.Block = blockList
		}
	}

	if config.Outbound != "" {
		var outboundList []systembiz.NodeOutbound
		if err := json.Unmarshal([]byte(config.Outbound), &outboundList); err == nil {
			for _, outbound := range outboundList {
				pbConfig.Outbound = append(pbConfig.Outbound, &pb.NodeOutbound{
					Name:     outbound.Name,
					Protocol: outbound.Protocol,
					Address:  outbound.Address,
					Port:     int64(outbound.Port),
					Password: outbound.Password,
					Rules:    outbound.Rules,
				})
			}
		}
	}

	return pbConfig
}

// convertPbNodeConfigToBiz converts pb NodeConfig request to biz NodeConfig
func convertPbNodeConfigToBiz(req *pb.UpdateNodeConfigRequest) *systembiz.NodeConfig {
	config := &systembiz.NodeConfig{
		NodeSecret:             req.NodeSecret,
		NodePullInterval:       int(req.NodePullInterval),
		NodePushInterval:       int(req.NodePushInterval),
		TrafficReportThreshold: int(req.TrafficReportThreshold),
		IPStrategy:             req.IpStrategy,
		DeviceAdmissionEnabled: req.DeviceAdmissionEnabled,
		DeviceCountMode:        req.DeviceCountMode,
	}

	// Convert DNS list to JSON
	if len(req.Dns) > 0 {
		var dnsList []systembiz.NodeDNS
		for _, dns := range req.Dns {
			dnsList = append(dnsList, systembiz.NodeDNS{
				Proto:   dns.Proto,
				Address: dns.Address,
				Domains: dns.Domains,
			})
		}
		if data, err := json.Marshal(dnsList); err == nil {
			config.DNS = string(data)
		}
	}

	// Convert Block list to JSON
	if len(req.Block) > 0 {
		if data, err := json.Marshal(req.Block); err == nil {
			config.Block = string(data)
		}
	}

	// Convert Outbound list to JSON
	if len(req.Outbound) > 0 {
		var outboundList []systembiz.NodeOutbound
		for _, outbound := range req.Outbound {
			outboundList = append(outboundList, systembiz.NodeOutbound{
				Name:     outbound.Name,
				Protocol: outbound.Protocol,
				Address:  outbound.Address,
				Port:     int(outbound.Port),
				Password: outbound.Password,
				Rules:    outbound.Rules,
			})
		}
		if data, err := json.Marshal(outboundList); err == nil {
			config.Outbound = string(data)
		}
	}

	return config
}

// GetRegisterConfig 获取注册配置
func (s *SystemService) GetRegisterConfig(ctx context.Context, req *pb.GetRegisterConfigRequest) (*pb.GetRegisterConfigReply, error) {

	config, err := s.uc.GetRegisterConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get register config: %v", err)
		return nil, err
	}

	return &pb.GetRegisterConfigReply{
		Code:    responsecode.AdminGetRegisterConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetRegisterConfigSuccess],
		Data: &pb.RegisterConfig{
			StopRegister:            config.StopRegister,
			EnableTrial:             config.EnableTrial,
			TrialSubscribe:          int64(config.TrialSubscribe),
			TrialTime:               int64(config.TrialTime),
			TrialTimeUnit:           config.TrialTimeUnit,
			EnableIpRegisterLimit:   config.EnableIpRegisterLimit,
			IpRegisterLimit:         int64(config.IpRegisterLimit),
			IpRegisterLimitDuration: int64(config.IpRegisterLimitDuration),
			DeviceLimit:             int64(config.DeviceLimit),
		},
	}, nil
}

// UpdateRegisterConfig 更新注册配置
func (s *SystemService) UpdateRegisterConfig(ctx context.Context, req *pb.UpdateRegisterConfigRequest) (*pb.UpdateRegisterConfigReply, error) {

	config := &systembiz.RegisterConfig{
		StopRegister:            req.StopRegister,
		EnableTrial:             req.EnableTrial,
		TrialSubscribe:          int(req.TrialSubscribe),
		TrialTime:               int(req.TrialTime),
		TrialTimeUnit:           req.TrialTimeUnit,
		EnableIpRegisterLimit:   req.EnableIpRegisterLimit,
		IpRegisterLimit:         int(req.IpRegisterLimit),
		IpRegisterLimitDuration: int(req.IpRegisterLimitDuration),
		DeviceLimit:             int(req.DeviceLimit),
	}

	if err := s.uc.UpdateRegisterConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update register config: %v", err)
		return nil, err
	}

	return &pb.UpdateRegisterConfigReply{
		Code:    responsecode.AdminUpdateRegisterConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateRegisterConfigSuccess],
		Data:    &pb.UpdateRegisterConfigData{Success: true},
	}, nil
}

// GetSiteConfig 获取站点配置
func (s *SystemService) GetSiteConfig(ctx context.Context, req *pb.GetSiteConfigRequest) (*pb.GetSiteConfigReply, error) {

	config, err := s.uc.GetSiteConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get site config: %v", err)
		return nil, err
	}

	return &pb.GetSiteConfigReply{
		Code:    responsecode.AdminGetSiteConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetSiteConfigSuccess],
		Data: &pb.SiteConfig{
			Host:       config.Host,
			SiteName:   config.SiteName,
			SiteDesc:   config.SiteDesc,
			SiteLogo:   config.SiteLogo,
			Keywords:   config.Keywords,
			CustomHtml: config.CustomHTML,
			CustomData: config.CustomData,
		},
	}, nil
}

// UpdateSiteConfig 更新站点配置
func (s *SystemService) UpdateSiteConfig(ctx context.Context, req *pb.UpdateSiteConfigRequest) (*pb.UpdateSiteConfigReply, error) {

	config := &systembiz.SiteConfig{
		Host:       req.Host,
		SiteName:   req.SiteName,
		SiteDesc:   req.SiteDesc,
		SiteLogo:   req.SiteLogo,
		Keywords:   req.Keywords,
		CustomHTML: req.CustomHtml,
		CustomData: req.CustomData,
	}

	if err := s.uc.UpdateSiteConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update site config: %v", err)
		return nil, err
	}

	return &pb.UpdateSiteConfigReply{
		Code:    responsecode.AdminUpdateSiteConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateSiteConfigSuccess],
		Data:    &pb.UpdateSiteConfigData{Success: true},
	}, nil
}

// GetTawkConfig 获取Tawk客服配置
func (s *SystemService) GetTawkConfig(ctx context.Context, req *pb.GetTawkConfigRequest) (*pb.GetTawkConfigReply, error) {
	config, err := s.uc.GetTawkConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get tawk config: %v", err)
		return nil, err
	}

	return &pb.GetTawkConfigReply{
		Code:    responsecode.AdminGetSiteConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetSiteConfigSuccess],
		Data: &pb.AdminTawkConfig{
			Enabled:      config.Enabled,
			PropertyId:   config.PropertyID,
			WidgetId:     config.WidgetID,
			IdentifyUser: config.IdentifyUser,
			SecureMode:   config.SecureMode,
			SecretKey:    config.SecretKey,
		},
	}, nil
}

// UpdateTawkConfig 更新Tawk客服配置
func (s *SystemService) UpdateTawkConfig(ctx context.Context, req *pb.UpdateTawkConfigRequest) (*pb.UpdateTawkConfigReply, error) {
	config := &systembiz.TawkConfig{
		Enabled:      req.Enabled,
		PropertyID:   req.PropertyId,
		WidgetID:     req.WidgetId,
		IdentifyUser: req.IdentifyUser,
		SecureMode:   req.SecureMode,
		SecretKey:    req.SecretKey,
	}

	if err := s.uc.UpdateTawkConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update tawk config: %v", err)
		return nil, err
	}

	return &pb.UpdateTawkConfigReply{
		Code:    responsecode.AdminUpdateSiteConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateSiteConfigSuccess],
		Data:    &pb.UpdateTawkConfigData{Success: true},
	}, nil
}

// GetSubscribeConfig 获取订阅配置
func (s *SystemService) GetSubscribeConfig(ctx context.Context, req *pb.GetSubscribeConfigRequest) (*pb.GetSubscribeConfigReply, error) {

	config, err := s.uc.GetSubscribeConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get subscribe config: %v", err)
		return nil, err
	}

	return &pb.GetSubscribeConfigReply{
		Code:    responsecode.AdminGetSubscribeConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetSubscribeConfigSuccess],
		Data: &pb.SubscribeConfig{
			SingleModel:     config.SingleModel,
			SubscribePath:   config.SubscribePath,
			SubscribeDomain: config.SubscribeDomain,
			PanDomain:       config.PanDomain,
			UserAgentLimit:  config.UserAgentLimit,
			UserAgentList:   config.UserAgentList,
		},
	}, nil
}

// UpdateSubscribeConfig 更新订阅配置
func (s *SystemService) UpdateSubscribeConfig(ctx context.Context, req *pb.UpdateSubscribeConfigRequest) (*pb.UpdateSubscribeConfigReply, error) {

	config := &systembiz.SubscribeConfig{
		SingleModel:     req.SingleModel,
		SubscribePath:   req.SubscribePath,
		SubscribeDomain: req.SubscribeDomain,
		PanDomain:       req.PanDomain,
		UserAgentLimit:  req.UserAgentLimit,
		UserAgentList:   req.UserAgentList,
	}

	if err := s.uc.UpdateSubscribeConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update subscribe config: %v", err)
		return nil, err
	}

	return &pb.UpdateSubscribeConfigReply{
		Code:    responsecode.AdminUpdateSubscribeConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateSubscribeConfigSuccess],
		Data:    &pb.UpdateSubscribeConfigData{Success: true},
	}, nil
}

// GetTosConfig 获取服务条款配置
func (s *SystemService) GetTosConfig(ctx context.Context, req *pb.GetTosConfigRequest) (*pb.GetTosConfigReply, error) {

	config, err := s.uc.GetTosConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get tos config: %v", err)
		return nil, err
	}

	return &pb.GetTosConfigReply{
		Code:    responsecode.AdminGetTosConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetTosConfigSuccess],
		Data: &pb.TosConfig{
			TosContent: config.TosContent,
		},
	}, nil
}

// UpdateTosConfig 更新服务条款配置
func (s *SystemService) UpdateTosConfig(ctx context.Context, req *pb.UpdateTosConfigRequest) (*pb.UpdateTosConfigReply, error) {

	config := &systembiz.TosConfig{
		TosContent: req.TosContent,
	}

	if err := s.uc.UpdateTosConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update tos config: %v", err)
		return nil, err
	}

	return &pb.UpdateTosConfigReply{
		Code:    responsecode.AdminUpdateTosConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateTosConfigSuccess],
		Data:    &pb.UpdateTosConfigData{Success: true},
	}, nil
}

// GetVerifyCodeConfig 获取验证码配置
func (s *SystemService) GetVerifyCodeConfig(ctx context.Context, req *pb.GetVerifyCodeConfigRequest) (*pb.GetVerifyCodeConfigReply, error) {

	config, err := s.uc.GetVerifyCodeConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get verify code config: %v", err)
		return nil, err
	}

	return &pb.GetVerifyCodeConfigReply{
		Code:    responsecode.AdminGetVerifyCodeConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetVerifyCodeConfigSuccess],
		Data: &pb.VerifyCodeConfig{
			VerifyCodeExpireTime: int64(config.VerifyCodeExpireTime),
			VerifyCodeLimit:      int64(config.VerifyCodeLimit),
			VerifyCodeInterval:   int64(config.VerifyCodeInterval),
		},
	}, nil
}

// UpdateVerifyCodeConfig 更新验证码配置
func (s *SystemService) UpdateVerifyCodeConfig(ctx context.Context, req *pb.UpdateVerifyCodeConfigRequest) (*pb.UpdateVerifyCodeConfigReply, error) {

	config := &systembiz.VerifyCodeConfig{
		VerifyCodeExpireTime: int(req.VerifyCodeExpireTime),
		VerifyCodeLimit:      int(req.VerifyCodeLimit),
		VerifyCodeInterval:   int(req.VerifyCodeInterval),
	}

	if err := s.uc.UpdateVerifyCodeConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update verify code config: %v", err)
		return nil, err
	}

	return &pb.UpdateVerifyCodeConfigReply{
		Code:    responsecode.AdminUpdateVerifyCodeConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateVerifyCodeConfigSuccess],
		Data:    &pb.UpdateVerifyCodeConfigData{Success: true},
	}, nil
}

// GetVerifyConfig 获取验证配置
func (s *SystemService) GetVerifyConfig(ctx context.Context, req *pb.GetVerifyConfigRequest) (*pb.GetVerifyConfigReply, error) {

	config, err := s.uc.GetVerifyConfig(ctx)
	if err != nil {
		s.log.Errorf("Failed to get verify config: %v", err)
		return nil, err
	}

	return &pb.GetVerifyConfigReply{
		Code:    responsecode.AdminGetVerifyConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetVerifyConfigSuccess],
		Data: &pb.VerifyConfig{
			CaptchaType:                    config.CaptchaType,
			TurnstileSiteKey:               config.TurnstileSiteKey,
			TurnstileSecret:                config.TurnstileSecret,
			EnableUserLoginCaptcha:         config.EnableUserLoginCaptcha,
			EnableUserRegisterCaptcha:      config.EnableUserRegisterCaptcha,
			EnableAdminLoginCaptcha:        config.EnableAdminLoginCaptcha,
			EnableUserResetPasswordCaptcha: config.EnableUserResetPasswordCaptcha,
		},
	}, nil
}

// UpdateVerifyConfig 更新验证配置
func (s *SystemService) UpdateVerifyConfig(ctx context.Context, req *pb.UpdateVerifyConfigRequest) (*pb.UpdateVerifyConfigReply, error) {

	config := &systembiz.VerifyConfig{
		CaptchaType:                    req.CaptchaType,
		TurnstileSiteKey:               req.TurnstileSiteKey,
		TurnstileSecret:                req.TurnstileSecret,
		EnableUserLoginCaptcha:         req.EnableUserLoginCaptcha,
		EnableUserRegisterCaptcha:      req.EnableUserRegisterCaptcha,
		EnableAdminLoginCaptcha:        req.EnableAdminLoginCaptcha,
		EnableUserResetPasswordCaptcha: req.EnableUserResetPasswordCaptcha,
	}

	if err := s.uc.UpdateVerifyConfig(ctx, config); err != nil {
		s.log.Errorf("Failed to update verify config: %v", err)
		return nil, err
	}

	return &pb.UpdateVerifyConfigReply{
		Code:    responsecode.AdminUpdateVerifyConfigSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminUpdateVerifyConfigSuccess],
		Data:    &pb.UpdateVerifyConfigData{Success: true},
	}, nil
}

// GetNodeMultiplier 获取节点倍率
func (s *SystemService) GetNodeMultiplier(ctx context.Context, req *pb.GetNodeMultiplierRequest) (*pb.GetNodeMultiplierReply, error) {

	periods, err := s.uc.GetNodeMultiplier(ctx)
	if err != nil {
		s.log.Errorf("Failed to get node multiplier: %v", err)
		return nil, err
	}

	var pbPeriods []*pb.TimePeriod
	for _, period := range periods {
		pbPeriods = append(pbPeriods, &pb.TimePeriod{
			StartTime:  period.StartTime,
			EndTime:    period.EndTime,
			Multiplier: period.Multiplier,
		})
	}

	return &pb.GetNodeMultiplierReply{
		Code:    responsecode.AdminGetNodeMultiplierSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminGetNodeMultiplierSuccess],
		Data:    &pb.GetNodeMultiplierData{Periods: pbPeriods},
	}, nil
}

// PreViewNodeMultiplier 预览节点倍率
func (s *SystemService) PreViewNodeMultiplier(ctx context.Context, req *pb.PreViewNodeMultiplierRequest) (*pb.PreViewNodeMultiplierReply, error) {

	periods, err := s.uc.GetNodeMultiplier(ctx)
	if err != nil {
		s.log.Errorf("Failed to get node multiplier for preview: %v", err)
		return nil, err
	}

	// Calculate current time multiplier
	now := time.Now()
	currentTime := now.Format("2006-01-02 15:04:05")
	ratio := systembiz.PreviewNodeMultiplier(now, periods)

	return &pb.PreViewNodeMultiplierReply{
		Code:    responsecode.AdminPreViewNodeMultiplierSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminPreViewNodeMultiplierSuccess],
		Data: &pb.PreViewNodeMultiplierData{CurrentTime: currentTime,
			Ratio: ratio},
	}, nil
}

// SetNodeMultiplier 设置节点倍率
func (s *SystemService) SetNodeMultiplier(ctx context.Context, req *pb.SetNodeMultiplierRequest) (*pb.SetNodeMultiplierReply, error) {

	var periods []systembiz.TimePeriod
	for _, pbPeriod := range req.Periods {
		periods = append(periods, systembiz.TimePeriod{
			StartTime:  pbPeriod.StartTime,
			EndTime:    pbPeriod.EndTime,
			Multiplier: pbPeriod.Multiplier,
		})
	}

	if err := s.uc.SetNodeMultiplier(ctx, periods); err != nil {
		s.log.Errorf("Failed to set node multiplier: %v", err)
		return nil, err
	}

	return &pb.SetNodeMultiplierReply{
		Code:    responsecode.AdminSetNodeMultiplierSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminSetNodeMultiplierSuccess],
		Data:    &pb.SetNodeMultiplierData{Success: true},
	}, nil
}

// SettingTelegramBot 设置Telegram机器人
func (s *SystemService) SettingTelegramBot(ctx context.Context, req *pb.SettingTelegramBotRequest) (*pb.SettingTelegramBotReply, error) {
	if err := s.uc.ApplyTelegramBot(ctx); err != nil {
		s.log.Errorf("Failed to apply telegram bot: %v", err)
		return nil, err
	}
	return &pb.SettingTelegramBotReply{
		Code:    responsecode.AdminSettingTelegramBotSuccess,
		Message: responsecode.CodeMessages[responsecode.AdminSettingTelegramBotSuccess],
		Data:    &pb.SettingTelegramBotData{Success: true},
	}, nil
}
