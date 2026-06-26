package data

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserdevice"
	authbiz "github.com/npanel-dev/NPanel-backend/internal/biz/auth"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	authmodel "github.com/npanel-dev/NPanel-backend/internal/model/auth"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/captcha"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
)

type AuthCompat struct {
	data   *Data
	config *conf.Application
	log    *log.Helper
}

type GenerateCaptchaResult struct {
	ID         string `json:"id"`
	Image      string `json:"image"`
	Type       string `json:"type"`
	BlockImage string `json:"block_image,omitempty"`
}

type SliderVerifyResult struct {
	Token string `json:"token"`
}

type DeviceLoginParams struct {
	Identifier string
	ShortCode  string
	Meta       authbiz.RequestMeta
}

type AdminLoginParams struct {
	Email    string
	Password string
	Meta     authbiz.RequestMeta
}

type AdminResetPasswordParams struct {
	Email    string
	Password string
	Code     string
	Meta     authbiz.RequestMeta
}

type LegacyGlobalConfig struct {
	Site         LegacySiteConfig       `json:"site"`
	Verify       LegacyVerifyConfig     `json:"verify"`
	Auth         LegacyAuthConfig       `json:"auth"`
	Invite       LegacyInviteConfig     `json:"invite"`
	Currency     LegacyCurrencyConfig   `json:"currency"`
	Subscribe    LegacySubscribeConfig  `json:"subscribe"`
	VerifyCode   LegacyVerifyCodeConfig `json:"verify_code"`
	OAuthMethods []string               `json:"oauth_methods"`
	WebAd        bool                   `json:"web_ad"`
}

type LegacySiteConfig struct {
	Host       string `json:"host"`
	SiteName   string `json:"site_name"`
	SiteDesc   string `json:"site_desc"`
	SiteLogo   string `json:"site_logo"`
	Keywords   string `json:"keywords"`
	CustomHTML string `json:"custom_html"`
	CustomData string `json:"custom_data"`
}

type LegacyVerifyConfig struct {
	CaptchaType                    string `json:"captcha_type"`
	TurnstileSiteKey               string `json:"turnstile_site_key"`
	EnableUserLoginCaptcha         bool   `json:"enable_user_login_captcha"`
	EnableUserRegisterCaptcha      bool   `json:"enable_user_register_captcha"`
	EnableAdminLoginCaptcha        bool   `json:"enable_admin_login_captcha"`
	EnableUserResetPasswordCaptcha bool   `json:"enable_user_reset_password_captcha"`
}

type LegacyAuthConfig struct {
	Mobile   LegacyMobileAuthConfig     `json:"mobile"`
	Email    LegacyEmailAuthConfig      `json:"email"`
	Device   LegacyDeviceAuthConfig     `json:"device"`
	Register LegacyPublicRegisterConfig `json:"register"`
}

type LegacyMobileAuthConfig struct {
	Enable          bool     `json:"enable"`
	EnableWhitelist bool     `json:"enable_whitelist"`
	Whitelist       []string `json:"whitelist"`
}

type LegacyEmailAuthConfig struct {
	Enable             bool   `json:"enable"`
	EnableVerify       bool   `json:"enable_verify"`
	EnableDomainSuffix bool   `json:"enable_domain_suffix"`
	DomainSuffixList   string `json:"domain_suffix_list"`
}

type LegacyDeviceAuthConfig struct {
	Enable         bool `json:"enable"`
	ShowAds        bool `json:"show_ads"`
	EnableSecurity bool `json:"enable_security"`
	OnlyRealDevice bool `json:"only_real_device"`
}

type LegacyPublicRegisterConfig struct {
	StopRegister            bool  `json:"stop_register"`
	EnableIpRegisterLimit   bool  `json:"enable_ip_register_limit"`
	IpRegisterLimit         int64 `json:"ip_register_limit"`
	IpRegisterLimitDuration int64 `json:"ip_register_limit_duration"`
}

