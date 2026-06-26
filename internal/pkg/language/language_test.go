package language

import "testing"

func TestNormalizeProductLanguage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty uses default", input: "", want: ProductLanguageDefault},
		{name: "trims default", input: "   ", want: ProductLanguageDefault},
		{name: "english short code", input: "en", want: ProductLanguageEnglish},
		{name: "english lowercase locale", input: "en-us", want: ProductLanguageEnglish},
		{name: "chinese short code", input: "zh", want: ProductLanguageChinese},
		{name: "chinese underscore locale", input: "zh_CN", want: ProductLanguageChinese},
		{name: "unknown is trimmed", input: " fr-FR ", want: "fr-FR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeProductLanguage(tt.input); got != tt.want {
				t.Fatalf("NormalizeProductLanguage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsSupportedProductLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "", want: true},
		{input: "en", want: true},
		{input: "en-US", want: true},
		{input: "zh_CN", want: true},
		{input: "fr-FR", want: false},
	}

	for _, tt := range tests {
		if got := IsSupportedProductLanguage(tt.input); got != tt.want {
			t.Fatalf("IsSupportedProductLanguage(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
