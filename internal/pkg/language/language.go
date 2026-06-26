package language

import "strings"

const (
	ProductLanguageDefault = ""
	ProductLanguageEnglish = "en-US"
	ProductLanguageChinese = "zh-CN"
)

// NormalizeProductLanguage keeps the stored language identifiers aligned with
// the frontend i18n values while tolerating common legacy spellings.
func NormalizeProductLanguage(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ProductLanguageDefault
	}

	switch strings.ToLower(strings.ReplaceAll(trimmed, "_", "-")) {
	case "en", "en-us":
		return ProductLanguageEnglish
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn":
		return ProductLanguageChinese
	default:
		return trimmed
	}
}

func IsSupportedProductLanguage(value string) bool {
	switch NormalizeProductLanguage(value) {
	case ProductLanguageDefault, ProductLanguageEnglish, ProductLanguageChinese:
		return true
	default:
		return false
	}
}