type LegacyInviteConfig struct {
	ForcedInvite       bool  `json:"forced_invite"`
	ReferralPercentage int64 `json:"referral_percentage"`
	OnlyFirstPurchase  bool  `json:"only_first_purchase"`
}

type LegacyCurrencyConfig struct {
	CurrencyUnit   string `json:"currency_unit"`
	CurrencySymbol string `json:"currency_symbol"`
}

type LegacySubscribeConfig struct {
	SingleModel     bool   `json:"single_model"`
	SubscribePath   string `json:"subscribe_path"`
	SubscribeDomain string `json:"subscribe_domain"`
	PanDomain       bool   `json:"pan_domain"`
	UserAgentLimit  bool   `json:"user_agent_limit"`
	UserAgentList   string `json:"user_agent_list"`
}

type LegacyVerifyCodeConfig struct {
	VerifyCodeInterval int64 `json:"verify_code_interval"`
}

type HeartbeatResult struct {
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func NewAuthCompat(data *Data, config *conf.Application, logger log.Logger) *AuthCompat {
	return &AuthCompat{
		data:   data,
		config: config,
		log:    log.NewHelper(log.With(logger, "module", "data/auth_compat")),
	}
}

func (c *AuthCompat) GenerateCaptcha(ctx context.Context) (*GenerateCaptchaResult, error) {
	repo := c.repo()
	verifyCfg, err := repo.loadVerifyConfig(ctx)
	if err != nil {
		return nil, err
	}

	result := &GenerateCaptchaResult{
		Type: verifyCfg.CaptchaType,
	}

	switch verifyCfg.CaptchaType {
	case string(captcha.CaptchaTypeLocal):
		service := captcha.NewService(captcha.Config{
			Type:        captcha.CaptchaTypeLocal,
			RedisClient: c.data.rdb,
		})
		id, image, err := service.Generate(ctx)
		if err != nil {
			return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
		}
		result.ID = id
		result.Image = image
	case string(captcha.CaptchaTypeSlider):
		service := captcha.NewSliderService(c.data.rdb)
		id, bgImage, blockImage, err := service.GenerateSlider(ctx)
		if err != nil {
			return nil, responsecode.NewKratosError(responsecode.ErrInternalError)
		}
		result.ID = id
		result.Image = bgImage
		result.BlockImage = blockImage
	case string(captcha.CaptchaTypeTurnstile):
		result.ID = verifyCfg.TurnstileSiteKey
	}

	return result, nil
}

func (c *AuthCompat) VerifySliderCaptcha(ctx context.Context, id string, x, y int, trail string) (*SliderVerifyResult, error) {
	repo := c.repo()
	verifyCfg, err := repo.loadVerifyConfig(ctx)
	if err != nil {
		return nil, err
	}
	if verifyCfg.CaptchaType != string(captcha.CaptchaTypeSlider) {
		return nil, responsecode.NewKratosError(responsecode.ErrVerifyCodeError)
	}

	service := captcha.NewSliderService(c.data.rdb)
	token, err := service.VerifySlider(ctx, id, x, y, trail)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrVerifyCodeError)
	}
	return &SliderVerifyResult{Token: token}, nil
}

