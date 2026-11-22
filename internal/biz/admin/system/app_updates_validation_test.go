package system

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

func updateConfigJSON(platform string, row map[string]any) string {
	data, err := json.Marshal(map[string]any{"app_updates": map[string]any{platform: row}})
	if err != nil {
		panic(err)
	}
	return string(data)
}

func validUpdateRow() map[string]any {
	return map[string]any{
		"version": "1.0.8", "download_url": "https://downloads.example.test/Bear.apk",
		"enabled": true, "force_update": false,
	}
}

func TestValidateAppUpdatesAcceptsSupportedAndLegacyRecords(t *testing.T) {
	for _, customData := range []string{
		"", "{}", `{"community":{"telegram":"https://t.me/group"},"unknown":[1,false]}`,
		`{"app_updates":null}`, `{"app_updates":{}}`,
		`{"app_updates":{"android":{"enabled":false,"version":[],"force_update":"bad","artifacts":1}}}`,
	} {
		if err := validateAppUpdates(customData); err != nil {
			t.Fatalf("unrelated/disabled configuration rejected: %v", err)
		}
	}
	for _, platform := range []string{"android", "windows", "macos"} {
		row := validUpdateRow()
		row["version"] = "1.1.0-rc.2+build.9"
		row["min_version"] = "1.0.8"
		row["build_number"] = 9
		row["sha256"] = strings.Repeat("A", 64)
		row["size"] = 100
		row["package_name"] = "com.bearru.client"
		row["release_notes"] = "First line\nSecond line"
		if platform == "android" {
			row["version_code"] = 10009
		} else {
			row["min_build_number"] = 8
		}
		if err := validateAppUpdates(updateConfigJSON(platform, row)); err != nil {
			t.Fatalf("valid %s record rejected: %v", platform, err)
		}
	}
}

func TestValidateAppUpdatesRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		edit     func(map[string]any)
		field    string
	}{
		{"enabled string", "android", func(r map[string]any) { r["enabled"] = "false" }, "enabled"},
		{"force number", "android", func(r map[string]any) { r["force_update"] = 1 }, "force_update"},
		{"version required", "android", func(r map[string]any) { delete(r, "version") }, "version"},
		{"version syntax", "android", func(r map[string]any) { r["version"] = "v1.2.3" }, "version"},
		{"url required", "android", func(r map[string]any) { delete(r, "download_url") }, "download_url"},
		{"url scheme", "windows", func(r map[string]any) { r["download_url"] = "file:///tmp/app" }, "download_url"},
		{"url port", "macos", func(r map[string]any) { r["download_url"] = "https://example.test:65536/app" }, "download_url"},
		{"url credentials", "macos", func(r map[string]any) { r["download_url"] = "https://user:secret@example.test/app" }, "download_url"},
		{"url control", "macos", func(r map[string]any) { r["download_url"] = "https://example.test/a\x00b" }, "download_url"},
		{"build negative", "windows", func(r map[string]any) { r["build_number"] = -1 }, "build_number"},
		{"integer string", "android", func(r map[string]any) { r["version_code"] = "10009" }, "version_code"},
		{"integer decimal", "android", func(r map[string]any) { r["version_code"] = json.Number("10009.0") }, "version_code"},
		{"integer overflow", "android", func(r map[string]any) { r["version_code"] = json.Number("9223372036854775808") }, "version_code"},
		{"code zero", "android", func(r map[string]any) { r["version_code"] = 0 }, "version_code"},
		{"size zero", "android", func(r map[string]any) { r["size"] = 0 }, "size"},
		{"digest invalid", "android", func(r map[string]any) { r["sha256"] = "not-a-hash" }, "sha256"},
		{"notes type", "windows", func(r map[string]any) { r["release_notes"] = 7 }, "release_notes"},
		{"minimum syntax", "windows", func(r map[string]any) { r["min_version"] = "1.0" }, "min_version"},
		{"minimum higher", "windows", func(r map[string]any) { r["min_version"] = "2.0.0" }, "min_version"},
		{"minimum build unpaired", "windows", func(r map[string]any) { r["min_build_number"] = 9 }, "min_build_number"},
		{"minimum build unavailable", "windows", func(r map[string]any) {
			r["min_version"], r["min_build_number"] = "1.0.8", 9
		}, "min_version"},
		{"minimum build higher", "macos", func(r map[string]any) {
			r["min_version"], r["min_build_number"], r["build_number"] = "1.0.8", 9, 8
		}, "min_version"},
		{"Android logical minimum", "android", func(r map[string]any) {
			r["min_version"], r["min_build_number"] = "1.0.7", 8
		}, "min_build_number"},
		{"nonflat artifacts", "android", func(r map[string]any) { r["artifacts"] = map[string]any{} }, "artifacts"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := validUpdateRow()
			test.edit(row)
			err := validateAppUpdates(updateConfigJSON(test.platform, row))
			if err == nil {
				t.Fatal("expected a validation error")
			}
			converted := errors.FromError(err)
			if converted.Code != 400 || converted.Metadata["custom_code"] != "400" || !strings.Contains(converted.Message, test.field) {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if strings.Contains(converted.Message, "secret") {
				t.Fatal("validation error leaked configured URL credentials")
			}
		})
	}
	for _, customData := range []string{"invalid JSON", "[]", "null", `{"app_updates":[]}`, `{"app_updates":{"android":null}}`} {
		if err := validateAppUpdates(customData); err == nil {
			t.Fatal("expected malformed configuration to fail")
		}
	}
}

func TestValidateAppUpdateURLs(t *testing.T) {
	for _, value := range []string{
		"https://downloads.example.test:8443/app?token=temporary",
		"http://downloads.example.test/app", "HTTPS://downloads.example.test/app",
		"https://[::1]:8443/app",
	} {
		if !validUpdateDownloadURL(value) {
			t.Fatalf("valid custom download URL rejected: %q", value)
		}
	}
}

type recordingSiteRepo struct {
	SystemRepo
	writes   int
	category string
	values   map[string]*tool.SystemConfig
}

func (r *recordingSiteRepo) UpdateConfigByCategory(_ context.Context, category string, values map[string]*tool.SystemConfig) error {
	r.writes++
	r.category, r.values = category, values
	return nil
}

func TestUpdateSiteConfigValidatesBeforeWritingAndPreservesOpaqueJSON(t *testing.T) {
	repo := &recordingSiteRepo{}
	uc := NewSystemUsecase(repo, log.NewStdLogger(io.Discard))
	if err := uc.UpdateSiteConfig(context.Background(), &SiteConfig{
		CustomData: updateConfigJSON("android", map[string]any{"version": "1.0.8"}),
	}); err == nil || repo.writes != 0 {
		t.Fatalf("invalid configuration reached repository: writes=%d error=%v", repo.writes, err)
	}
	if err := uc.UpdateSiteConfig(context.Background(), nil); err == nil || repo.writes != 0 {
		t.Fatal("nil configuration must fail before repository")
	}
	customData := ` { "website":"https://users.example.test", "community":{"telegram":"tg://join?invite=group"}, "app_updates":{"android":{"enabled":false}} } `
	if err := uc.UpdateSiteConfig(context.Background(), &SiteConfig{SiteName: "Bear", CustomData: customData}); err != nil {
		t.Fatal(err)
	}
	if repo.writes != 1 || repo.category != "site" || repo.values["CustomData"].Value != customData || repo.values["SiteName"].Value != "Bear" {
		t.Fatal("valid site fields or opaque custom JSON changed while saving")
	}
}
