package common

import (
	"context"
	"strconv"
	"time"

	pb "github.com/npanel-dev/NPanel-backend/api/public/common/v1"
	"github.com/npanel-dev/NPanel-backend/internal/biz/common"
)

type CommonService struct {
	pb.UnimplementedCommonServer

	uc *common.CommonUsecase
}

func NewCommonService(uc *common.CommonUsecase) *CommonService {
	return &CommonService{
		uc: uc,
	}
}

// GetAds gets ads list
func (s *CommonService) GetAds(ctx context.Context, req *pb.GetAdsRequest) (*pb.GetAdsReply, error) {
	adsList, err := s.uc.GetAds(ctx, req.Device, req.Position)
	if err != nil {
		return nil, err
	}

	// Convert biz objects to proto objects
	pbAds := make([]*pb.Ads, len(adsList))
	for i, ad := range adsList {
		pbAds[i] = &pb.Ads{
			Id:          ad.ID,
			Title:       ad.Title,
			Type:        ad.Type,
			Content:     ad.Content,
			Description: ad.Description,
			TargetUrl:   ad.TargetURL,
			StartTime:   ad.StartTime,
			EndTime:     ad.EndTime,
			Status:      int32(ad.Status),
			CreatedAt:   ad.CreatedAt,
			UpdatedAt:   ad.UpdatedAt,
		}
	}

	return &pb.GetAdsReply{
		List: pbAds,
	}, nil
}

// GetClient gets subscribe client list
func (s *CommonService) GetClient(ctx context.Context, req *pb.GetClientRequest) (*pb.GetClientReply, error) {
	clientList, total, err := s.uc.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	// Convert biz objects to proto objects
	pbClients := make([]*pb.SubscribeClient, len(clientList))
	for i, client := range clientList {
		pbClients[i] = &pb.SubscribeClient{
			Id:          client.ID,
			Name:        client.Name,
			Description: client.Description,
			Icon:        client.Icon,
			Scheme:      client.Scheme,
			IsDefault:   client.IsDefault,
			DownloadLink: &pb.DownloadLink{
				Ios:     client.DownloadLink.IOS,
				Android: client.DownloadLink.Android,
				Windows: client.DownloadLink.Windows,
				Mac:     client.DownloadLink.Mac,
				Linux:   client.DownloadLink.Linux,
				Harmony: client.DownloadLink.Harmony,
			},
		}
	}

	return &pb.GetClientReply{
		Total: total,
		List:  pbClients,
	}, nil
}