func (c *AuthCompat) DeviceLogin(ctx context.Context, params *DeviceLoginParams) (*authbiz.LoginResult, error) {
	if params == nil || strings.TrimSpace(params.Identifier) == "" {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}
	if c.config == nil || c.config.Device == nil || !c.config.Device.Enable {
		return nil, kerrors.BadRequest("BAD_REQUEST", "Device login is disabled")
	}

	repo := c.repo()
	if params.Meta.Identifier == "" {
		params.Meta.Identifier = strings.TrimSpace(params.Identifier)
	}
	if params.Meta.LoginType == "" {
		params.Meta.LoginType = "device"
	}

	var (
		userInfo    *ent.ProxyUser
		loginStatus bool
	)
	defer func() {
		if userInfo != nil && userInfo.ID != 0 {
			repo.logLogin(ctx, int(userInfo.ID), "device", params.Meta.IP, params.Meta.UserAgent, loginStatus)
		}
	}()

	deviceInfo, err := c.data.db.ProxyUserDevice.Query().
		Where(proxyuserdevice.IdentifierEQ(params.Identifier)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			c.log.Errorw("DeviceLogin query device failed", "error", err, "identifier", params.Identifier)
			return nil, responsecode.NewDatabaseQueryError()
		}

		userInfo, err = c.registerDeviceUser(ctx, repo, params)
		if err != nil {
			return nil, err
		}
	} else {
		userInfo, err = c.data.db.ProxyUser.Get(ctx, deviceInfo.UserID)
		if err != nil {
			c.log.Errorw("DeviceLogin query user failed", "error", err, "identifier", params.Identifier, "user_id", deviceInfo.UserID)
			return nil, responsecode.NewDatabaseQueryError()
		}
	}

	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	repo.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err := repo.issueDeviceSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (c *AuthCompat) AdminLogin(ctx context.Context, params *AdminLoginParams) (*authbiz.LoginResult, error) {
	if params == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	repo := c.repo()
	verifyCfg, err := repo.loadVerifyConfig(ctx)
	if err != nil {
		return nil, err
	}
	if verifyCfg.EnableAdminLoginCaptcha {
		if err := captcha.VerifyCaptcha(ctx, c.data.rdb, verifyCfg.CaptchaType, verifyCfg.TurnstileSecret, captcha.VerifyInput{
			CaptchaID:   params.Meta.CaptchaID,
			CaptchaCode: params.Meta.CaptchaCode,
			CfToken:     params.Meta.CfToken,
			SliderToken: params.Meta.SliderToken,
			IP:          params.Meta.IP,
		}); err != nil {
			return nil, err
		}
	}

	var loginStatus bool
	var userInfo *ent.ProxyUser
	defer func() {
		if userInfo != nil && userInfo.ID != 0 {
			repo.logLogin(ctx, int(userInfo.ID), "email", params.Meta.IP, params.Meta.UserAgent, loginStatus)
		}
	}()

	_, userInfo, err = repo.getUserByAuth(ctx, "email", params.Email)
	if err != nil {
		return nil, err
	}
	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	if !userInfo.IsAdmin {
		return nil, responsecode.NewKratosError(responsecode.ErrPermissionDenied)
	}
	if !tool.MultiPasswordVerify(userInfo.Algo, stringPointerValue(userInfo.Salt), params.Password, userInfo.Password) {
		return nil, responsecode.NewKratosError(responsecode.ErrPasswordIncorrect)
	}

	repo.bindDeviceSafely(ctx, params.Meta, userInfo.ID)
	token, err := repo.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (c *AuthCompat) AdminResetPassword(ctx context.Context, params *AdminResetPasswordParams) (*authbiz.LoginResult, error) {
	if params == nil {
		return nil, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	repo := c.repo()
	if err := repo.checkVerificationCode(ctx, verifyCodeEmailCacheKey(verifySceneSecurity, params.Email), params.Code); err != nil {
		return nil, err
	}

	verifyCfg, err := repo.loadVerifyConfig(ctx)
	if err != nil {
		return nil, err
	}
	if verifyCfg.EnableAdminLoginCaptcha {
		if err := captcha.VerifyCaptcha(ctx, c.data.rdb, verifyCfg.CaptchaType, verifyCfg.TurnstileSecret, captcha.VerifyInput{
			CaptchaID:   params.Meta.CaptchaID,
			CaptchaCode: params.Meta.CaptchaCode,
			CfToken:     params.Meta.CfToken,
			SliderToken: params.Meta.SliderToken,
			IP:          params.Meta.IP,
		}); err != nil {
			return nil, err
		}
	}

	var loginStatus bool
	var userInfo *ent.ProxyUser
	defer func() {
		if userInfo != nil && userInfo.ID != 0 && loginStatus {
			repo.logLogin(ctx, int(userInfo.ID), "email", params.Meta.IP, params.Meta.UserAgent, true)
		}
	}()

	_, userInfo, err = repo.getUserByAuth(ctx, "email", params.Email)
	if err != nil {
		return nil, err
	}
	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	if !userInfo.IsAdmin {
		return nil, responsecode.NewKratosError(responsecode.ErrPermissionDenied)
	}

	if _, err := c.data.db.ProxyUser.UpdateOneID(userInfo.ID).
		SetPassword(tool.EncodePassWord(params.Password)).
		SetAlgo("default").
		ClearSalt().
		Save(ctx); err != nil {
		c.log.Errorw("AdminResetPassword update password failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	repo.bindDeviceSafely(ctx, params.Meta, userInfo.ID)
	token, err := repo.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (c *AuthCompat) GetLegacyGlobalConfig(ctx context.Context) (*LegacyGlobalConfig, error) {
	repo := c.repo()

	verifyValues, err := repo.getSystemConfigMap(ctx, "verify")
	if err != nil {
		return nil, err
	}
	currencyValues, err := repo.getSystemConfigMap(ctx, "currency")
	if err != nil {
		return nil, err
	}
	verifyCodeValues, err := repo.getSystemConfigMap(ctx, "verify_code")
	if err != nil {
		return nil, err
	}

	result := &LegacyGlobalConfig{
		Verify: LegacyVerifyConfig{
			CaptchaType:                    getStringConfigWithDefault(verifyValues, "", "CaptchaType", "captcha_type"),
			TurnstileSiteKey:               getStringConfigWithDefault(verifyValues, "", "TurnstileSiteKey", "turnstile_site_key"),
			EnableUserLoginCaptcha:         getBoolConfig(verifyValues, false, "EnableUserLoginCaptcha", "enable_user_login_captcha", "EnableLoginVerify", "enable_login_verify"),
			EnableUserRegisterCaptcha:      getBoolConfig(verifyValues, false, "EnableUserRegisterCaptcha", "enable_user_register_captcha", "EnableRegisterVerify", "enable_register_verify"),
			EnableAdminLoginCaptcha:        getBoolConfig(verifyValues, false, "EnableAdminLoginCaptcha", "enable_admin_login_captcha"),
			EnableUserResetPasswordCaptcha: getBoolConfig(verifyValues, false, "EnableUserResetPasswordCaptcha", "enable_user_reset_password_captcha", "EnableResetPasswordVerify", "enable_reset_password_verify"),
		},
		Auth:   LegacyAuthConfig{},
		Invite: LegacyInviteConfig{},
		Currency: LegacyCurrencyConfig{
			CurrencyUnit:   getStringConfig(currencyValues, "CurrencyUnit", "currency_unit"),
			CurrencySymbol: getStringConfig(currencyValues, "CurrencySymbol", "currency_symbol"),
		},
		VerifyCode: LegacyVerifyCodeConfig{
			VerifyCodeInterval: getInt64Config(verifyCodeValues, 0, "VerifyCodeInterval", "verify_code_interval"),
		},
	}

	if c.config != nil {
		if c.config.Site != nil {
			result.Site = LegacySiteConfig{
				Host:       c.config.Site.Host,
				SiteName:   c.config.Site.SiteName,
				SiteDesc:   c.config.Site.SiteDesc,
				SiteLogo:   c.config.Site.SiteLogo,
				Keywords:   c.config.Site.Keywords,
				CustomHTML: c.config.Site.CustomHtml,
				CustomData: c.config.Site.CustomData,
			}
		}
		if c.config.Subscribe != nil {
			result.Subscribe = LegacySubscribeConfig{
				SingleModel:     c.config.Subscribe.SingleModel,
				SubscribePath:   c.config.Subscribe.SubscribePath,
				SubscribeDomain: c.config.Subscribe.SubscribeDomain,
				PanDomain:       c.config.Subscribe.PanDomain,
				UserAgentLimit:  c.config.Subscribe.UserAgentLimit,
				UserAgentList:   c.config.Subscribe.UserAgentList,
			}
		}
		if c.config.Register != nil {
			result.Auth.Register = LegacyPublicRegisterConfig{
				StopRegister:            c.config.Register.StopRegister,
				EnableIpRegisterLimit:   c.config.Register.EnableIpRegisterLimit,
				IpRegisterLimit:         c.config.Register.IpRegisterLimit,
				IpRegisterLimitDuration: c.config.Register.IpRegisterLimitDuration,
			}
		}
		if c.config.Invite != nil {
			result.Invite = LegacyInviteConfig{
				ForcedInvite:       c.config.Invite.ForcedInvite,
				ReferralPercentage: c.config.Invite.ReferralPercentage,
				OnlyFirstPurchase:  c.config.Invite.OnlyFirstPurchase,
			}
		}
		if c.config.Mobile != nil {
			result.Auth.Mobile = LegacyMobileAuthConfig{
				Enable:          c.config.Mobile.Enable,
				EnableWhitelist: c.config.Mobile.EnableWhitelist,
				Whitelist:       append([]string(nil), c.config.Mobile.Whitelist...),
			}
		}
		if c.config.Email != nil {
			result.Auth.Email = LegacyEmailAuthConfig{
				Enable:             c.config.Email.Enable,
				EnableVerify:       c.config.Email.EnableVerify,
				EnableDomainSuffix: c.config.Email.EnableDomainSuffix,
				DomainSuffixList:   c.config.Email.DomainSuffixList,
			}
		}
	}
	if legacyGatewayModeEnabled() {
		result.Subscribe.SubscribePath = "/sub" + result.Subscribe.SubscribePath
	}

	oauthMethods, authConfig := c.loadLegacyAuthMethods(ctx, result.Auth)
	result.OAuthMethods = oauthMethods
	result.Auth = authConfig

	webAd, err := c.loadWebAd(ctx)
	if err != nil {
		return nil, err
	}
	result.WebAd = webAd

	return result, nil
}

func (c *AuthCompat) Heartbeat() *HeartbeatResult {
	return &HeartbeatResult{
		Status:    true,
		Message:   "service is alive",
		Timestamp: time.Now().Unix(),
	}
}

func (c *AuthCompat) repo() *authRepo {
	return &authRepo{
		data:   c.data,
		config: c.config,
		log:    c.log,
	}
}

func (c *AuthCompat) registerDeviceUser(ctx context.Context, repo *authRepo, params *DeviceLoginParams) (*ent.ProxyUser, error) {
	registerCfg, err := repo.loadRegisterConfig(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := repo.checkRegisterIPLimit(ctx, registerCfg, params.Meta.IP, "device", params.Identifier); err != nil {
		return nil, err
	} else if !ok {
		return nil, responsecode.NewKratosError(responsecode.ErrRegisterIPLimit)
	}

	inviteCfg, err := repo.loadInviteConfig(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := c.data.db.Tx(ctx)
	if err != nil {
		c.log.Errorw("DeviceLogin start tx failed", "error", err)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	userInfo, err := tx.ProxyUser.Create().
		SetPassword(tool.EncodePassWord(uuidx.NewUUID().String())).
		SetAlgo("default").
		SetOnlyFirstPurchase(inviteCfg.OnlyFirstPurchase).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		c.log.Errorw("DeviceLogin create user failed", "error", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	referCode := tool.GenerateReferCode(userInfo.ID)
	userInfo, err = tx.ProxyUser.UpdateOneID(userInfo.ID).
		SetReferCode(referCode).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		c.log.Errorw("DeviceLogin update refer code failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	if _, err := tx.ProxyUserAuthMethod.Create().
		SetUserID(userInfo.ID).
		SetAuthType("device").
		SetAuthIdentifier(params.Identifier).
		SetVerified(true).
		Save(ctx); err != nil {
		_ = tx.Rollback()
		c.log.Errorw("DeviceLogin create auth method failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	if _, err := tx.ProxyUserDevice.Create().
		SetUserID(userInfo.ID).
		SetIP(params.Meta.IP).
		SetIdentifier(params.Identifier).
		SetShortCode(params.ShortCode).
		SetUserAgent(trimUserAgent(params.Meta.UserAgent)).
		SetEnabled(true).
		SetOnline(false).
		Save(ctx); err != nil {
		_ = tx.Rollback()
		c.log.Errorw("DeviceLogin create device failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	var trialSub *ent.ProxyUserSubscribe
	if registerCfg.EnableTrial {
		trialSub, err = repo.createTrialSubscriptionTx(ctx, tx, userInfo.ID, registerCfg)
		if err != nil {
			_ = tx.Rollback()
			c.log.Errorw("DeviceLogin create trial subscription failed", "error", err, "user_id", userInfo.ID)
			return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
		}
	}

	if err := tx.Commit(); err != nil {
		c.log.Errorw("DeviceLogin commit failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	if trialSub != nil {
		repo.clearTrialCaches(ctx, trialSub)
		repo.triggerGroupRecalculation(ctx)
	}
	repo.logRegister(ctx, int(userInfo.ID), "device", params.Identifier, params.Meta.IP, params.Meta.UserAgent)

	return userInfo, nil
}

func (c *AuthCompat) loadLegacyAuthMethods(ctx context.Context, authConfig LegacyAuthConfig) ([]string, LegacyAuthConfig) {
	methods, err := c.data.db.ProxyAuthMethod.Query().
		Order(ent.Asc(proxyauthmethod.FieldID)).
		All(ctx)
	if err != nil {
		c.log.Errorw("loadLegacyAuthMethods query failed", "error", err)
		return nil, authConfig
	}

	var enabled []string
	for _, method := range methods {
		switch method.Method {
		case "email":
			var raw authmodel.EmailAuthConfig
			raw.Unmarshal(method.Config)
			authConfig.Email.Enable = method.Enabled
			authConfig.Email.EnableVerify = raw.EnableVerify
			authConfig.Email.EnableDomainSuffix = raw.EnableDomainSuffix
			authConfig.Email.DomainSuffixList = raw.DomainSuffixList
		case "mobile":
			var raw authmodel.MobileAuthConfig
			raw.Unmarshal(method.Config)
			authConfig.Mobile.Enable = method.Enabled
			authConfig.Mobile.EnableWhitelist = raw.EnableWhitelist
			authConfig.Mobile.Whitelist = append([]string(nil), raw.Whitelist...)
		}

		if !method.Enabled {
			continue
		}
		enabled = append(enabled, method.Method)
		if method.Method == "device" {
			authConfig.Device.Enable = true
			var raw authmodel.DeviceConfig
			if err := json.Unmarshal([]byte(method.Config), &raw); err == nil {
				authConfig.Device.ShowAds = raw.ShowAds
				authConfig.Device.EnableSecurity = raw.EnableSecurity
				authConfig.Device.OnlyRealDevice = raw.OnlyRealDevice
			}
		}
	}

	return enabled, authConfig
}

func (c *AuthCompat) loadWebAd(ctx context.Context) (bool, error) {
	entry, err := c.data.db.ProxySystem.Query().
		Where(proxysystem.KeyEQ("WebAD")).
		First(ctx)
	if err != nil {
		c.log.Errorw("loadWebAd query failed", "error", err)
		return false, responsecode.NewDatabaseQueryError()
	}
	return entry.Value == "true", nil
}

func legacyGatewayModeEnabled() bool {
	value, exists := os.LookupEnv("GATEWAY_MODE")
	if !exists || strings.TrimSpace(value) != "true" {
		return false
	}

	port, exists := os.LookupEnv("GATEWAY_PORT")
	if !exists || strings.TrimSpace(port) == "" {
		return false
	}

	if _, err := strconv.Atoi(strings.TrimSpace(port)); err != nil {
		return false
	}

	return true
}
