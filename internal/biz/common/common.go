package common

import (
	"context"
	"net/mail"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// CommonRepo defines repository interface for common operations
type CommonRepo interface {
	// GetAdsList gets ads list by status
	GetAdsList(ctx context.Context, status int) ([]*Ads, error)
	// GetClientList gets subscribe application list
	GetClientList(ctx context.Context) ([]*SubscribeClient, error)
	// GetTosConfig gets TOS/Privacy config from system table
	GetTosConfig(ctx context.Context, key string) (string, error)
	// GetSystemConfigByCategory gets system config by category
	GetSystemConfigByCategory(ctx context.Context, category string) (map[string]string, error)
	// GetWebAdConfig gets WebAD config
	GetWebAdConfig(ctx context.Context) (bool, error)
	// GetEnabledAuthMethods gets enabled auth methods
	GetEnabledAuthMethods(ctx context.Context) ([]string, error)
	// GetStatistics gets system statistics (user count, node count, etc.)
	GetStatistics(ctx context.Context) (*Statistics, error)
	// SendEmailVerificationCode sends email verification code
	SendEmailVerificationCode(ctx context.Context, email string, verifyType int32) error
	// SendSmsVerificationCode sends SMS verification code
	SendSmsVerificationCode(ctx context.Context, telephone, telephoneArea string, verifyType int32) (code string, err error)
	// CheckVerificationCode checks verification code
	CheckVerificationCode(ctx context.Context, method, account, code string, verifyType int32) (bool, error)
}

// Ads contains ads information
type Ads struct {
	ID          int64
	Title       string
	Type        string
	Content     string
	Description string
	TargetURL   string
	StartTime   int64
	EndTime     int64
	Status      int
	CreatedAt   int64
	UpdatedAt   int64
}

// DownloadLink contains platform-specific download links
type DownloadLink struct {
	IOS     string
	Android string
	Windows string
	Mac     string
	Linux   string
	Harmony string
}

// SubscribeClient contains subscribe application information
type SubscribeClient struct {
	ID           int64
	Name         string
	Description  string
	Icon         string
	Scheme       string
	IsDefault    bool
	DownloadLink DownloadLink
}

// Statistics contains system statistics information
type Statistics struct {
	User     int64
	Node     int64
	Country  int64
	Protocol []string
}

// CommonUsecase handles common business logic
type CommonUsecase struct {
	repo CommonRepo
	conf *conf.Application
	log  *log.Helper
}

// NewCommonUsecase creates a new common usecase
func NewCommonUsecase(repo CommonRepo, c *conf.Application, logger log.Logger) *CommonUsecase {
	return &CommonUsecase{
		repo: repo,
		conf: c,
		log:  log.NewHelper(log.With(logger, "module", "biz/common")),
	}
}

// GetAds gets ads list
func (uc *CommonUsecase) GetAds(ctx context.Context, device, position string) ([]*Ads, error) {
	// 广告获取：当前实现获取所有活跃广告，与原项目保持一致
	adsList, err := uc.repo.GetAdsList(ctx, 1)
	if err != nil {
		uc.log.Errorw("GetAdsList error", "error", err)
		return nil, err
	}

	return adsList, nil
}

// GetClient gets subscribe client list
func (uc *CommonUsecase) GetClient(ctx context.Context) ([]*SubscribeClient, int32, error) {
	clientList, err := uc.repo.GetClientList(ctx)
	if err != nil {
		uc.log.Errorw("GetClientList error", "error", err)
		return nil, 0, err
	}

	return clientList, int32(len(clientList)), nil
}

// GetPrivacyPolicy gets privacy policy content
func (uc *CommonUsecase) GetPrivacyPolicy(ctx context.Context) (string, error) {
	content, err := uc.repo.GetTosConfig(ctx, "PrivacyPolicy")
	if err != nil {
		uc.log.Errorw("GetPrivacyPolicy error", "error", err)
		return "", err
	}

	return content, nil
}

// GetTos gets terms of service content
func (uc *CommonUsecase) GetTos(ctx context.Context) (string, error) {
	content, err := uc.repo.GetTosConfig(ctx, "TosContent")
	if err != nil {
		uc.log.Errorw("GetTos error", "error", err)
		return "", err
	}

	return content, nil
}

// GlobalConfig represents complete global configuration
type GlobalConfig struct {
	Site         *conf.Site
	Verify       *VerifyConfig
	Auth         *AuthConfig
	Invite       *conf.Invite
	Currency     map[string]string
	Subscribe    *conf.Subscribe
	Tawk         *TawkConfig
	VerifyCode   map[string]string
	OAuthMethods []string
	WebAd        bool
}

// AuthConfig combines auth-related configurations
type AuthConfig struct {
	Mobile   *conf.MobileAuth
	Email    *conf.EmailAuth
	Device   *DeviceAuthConfig
	Register *conf.Register
}

type VerifyConfig struct {
	CaptchaType                    string
	TurnstileSiteKey               string
	EnableUserLoginVerify          bool
	EnableUserRegisterVerify       bool
	EnableAdminLoginCaptcha        bool
	EnableUserResetPasswordCaptcha bool
}

type DeviceAuthConfig struct {
	Enable         bool
	ShowAds        bool
	EnableSecurity bool
	OnlyRealDevice bool
}

type TawkConfig struct {
	Enabled      bool
	PropertyID   string
	WidgetID     string
	IdentifyUser bool
	SecureMode   bool
}

// GetGlobalConfig gets global configuration
func (uc *CommonUsecase) GetGlobalConfig(ctx context.Context) (*GlobalConfig, error) {
	// Query currency config from database
	currency, err := uc.repo.GetSystemConfigByCategory(ctx, "currency")
	if err != nil {
		uc.log.Errorw("GetSystemConfigByCategory currency error", "error", err)
		// Use empty map if not found
		currency = make(map[string]string)
	}

	// Query verify code config from database
	verifyCode, err := uc.repo.GetSystemConfigByCategory(ctx, "verify_code")
	if err != nil {
		uc.log.Errorw("GetSystemConfigByCategory verify_code error", "error", err)
		// Use empty map if not found
		verifyCode = make(map[string]string)
	}

	verifyConfigMap, err := uc.repo.GetSystemConfigByCategory(ctx, "verify")
	if err != nil {
		uc.log.Errorw("GetSystemConfigByCategory verify error", "error", err)
		verifyConfigMap = make(map[string]string)
	}

	tawkConfigMap, err := uc.repo.GetSystemConfigByCategory(ctx, "tawk")
	if err != nil {
		uc.log.Errorw("GetSystemConfigByCategory tawk error", "error", err)
		tawkConfigMap = make(map[string]string)
	}

	inviteConfigMap, err := uc.repo.GetSystemConfigByCategory(ctx, "invite")
	if err != nil {
		uc.log.Errorw("GetSystemConfigByCategory invite error", "error", err)
		inviteConfigMap = make(map[string]string)
	}

	// Get enabled auth methods from database
	oauthMethods, err := uc.repo.GetEnabledAuthMethods(ctx)
	if err != nil {
		uc.log.Errorw("GetEnabledAuthMethods error", "error", err)
		// Not critical, continue with empty list
		oauthMethods = []string{}
	}

	// Get WebAD config from database
	webAd, err := uc.repo.GetWebAdConfig(ctx)
	if err != nil {
		uc.log.Errorw("GetWebAdConfig error", "error", err)
		webAd = false
	}

	// Build auth config with nil checks
	authConfig := &AuthConfig{}
	if uc.conf != nil {
		if uc.conf.Mobile != nil {
			authConfig.Mobile = uc.conf.Mobile
		}
		if uc.conf.Email != nil {
			authConfig.Email = uc.conf.Email
		}
	}
	authConfig.Device = buildDeviceAuthConfig(ctx, uc.repo)
	authConfig.Register = buildRegisterConfig(uc.conf, nil)

	// Old /site/config behavior relies on the runtime config snapshot for
	// site/register/subscribe, while invite/verify/currency/verify_code are
	// layered from database values so user-facing settings refresh immediately.
	return &GlobalConfig{
		Site:         buildSiteConfig(uc.conf, nil),
		Verify:       buildVerifyConfig(uc.conf, verifyConfigMap),
		Auth:         authConfig,
		Invite:       buildInviteConfig(uc.conf, inviteConfigMap),
		Currency:     currency,
		Subscribe:    buildSubscribeConfig(uc.conf, nil),
		Tawk:         buildTawkConfig(tawkConfigMap),
		VerifyCode:   verifyCode,
		OAuthMethods: oauthMethods,
		WebAd:        webAd,
	}, nil
}

// Helper functions to safely get config with defaults

func getSiteConfig(conf *conf.Application) *conf.Site {
	if conf == nil {
		return nil
	}
	return conf.Site
}

func getVerifyConfig(conf *conf.Application) *conf.Verify {
	if conf == nil {
		return nil
	}
	return conf.Verify
}

func getSubscribeConfig(conf *conf.Application) *conf.Subscribe {
	if conf == nil {
		return nil
	}
	return conf.Subscribe
}

func buildSiteConfig(app *conf.Application, values map[string]string) *conf.Site {
	var result conf.Site
	if app != nil && app.Site != nil {
		result = *app.Site
	}
	if value := stringFromMap(values, "Host", "host"); value != "" {
		result.Host = value
	}
	if value := stringFromMap(values, "SiteName", "site_name"); value != "" {
		result.SiteName = value
	}
	if value := stringFromMap(values, "SiteDesc", "site_desc"); value != "" {
		result.SiteDesc = value
	}
	if value := stringFromMap(values, "SiteLogo", "site_logo"); value != "" {
		result.SiteLogo = value
	}
	if value := stringFromMap(values, "Keywords", "keywords"); value != "" {
		result.Keywords = value
	}
	if value := stringFromMap(values, "CustomHTML", "custom_html"); value != "" {
		result.CustomHtml = value
	}
	if value := stringFromMap(values, "CustomData", "custom_data"); value != "" {
		result.CustomData = value
	}
	return &result
}

func buildSubscribeConfig(app *conf.Application, values map[string]string) *conf.Subscribe {
	var result conf.Subscribe
	if app != nil && app.Subscribe != nil {
		result = *app.Subscribe
	}
	result.SingleModel = boolFromMap(values, result.SingleModel, "SingleModel", "single_model")
	if value := stringFromMap(values, "SubscribePath", "subscribe_path"); value != "" {
		result.SubscribePath = value
	}
	if value := stringFromMap(values, "SubscribeDomain", "subscribe_domain"); value != "" {
		result.SubscribeDomain = value
	}
	result.PanDomain = boolFromMap(values, result.PanDomain, "PanDomain", "pan_domain")
	result.UserAgentLimit = boolFromMap(values, result.UserAgentLimit, "UserAgentLimit", "user_agent_limit")
	if value := stringFromMap(values, "UserAgentList", "user_agent_list"); value != "" {
		result.UserAgentList = value
	}
	return &result
}

func buildTawkConfig(values map[string]string) *TawkConfig {
	return &TawkConfig{
		Enabled:      boolFromMap(values, false, "Enabled", "enabled", "TawkEnabled", "tawk_enabled"),
		PropertyID:   stringFromMap(values, "PropertyID", "property_id", "TawkPropertyID", "tawk_property_id"),
		WidgetID:     stringFromMap(values, "WidgetID", "widget_id", "TawkWidgetID", "tawk_widget_id"),
		IdentifyUser: boolFromMap(values, false, "IdentifyUser", "identify_user", "TawkIdentifyUser", "tawk_identify_user"),
		SecureMode:   boolFromMap(values, false, "SecureMode", "secure_mode", "TawkSecureMode", "tawk_secure_mode"),
	}
}

// GetStat gets system statistics
func (uc *CommonUsecase) GetStat(ctx context.Context) (*Statistics, error) {
	stat, err := uc.repo.GetStatistics(ctx)
	if err != nil {
		uc.log.Errorw("GetStatistics error", "error", err)
		return nil, err
	}

	return stat, nil
}

func buildVerifyConfig(app *conf.Application, values map[string]string) *VerifyConfig {
	var result VerifyConfig
	if app != nil && app.Verify != nil {
		result.TurnstileSiteKey = app.Verify.TurnstileSiteKey
		result.EnableUserLoginVerify = app.Verify.EnableLoginVerify
		result.EnableUserRegisterVerify = app.Verify.EnableRegisterVerify
		result.EnableUserResetPasswordCaptcha = app.Verify.EnableResetPasswordVerify
	}
	if value := stringFromMap(values, "CaptchaType", "captcha_type"); value != "" {
		result.CaptchaType = value
	}
	if value := stringFromMap(values, "TurnstileSiteKey", "turnstile_site_key"); value != "" {
		result.TurnstileSiteKey = value
	}
	result.EnableUserLoginVerify = boolFromMap(values, result.EnableUserLoginVerify, "EnableUserLoginCaptcha", "enable_user_login_captcha", "EnableLoginVerify", "enable_login_verify")
	result.EnableUserRegisterVerify = boolFromMap(values, result.EnableUserRegisterVerify, "EnableUserRegisterCaptcha", "enable_user_register_captcha", "EnableRegisterVerify", "enable_register_verify")
	result.EnableAdminLoginCaptcha = boolFromMap(values, result.EnableAdminLoginCaptcha, "EnableAdminLoginCaptcha", "enable_admin_login_captcha")
	result.EnableUserResetPasswordCaptcha = boolFromMap(values, result.EnableUserResetPasswordCaptcha, "EnableUserResetPasswordCaptcha", "enable_user_reset_password_captcha", "EnableResetPasswordVerify", "enable_reset_password_verify")
	return &result
}

func buildDeviceAuthConfig(ctx context.Context, repo CommonRepo) *DeviceAuthConfig {
	methods, err := repo.GetEnabledAuthMethods(ctx)
	if err != nil {
		return &DeviceAuthConfig{}
	}

	result := &DeviceAuthConfig{}
	for _, method := range methods {
		if method == "device" {
			result.Enable = true
			break
		}
	}
	return result
}

func buildRegisterConfig(app *conf.Application, values map[string]string) *conf.Register {
	var result conf.Register
	if app != nil && app.Register != nil {
		result = *app.Register
	}
	result.StopRegister = boolFromMap(values, result.StopRegister, "StopRegister", "stop_register")
	result.EnableIpRegisterLimit = boolFromMap(values, result.EnableIpRegisterLimit, "EnableIpRegisterLimit", "enable_ip_register_limit")
	result.IpRegisterLimit = int64FromMap(values, result.IpRegisterLimit, "IpRegisterLimit", "ip_register_limit")
	result.IpRegisterLimitDuration = int64FromMap(values, result.IpRegisterLimitDuration, "IpRegisterLimitDuration", "ip_register_limit_duration")
	result.EnableTrial = boolFromMap(values, result.EnableTrial, "EnableTrial", "enable_trial")
	result.TrialSubscribe = int64FromMap(values, result.TrialSubscribe, "TrialSubscribe", "trial_subscribe")
	result.TrialTime = int64FromMap(values, result.TrialTime, "TrialTime", "trial_time")
	if value := stringFromMap(values, "TrialTimeUnit", "trial_time_unit"); value != "" {
		result.TrialTimeUnit = value
	}
	return &result
}

func buildInviteConfig(app *conf.Application, values map[string]string) *conf.Invite {
	var result conf.Invite
	if app != nil && app.Invite != nil {
		result = *app.Invite
	}
	result.ForcedInvite = boolFromMap(values, result.ForcedInvite, "ForcedInvite", "forced_invite")
	result.OnlyFirstPurchase = boolFromMap(values, result.OnlyFirstPurchase, "OnlyFirstPurchase", "only_first_purchase")
	result.ReferralPercentage = int64FromMap(values, result.ReferralPercentage, "ReferralPercentage", "referral_percentage")
	result.WithdrawalMinAmount = int64FromMap(values, result.WithdrawalMinAmount, "WithdrawalMinAmount", "withdrawal_min_amount")
	if value := stringFromMap(values, "WithdrawalMethods", "withdrawal_methods", "WithdrawalMethod"); value != "" {
		result.WithdrawalMethods = value
	}
	return &result
}

func stringFromMap(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != "" {
			return value
		}
	}
	return ""
}