// GetPrivacyPolicy gets privacy policy content
func (s *CommonService) GetPrivacyPolicy(ctx context.Context, req *pb.GetPrivacyPolicyRequest) (*pb.GetPrivacyPolicyReply, error) {
	content, err := s.uc.GetPrivacyPolicy(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetPrivacyPolicyReply{
		PrivacyPolicy: content,
	}, nil
}

// GetTos gets terms of service content
func (s *CommonService) GetTos(ctx context.Context, req *pb.GetTosRequest) (*pb.GetTosReply, error) {
	content, err := s.uc.GetTos(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetTosReply{
		TosContent: content,
	}, nil
}

// GetGlobalConfig gets global configuration
func (s *CommonService) GetGlobalConfig(ctx context.Context, req *pb.GetGlobalConfigRequest) (*pb.GetGlobalConfigReply, error) {
	config, err := s.uc.GetGlobalConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Convert config to proto structures with nil checks
	configData := &pb.GetGlobalConfigReply{}

	// Site config
	if config.Site != nil {
		configData.Site = &pb.SiteConfig{
			Host:       config.Site.Host,
			SiteName:   config.Site.SiteName,
			SiteDesc:   config.Site.SiteDesc,
			SiteLogo:   config.Site.SiteLogo,
			Keywords:   config.Site.Keywords,
			CustomHtml: config.Site.CustomHtml,
			CustomData: config.Site.CustomData,
		}
	} else {
		configData.Site = &pb.SiteConfig{}
	}

	// Verify config
	if config.Verify != nil {
		configData.Verify = &pb.VerifyConfig{
			CaptchaType:                    config.Verify.CaptchaType,
			TurnstileSiteKey:               config.Verify.TurnstileSiteKey,
			EnableUserLoginCaptcha:         config.Verify.EnableUserLoginVerify,
			EnableUserRegisterCaptcha:      config.Verify.EnableUserRegisterVerify,
			EnableAdminLoginCaptcha:        config.Verify.EnableAdminLoginCaptcha,
			EnableUserResetPasswordCaptcha: config.Verify.EnableUserResetPasswordCaptcha,
		}
	} else {
		configData.Verify = &pb.VerifyConfig{}
	}

	// Auth config
	configData.Auth = &pb.AuthConfig{
		Mobile:   &pb.MobileAuthConfig{},
		Email:    &pb.EmailAuthConfig{},
		Device:   &pb.DeviceAuthConfig{},
		Register: &pb.RegisterConfig{},
	}
	if config.Auth != nil {
		if config.Auth.Mobile != nil {
			configData.Auth.Mobile = &pb.MobileAuthConfig{
				Enable:          config.Auth.Mobile.Enable,
				EnableWhitelist: config.Auth.Mobile.EnableWhitelist,
				Whitelist:       config.Auth.Mobile.Whitelist,
			}
		}
		if config.Auth.Email != nil {
			configData.Auth.Email = &pb.EmailAuthConfig{
				Enable:             config.Auth.Email.Enable,
				EnableVerify:       config.Auth.Email.EnableVerify,
				EnableDomainSuffix: config.Auth.Email.EnableDomainSuffix,
				DomainSuffixList:   config.Auth.Email.DomainSuffixList,
			}
		}
		if config.Auth.Device != nil {
			configData.Auth.Device = &pb.DeviceAuthConfig{
				Enable:         config.Auth.Device.Enable,
				ShowAds:        config.Auth.Device.ShowAds,
				EnableSecurity: config.Auth.Device.EnableSecurity,
				OnlyRealDevice: config.Auth.Device.OnlyRealDevice,
			}
		}
		if config.Auth.Register != nil {
			configData.Auth.Register = &pb.RegisterConfig{
				StopRegister:            config.Auth.Register.StopRegister,
				EnableIpRegisterLimit:   config.Auth.Register.EnableIpRegisterLimit,
				IpRegisterLimit:         config.Auth.Register.IpRegisterLimit,
				IpRegisterLimitDuration: config.Auth.Register.IpRegisterLimitDuration,
			}
		}
	}

	// Invite config
	if config.Invite != nil {
		configData.Invite = &pb.InviteConfig{
			ForcedInvite:        config.Invite.ForcedInvite,
			ReferralPercentage:  config.Invite.ReferralPercentage,
			OnlyFirstPurchase:   config.Invite.OnlyFirstPurchase,
			WithdrawalMinAmount: config.Invite.WithdrawalMinAmount,
			WithdrawalMethods:   config.Invite.WithdrawalMethods,
		}
	} else {
		configData.Invite = &pb.InviteConfig{}
	}

	// Currency config
	configData.Currency = &pb.CurrencyConfig{
		CurrencyUnit:   getMapValue(config.Currency, "CurrencyUnit", "currency_unit"),
		CurrencySymbol: getMapValue(config.Currency, "CurrencySymbol", "currency_symbol"),
	}

	// Subscribe config
	if config.Subscribe != nil {
		configData.Subscribe = &pb.SubscribeConfig{
			SingleModel:     config.Subscribe.SingleModel,
			SubscribePath:   config.Subscribe.SubscribePath,
			SubscribeDomain: config.Subscribe.SubscribeDomain,
			PanDomain:       config.Subscribe.PanDomain,
			UserAgentLimit:  config.Subscribe.UserAgentLimit,
			UserAgentList:   config.Subscribe.UserAgentList,
		}
	} else {
		configData.Subscribe = &pb.SubscribeConfig{}
	}

	// Tawk config. Secret key is intentionally not exposed in public config.
	if config.Tawk != nil {
		configData.Tawk = &pb.PublicTawkConfig{
			Enabled:      config.Tawk.Enabled,
			PropertyId:   config.Tawk.PropertyID,
			WidgetId:     config.Tawk.WidgetID,
			IdentifyUser: config.Tawk.IdentifyUser,
			SecureMode:   config.Tawk.SecureMode,
		}
	} else {
		configData.Tawk = &pb.PublicTawkConfig{}
	}

	// Verify code config
	configData.VerifyCode = &pb.PublicVerifyCodeConfig{
		VerifyCodeInterval: 60, // Default 60 seconds
	}

	// Try to parse verify_code_interval from database config
	if interval := getMapValue(config.VerifyCode, "VerifyCodeInterval", "verify_code_interval"); interval != "" {
		if val, err := strconv.ParseInt(interval, 10, 64); err == nil {
			configData.VerifyCode.VerifyCodeInterval = val
		}
	}

	configData.OauthMethods = config.OAuthMethods
	configData.WebAd = config.WebAd

	return configData, nil
}

func getMapValue(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != "" {
			return value
		}
	}
	return ""
}

// Heartbeat returns service liveness info
func (s *CommonService) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatReply, error) {
	return &pb.HeartbeatReply{
		Status:    true,
		Message:   "service is alive",
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetStat gets system statistics
func (s *CommonService) GetStat(ctx context.Context, req *pb.GetStatRequest) (*pb.GetStatReply, error) {
	stat, err := s.uc.GetStat(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetStatReply{
		User:     stat.User,
		Node:     stat.Node,
		Country:  stat.Country,
		Protocol: stat.Protocol,
	}, nil
}

// SendEmailCode sends email verification code
func (s *CommonService) SendEmailCode(ctx context.Context, req *pb.SendEmailCodeRequest) (*pb.SendCodeReply, error) {
	if err := s.uc.SendEmailCode(ctx, req.Email, req.Type); err != nil {
		return nil, err
	}

	return &pb.SendCodeReply{Status: true}, nil
}

// SendSmsCode sends SMS verification code
func (s *CommonService) SendSmsCode(ctx context.Context, req *pb.SendSmsCodeRequest) (*pb.SendCodeReply, error) {
	if _, err := s.uc.SendSmsCode(ctx, req.Telephone, req.TelephoneAreaCode, req.Type); err != nil {
		return nil, err
	}

	return &pb.SendCodeReply{Status: true}, nil
}

// CheckVerificationCode checks verification code
func (s *CommonService) CheckVerificationCode(ctx context.Context, req *pb.CheckVerificationCodeRequest) (*pb.CheckVerificationCodeReply, error) {
	valid, err := s.uc.CheckVerificationCode(ctx, req.Method, req.Account, req.Code, req.Type)
	if err != nil {
		return nil, err
	}

	return &pb.CheckVerificationCodeReply{
		Status: valid,
	}, nil
}
