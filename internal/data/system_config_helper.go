package data

import (
	"context"
	"strconv"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
)

func loadSystemConfigMap(ctx context.Context, client *ent.Client, category string) (map[string]string, error) {
	entries, err := client.ProxySystem.Query().
		Where(proxysystem.CategoryEQ(category)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		setCanonicalSystemConfigValue(values, entry.Key, entry.Value)
	}

	return values, nil
}

func normalizeSystemConfigKey(key string) string {
	switch key {
	case "site_name":
		return "SiteName"
	case "site_desc":
		return "SiteDesc"
	case "site_logo":
		return "SiteLogo"
	case "custom_html":
		return "CustomHTML"
	case "custom_data":
		return "CustomData"
	case "privacy_policy":
		return "PrivacyPolicy"
	case "tos_content":
		return "TosContent"
	case "currency_symbol":
		return "CurrencySymbol"
	case "currency_unit", "default_currency":
		return "CurrencyUnit"
	case "access_key":
		return "AccessKey"
	case "single_model":
		return "SingleModel"
	case "subscribe_path":
		return "SubscribePath"
	case "subscribe_domain":
		return "SubscribeDomain"
	case "pan_domain":
		return "PanDomain"
	case "user_agent_limit":
		return "UserAgentLimit"
	case "user_agent_list":
		return "UserAgentList"
	case "turnstile_site_key":
		return "TurnstileSiteKey"
	case "turnstile_secret":
		return "TurnstileSecret"
	case "EnableLoginVerify", "enable_login_verify":
		return "EnableUserLoginCaptcha"
	case "EnableRegisterVerify", "enable_register_verify":
		return "EnableUserRegisterCaptcha"
	case "EnableResetPasswordVerify", "enable_reset_password_verify":
		return "EnableUserResetPasswordCaptcha"
	case "captcha_type":
		return "CaptchaType"
	case "enable_user_login_captcha":
		return "EnableUserLoginCaptcha"
	case "enable_user_register_captcha":
		return "EnableUserRegisterCaptcha"
	case "enable_admin_login_captcha":
		return "EnableAdminLoginCaptcha"
	case "enable_user_reset_password_captcha":
		return "EnableUserResetPasswordCaptcha"
	case "verify_code_expire_time":
		return "VerifyCodeExpireTime"
	case "verify_code_limit":
		return "VerifyCodeLimit"
	case "verify_code_interval":
		return "VerifyCodeInterval"
	case "stop_register":
		return "StopRegister"
	case "enable_trial":
		return "EnableTrial"
	case "trial_subscribe":
		return "TrialSubscribe"
	case "trial_time":
		return "TrialTime"
	case "trial_time_unit":
		return "TrialTimeUnit"
	case "enable_ip_register_limit":
		return "EnableIpRegisterLimit"
	case "ip_register_limit":
		return "IpRegisterLimit"
	case "ip_register_limit_duration":
		return "IpRegisterLimitDuration"
	case "device_limit":
		return "DeviceLimit"
	case "forced_invite":
		return "ForcedInvite"
	case "referral_percentage":
		return "ReferralPercentage"
	case "only_first_purchase":
		return "OnlyFirstPurchase"
	case "withdrawal_min_amount":
		return "WithdrawalMinAmount"
	case "withdrawal_methods", "WithdrawalMethod":
		return "WithdrawalMethods"
	case "node_secret":
		return "NodeSecret"
	case "node_pull_interval":
		return "NodePullInterval"
	case "node_push_interval":
		return "NodePushInterval"
	case "traffic_report_threshold":
		return "TrafficReportThreshold"
	case "ip_strategy":
		return "IPStrategy"
	case "dns":
		return "DNS"
	case "block":
		return "Block"
	case "outbound":
		return "Outbound"
	case "device_admission_enabled":
		return "DeviceAdmissionEnabled"
	case "device_count_mode":
		return "DeviceCountMode"
	case "node_multiplier", "node_multiplier_config", "NodeMultiplier":
		return "NodeMultiplierConfig"
	case "web_ad":
		return "WebAD"
	case "enabled", "tawk_enabled":
		return "Enabled"
	case "property_id", "tawk_property_id":
		return "PropertyID"
	case "widget_id", "tawk_widget_id":
		return "WidgetID"
	case "identify_user", "tawk_identify_user":
		return "IdentifyUser"
	case "secure_mode", "tawk_secure_mode":
		return "SecureMode"
	case "secret_key", "tawk_secret_key":
		return "SecretKey"
	default:
		return key
	}
}

func setCanonicalSystemConfigValue(values map[string]string, rawKey, value string) {
	canonicalKey := normalizeSystemConfigKey(rawKey)
	if _, exists := values[canonicalKey]; exists && rawKey != canonicalKey {
		return
	}
	values[canonicalKey] = value
}

func systemConfigString(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok && value != "" {
			return value
		}
		if normalized := normalizeSystemConfigKey(key); normalized != key {
			if value, ok := values[normalized]; ok && value != "" {
				return value
			}
		}
	}
	return ""
}

func systemConfigLookup(values map[string]string, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value, true
		}
		if normalized := normalizeSystemConfigKey(key); normalized != key {
			if value, ok := values[normalized]; ok {
				return value, true
			}
		}
	}
	return "", false
}

func systemConfigBool(values map[string]string, fallback bool, keys ...string) bool {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if parsed, err := strconv.ParseBool(value); err == nil {
				return parsed
			}
		}
		if normalized := normalizeSystemConfigKey(key); normalized != key {
			if value, ok := values[normalized]; ok {
				if parsed, err := strconv.ParseBool(value); err == nil {
					return parsed
				}
			}
		}
	}
	return fallback
}

func parseSystemBool(value string) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}