func boolFromMap(values map[string]string, fallback bool, keys ...string) bool {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if parsed, err := strconv.ParseBool(value); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func int64FromMap(values map[string]string, fallback int64, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

// SendEmailCode sends email verification code
func (uc *CommonUsecase) SendEmailCode(ctx context.Context, email string, verifyType int32) error {
	if strings.TrimSpace(email) == "" {
		uc.log.Warnw("SendEmailCode invalid email", "email", email)
		return responsecode.NewKratosError(responsecode.ErrInvalidEmail)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		uc.log.Warnw("SendEmailCode invalid email", "error", err, "email", email)
		return responsecode.NewKratosError(responsecode.ErrInvalidEmail)
	}

	if err := uc.repo.SendEmailVerificationCode(ctx, email, verifyType); err != nil {
		uc.log.Errorw("SendEmailVerificationCode error", "error", err, "email", email)
		return err
	}

	return nil
}

// SendSmsCode sends SMS verification code
func (uc *CommonUsecase) SendSmsCode(ctx context.Context, telephone, telephoneArea string, verifyType int32) (string, error) {
	code, err := uc.repo.SendSmsVerificationCode(ctx, telephone, telephoneArea, verifyType)
	if err != nil {
		uc.log.Errorw("SendSmsVerificationCode error", "error", err, "telephone", telephone)
		return "", err
	}

	return code, nil
}

// CheckVerificationCode checks verification code
func (uc *CommonUsecase) CheckVerificationCode(ctx context.Context, method, account, code string, verifyType int32) (bool, error) {
	valid, err := uc.repo.CheckVerificationCode(ctx, method, account, code, verifyType)
	if err != nil {
		uc.log.Errorw("CheckVerificationCode error", "error", err, "method", method, "account", account)
		return false, err
	}

	return valid, nil
}
