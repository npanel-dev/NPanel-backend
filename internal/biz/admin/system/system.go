package system

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/internal/buildmeta"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

// CurrencyConfig 货币配置
type CurrencyConfig struct {
	AccessKey      string `json:"AccessKey"`
	CurrencyUnit   string `json:"CurrencyUnit"`
	CurrencySymbol string `json:"CurrencySymbol"`
}

// InviteConfig 邀请配置
type InviteConfig struct {
	ForcedInvite        bool   `json:"ForcedInvite"`
	ReferralPercentage  int    `json:"ReferralPercentage"`
	OnlyFirstPurchase   bool   `json:"OnlyFirstPurchase"`
	WithdrawalMinAmount int64  `json:"WithdrawalMinAmount"`
	WithdrawalMethods   string `json:"WithdrawalMethods"`
}

// NodeDNS 节点DNS配置
type NodeDNS struct {
	Proto   string   `json:"Proto"`
	Address string   `json:"Address"`
	Domains []string `json:"Domains"`
}

// NodeOutbound 节点出站配置
type NodeOutbound struct {
	Name     string   `json:"Name"`
	Protocol string   `json:"Protocol"`
	Address  string   `json:"Address"`
	Port     int      `json:"Port"`
	Password string   `json:"Password"`
	Rules    []string `json:"Rules"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	NodeSecret             string `json:"NodeSecret"`
	NodePullInterval       int    `json:"NodePullInterval"`
	NodePushInterval       int    `json:"NodePushInterval"`
	TrafficReportThreshold int    `json:"TrafficReportThreshold"`
	IPStrategy             string `json:"IPStrategy"`
	DNS                    string `json:"DNS"`      // JSON string
	Block                  string `json:"Block"`    // JSON string
	Outbound               string `json:"Outbound"` // JSON string
	DeviceAdmissionEnabled bool   `json:"DeviceAdmissionEnabled"`
	DeviceCountMode        string `json:"DeviceCountMode"`
}

// PrivacyPolicyConfig 隐私政策配置
type PrivacyPolicyConfig struct {
	PrivacyPolicy string `json:"PrivacyPolicy"`
}

// RegisterConfig 注册配置
type RegisterConfig struct {
	StopRegister            bool   `json:"StopRegister"`
	EnableTrial             bool   `json:"EnableTrial"`
	TrialSubscribe          int    `json:"TrialSubscribe"`
	TrialTime               int    `json:"TrialTime"`
	TrialTimeUnit           string `json:"TrialTimeUnit"`
	EnableIpRegisterLimit   bool   `json:"EnableIpRegisterLimit"`
	IpRegisterLimit         int    `json:"IpRegisterLimit"`
	IpRegisterLimitDuration int    `json:"IpRegisterLimitDuration"`
	DeviceLimit             int    `json:"DeviceLimit"`
}

// SiteConfig 站点配置
type SiteConfig struct {
	Host       string `json:"Host"`
	SiteName   string `json:"SiteName"`
	SiteDesc   string `json:"SiteDesc"`
	SiteLogo   string `json:"SiteLogo"`
	Keywords   string `json:"Keywords"`
	CustomHTML string `json:"CustomHTML"`
	CustomData string `json:"CustomData"`
}

// TawkConfig Tawk客服配置
type TawkConfig struct {
	Enabled      bool   `json:"Enabled"`
	PropertyID   string `json:"PropertyID"`
	WidgetID     string `json:"WidgetID"`
	IdentifyUser bool   `json:"IdentifyUser"`
	SecureMode   bool   `json:"SecureMode"`
	SecretKey    string `json:"SecretKey"`
}

// SubscribeConfig 订阅配置
type SubscribeConfig struct {
	SingleModel     bool   `json:"SingleModel"`
	SubscribePath   string `json:"SubscribePath"`
	SubscribeDomain string `json:"SubscribeDomain"`
	PanDomain       bool   `json:"PanDomain"`
	UserAgentLimit  bool   `json:"UserAgentLimit"`
	UserAgentList   string `json:"UserAgentList"`
}

// TosConfig 服务条款配置
type TosConfig struct {
	TosContent string `json:"TosContent"`
}

// VerifyCodeConfig 验证码配置
type VerifyCodeConfig struct {
	VerifyCodeExpireTime int `json:"VerifyCodeExpireTime"`
	VerifyCodeLimit      int `json:"VerifyCodeLimit"`
	VerifyCodeInterval   int `json:"VerifyCodeInterval"`
}

// VerifyConfig 验证配置
type VerifyConfig struct {
	CaptchaType                    string `json:"CaptchaType"`
	TurnstileSiteKey               string `json:"TurnstileSiteKey"`
	TurnstileSecret                string `json:"TurnstileSecret"`
	EnableUserLoginCaptcha         bool   `json:"EnableUserLoginCaptcha"`
	EnableUserRegisterCaptcha      bool   `json:"EnableUserRegisterCaptcha"`
	EnableAdminLoginCaptcha        bool   `json:"EnableAdminLoginCaptcha"`
	EnableUserResetPasswordCaptcha bool   `json:"EnableUserResetPasswordCaptcha"`
}

type SystemModule struct {
	ServiceName    string
	ServiceVersion string
	Secret         string
}

// TimePeriod 时间段倍率
type TimePeriod struct {
	StartTime  string  `json:"StartTime"`
	EndTime    string  `json:"EndTime"`
	Multiplier float32 `json:"Multiplier"`
}

// SystemRepo defines the interface for system repository
type SystemRepo interface {
	GetConfigByCategory(ctx context.Context, category string) ([]*tool.SystemConfig, error)
	UpdateConfigByCategory(ctx context.Context, category string, configs map[string]*tool.SystemConfig) error
	GetNodeMultiplier(ctx context.Context) (string, error)
	UpdateNodeMultiplier(ctx context.Context, value string) error
	ApplyTelegramBot(ctx context.Context) error
}

// SystemUsecase is the system use case
type SystemUsecase struct {
	repo SystemRepo
	log  *log.Helper
}

// NewSystemUsecase creates a new system use case
func NewSystemUsecase(repo SystemRepo, logger log.Logger) *SystemUsecase {
	return &SystemUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// GetCurrencyConfig 获取货币配置
func (uc *SystemUsecase) GetCurrencyConfig(ctx context.Context) (*CurrencyConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "currency")
	if err != nil {
		uc.log.Errorf("Failed to get currency config: %v", err)
		return nil, err
	}

	result := &CurrencyConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

func (uc *SystemUsecase) GetSystemModule(ctx context.Context) (*SystemModule, error) {
	_ = ctx

	version := buildmeta.Version()

	return &SystemModule{
		ServiceName:    "ApiService",
		ServiceVersion: strings.ReplaceAll(version, "v", ""),
		Secret:         "",
	}, nil
}

// UpdateCurrencyConfig 更新货币配置
func (uc *SystemUsecase) UpdateCurrencyConfig(ctx context.Context, config *CurrencyConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "currency", configs)
}

// GetInviteConfig 获取邀请配置
func (uc *SystemUsecase) GetInviteConfig(ctx context.Context) (*InviteConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "invite")
	if err != nil {
		uc.log.Errorf("Failed to get invite config: %v", err)
		return nil, err
	}

	result := &InviteConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateInviteConfig 更新邀请配置
func (uc *SystemUsecase) UpdateInviteConfig(ctx context.Context, config *InviteConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "invite", configs)
}

// GetNodeConfig 获取节点配置
func (uc *SystemUsecase) GetNodeConfig(ctx context.Context) (*NodeConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "server")
	if err != nil {
		uc.log.Errorf("Failed to get node config: %v", err)
		return nil, err
	}

	result := &NodeConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateNodeConfig 更新节点配置
func (uc *SystemUsecase) UpdateNodeConfig(ctx context.Context, config *NodeConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "server", configs)
}

// GetPrivacyPolicyConfig 获取隐私政策配置
func (uc *SystemUsecase) GetPrivacyPolicyConfig(ctx context.Context) (*PrivacyPolicyConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "tos")
	if err != nil {
		uc.log.Errorf("Failed to get privacy policy config: %v", err)
		return nil, err
	}

	result := &PrivacyPolicyConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdatePrivacyPolicyConfig 更新隐私政策配置
func (uc *SystemUsecase) UpdatePrivacyPolicyConfig(ctx context.Context, config *PrivacyPolicyConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "tos", configs)
}

// GetRegisterConfig 获取注册配置
func (uc *SystemUsecase) GetRegisterConfig(ctx context.Context) (*RegisterConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "register")
	if err != nil {
		uc.log.Errorf("Failed to get register config: %v", err)
		return nil, err
	}

	result := &RegisterConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateRegisterConfig 更新注册配置
func (uc *SystemUsecase) UpdateRegisterConfig(ctx context.Context, config *RegisterConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "register", configs)
}

// GetSiteConfig 获取站点配置
func (uc *SystemUsecase) GetSiteConfig(ctx context.Context) (*SiteConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "site")
	if err != nil {
		uc.log.Errorf("Failed to get site config: %v", err)
		return nil, err
	}

	result := &SiteConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateSiteConfig 更新站点配置
func (uc *SystemUsecase) UpdateSiteConfig(ctx context.Context, config *SiteConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "site", configs)
}

// GetTawkConfig 获取Tawk客服配置
func (uc *SystemUsecase) GetTawkConfig(ctx context.Context) (*TawkConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "tawk")
	if err != nil {
		uc.log.Errorf("Failed to get tawk config: %v", err)
		return nil, err
	}

	result := &TawkConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateTawkConfig 更新Tawk客服配置
func (uc *SystemUsecase) UpdateTawkConfig(ctx context.Context, config *TawkConfig) error {
	config.PropertyID = strings.TrimSpace(config.PropertyID)
	config.WidgetID = strings.TrimSpace(config.WidgetID)
	config.SecretKey = strings.TrimSpace(config.SecretKey)

	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "tawk", configs)
}

// GetSubscribeConfig 获取订阅配置
func (uc *SystemUsecase) GetSubscribeConfig(ctx context.Context) (*SubscribeConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "subscribe")
	if err != nil {
		uc.log.Errorf("Failed to get subscribe config: %v", err)
		return nil, err
	}

	result := &SubscribeConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateSubscribeConfig 更新订阅配置
func (uc *SystemUsecase) UpdateSubscribeConfig(ctx context.Context, config *SubscribeConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "subscribe", configs)
}

// GetTosConfig 获取服务条款配置
func (uc *SystemUsecase) GetTosConfig(ctx context.Context) (*TosConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "tos")
	if err != nil {
		uc.log.Errorf("Failed to get tos config: %v", err)
		return nil, err
	}

	result := &TosConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateTosConfig 更新服务条款配置
func (uc *SystemUsecase) UpdateTosConfig(ctx context.Context, config *TosConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "tos", configs)
}

// GetVerifyCodeConfig 获取验证码配置
func (uc *SystemUsecase) GetVerifyCodeConfig(ctx context.Context) (*VerifyCodeConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "verify_code")
	if err != nil {
		uc.log.Errorf("Failed to get verify code config: %v", err)
		return nil, err
	}

	result := &VerifyCodeConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateVerifyCodeConfig 更新验证码配置
func (uc *SystemUsecase) UpdateVerifyCodeConfig(ctx context.Context, config *VerifyCodeConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "verify_code", configs)
}

// GetVerifyConfig 获取验证配置
func (uc *SystemUsecase) GetVerifyConfig(ctx context.Context) (*VerifyConfig, error) {
	configs, err := uc.repo.GetConfigByCategory(ctx, "verify")
	if err != nil {
		uc.log.Errorf("Failed to get verify config: %v", err)
		return nil, err
	}

	result := &VerifyConfig{}
	tool.SystemConfigSliceReflectToStruct(configs, result)

	return result, nil
}

// UpdateVerifyConfig 更新验证配置
func (uc *SystemUsecase) UpdateVerifyConfig(ctx context.Context, config *VerifyConfig) error {
	// Use reflection to convert struct to map
	v := reflect.ValueOf(*config)
	t := v.Type()

	configs := make(map[string]*tool.SystemConfig)
	for i := 0; i < v.NumField(); i++ {
		fieldName := t.Field(i).Name
		fieldValue := tool.ConvertValueToString(v.Field(i))
		fieldType := getFieldTypeString(v.Field(i))
		configs[fieldName] = &tool.SystemConfig{
			Key:   fieldName,
			Value: fieldValue,
			Type:  fieldType,
		}
	}

	return uc.repo.UpdateConfigByCategory(ctx, "verify", configs)
}

// GetNodeMultiplier 获取节点倍率配置
func (uc *SystemUsecase) GetNodeMultiplier(ctx context.Context) ([]TimePeriod, error) {
	value, err := uc.repo.GetNodeMultiplier(ctx)
	if err != nil {
		uc.log.Errorf("Failed to get node multiplier: %v", err)
		return nil, err
	}

	var periods []TimePeriod
	if value != "" {
		if err := json.Unmarshal([]byte(value), &periods); err != nil {
			uc.log.Errorf("Failed to unmarshal node multiplier: %v", err)
			return nil, err
		}
	}

	return periods, nil
}

// SetNodeMultiplier 设置节点倍率配置
func (uc *SystemUsecase) SetNodeMultiplier(ctx context.Context, periods []TimePeriod) error {
	data, err := json.Marshal(periods)
	if err != nil {
		uc.log.Errorf("Failed to marshal node multiplier: %v", err)
		return err
	}

	return uc.repo.UpdateNodeMultiplier(ctx, string(data))
}

func (uc *SystemUsecase) ApplyTelegramBot(ctx context.Context) error {
	return uc.repo.ApplyTelegramBot(ctx)
}

func PreviewNodeMultiplier(now time.Time, periods []TimePeriod) float32 {
	for _, period := range periods {
		if timeWithinPeriod(now, period.StartTime, period.EndTime) {
			if period.Multiplier > 0 {
				return period.Multiplier
			}
			return 1.0
		}
	}
	return 1.0
}

func timeWithinPeriod(current time.Time, start, end string) bool {
	startTime, err := time.Parse("15:04.000", start)
	if err != nil {
		return false
	}
	endTime, err := time.Parse("15:04.000", end)
	if err != nil {
		return false
	}

	currentTime := time.Date(0, 1, 1, current.Hour(), current.Minute(), 0, 0, time.UTC)
	startFormatted := time.Date(0, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	endFormatted := time.Date(0, 1, 1, endTime.Hour(), endTime.Minute(), 0, 0, time.UTC)

	if startFormatted.Before(endFormatted) {
		return currentTime.After(startFormatted) && currentTime.Before(endFormatted)
	}
	return currentTime.After(startFormatted) || currentTime.Before(endFormatted)
}

// getFieldTypeString returns the type string for a reflect.Value
func getFieldTypeString(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "int"
	case reflect.Int64:
		return "int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int64"
	case reflect.Float32, reflect.Float64:
		return "float"
	default:
		return "string"
	}
}
