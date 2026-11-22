package system

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/Masterminds/semver/v3"
	"github.com/go-kratos/kratos/v2/errors"
)

var updateSHA256Pattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// validateAppUpdates 只校验更新记录，不改写其它自定义配置或原 JSON 字符串。
func validateAppUpdates(customData string) error {
	if strings.TrimSpace(customData) == "" {
		return nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(customData), &root); err != nil || root == nil {
		return invalidUpdateConfig("custom_data", "must be a JSON object")
	}
	raw, exists := root["app_updates"]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var platforms map[string]json.RawMessage
	if err := json.Unmarshal(raw, &platforms); err != nil || platforms == nil {
		return invalidUpdateConfig("app_updates", "must be an object keyed by platform")
	}
	for platform, raw := range platforms {
		var row map[string]json.RawMessage
		field := "app_updates." + platform
		if err := json.Unmarshal(raw, &row); err != nil || row == nil {
			return invalidUpdateConfig(field, "must be an object")
		}
		if err := validatePlatformUpdate(platform, field, row); err != nil {
			return err
		}
	}
	return nil
}

func validatePlatformUpdate(platform, field string, row map[string]json.RawMessage) error {
	enabled, err := updateBool(row, "enabled", true, field)
	if err != nil || !enabled {
		return err // 关闭优先，允许清理一条包含无效遗留字段的记录。
	}
	if _, err := updateBool(row, "force_update", false, field); err != nil {
		return err
	}
	versionText, err := updateString(row, "version", true, field)
	if err != nil {
		return err
	}
	version, err := semver.StrictNewVersion(strings.TrimSpace(versionText))
	if err != nil {
		return invalidUpdateConfig(field+".version", "must be a complete semantic version")
	}
	downloadURL, err := updateString(row, "download_url", true, field)
	if err != nil {
		return err
	}
	if !validUpdateDownloadURL(downloadURL) {
		return invalidUpdateConfig(field+".download_url", "must be a valid HTTP(S) URL without credentials")
	}
	build, err := updateInteger(row, "build_number", false, field)
	if err != nil {
		return err
	}
	minimumBuild, err := updateInteger(row, "min_build_number", false, field)
	if err != nil {
		return err
	}
	minimumText, err := updateString(row, "min_version", false, field)
	if err != nil {
		return err
	}
	if minimumBuild != nil && strings.TrimSpace(minimumText) == "" {
		return invalidUpdateConfig(field+".min_build_number", "requires min_version")
	}
	if platform == "android" && minimumBuild != nil {
		return invalidUpdateConfig(field+".min_build_number", "is unsupported on Android; use min_version")
	}
	if strings.TrimSpace(minimumText) != "" {
		minimum, err := semver.StrictNewVersion(strings.TrimSpace(minimumText))
		if err != nil {
			return invalidUpdateConfig(field+".min_version", "must be a complete semantic version")
		}
		comparison := minimum.Compare(version)
		if comparison > 0 || (comparison == 0 && minimumBuild != nil && (build == nil || *minimumBuild > *build)) {
			return invalidUpdateConfig(field+".min_version", "must not exceed the target version/build")
		}
	}
	for _, name := range []string{"version_code", "size"} {
		if _, err := updateInteger(row, name, true, field); err != nil {
			return err
		}
	}
	for _, name := range []string{"package_name", "release_notes"} {
		if _, err := updateString(row, name, false, field); err != nil {
			return err
		}
	}
	digest, err := updateString(row, "sha256", false, field)
	if err != nil {
		return err
	}
	if strings.TrimSpace(digest) != "" && !updateSHA256Pattern.MatchString(strings.TrimSpace(digest)) {
		return invalidUpdateConfig(field+".sha256", "must contain 64 hexadecimal digits")
	}
	if _, exists := row["artifacts"]; exists {
		return invalidUpdateConfig(field+".artifacts", "is unsupported; configure the flat download_url record")
	}
	return nil
}

func updateBool(row map[string]json.RawMessage, key string, fallback bool, field string) (bool, error) {
	raw, exists := row[key]
	if !exists {
		return fallback, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err == nil {
		if value, ok := value.(bool); ok {
			return value, nil
		}
	}
	return false, invalidUpdateConfig(field+"."+key, "must be a boolean")
}

func updateString(row map[string]json.RawMessage, key string, required bool, field string) (string, error) {
	raw, exists := row[key]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if required {
			return "", invalidUpdateConfig(field+"."+key, "is required")
		}
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", invalidUpdateConfig(field+"."+key, "must be a string")
	}
	if required && strings.TrimSpace(value) == "" {
		return "", invalidUpdateConfig(field+"."+key, "is required")
	}
	return value, nil
}

func updateInteger(row map[string]json.RawMessage, key string, positive bool, field string) (*int64, error) {
	raw, exists := row[key]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err == nil {
		if number, ok := value.(json.Number); ok {
			if integer, err := number.Int64(); err == nil && integer >= 0 && (!positive || integer > 0) {
				return &integer, nil
			}
		}
	}
	constraint := "must be a non-negative integer"
	if positive {
		constraint = "must be a positive integer"
	}
	return nil, invalidUpdateConfig(field+"."+key, constraint)
}

func validUpdateDownloadURL(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return false
		}
	}
	value = strings.TrimSpace(value)
	if strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return false
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return false
		}
	}
	return true
}

func invalidUpdateConfig(field, constraint string) error {
	// CustomErrorEncoder 通过 custom_code 输出业务码，避免把参数错误报为 500。
	return errors.BadRequest("INVALID_APP_UPDATES", field+" "+constraint).
		WithMetadata(map[string]string{"custom_code": "400"})
}
