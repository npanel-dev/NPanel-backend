package subscription

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
)

// TemplateData 模板数据结构
type TemplateData struct {
	SiteName      string
	SubscribeName string
	OutputFormat  string
	Proxies       []map[string]interface{}
	UserInfo      UserInfo
	Params        map[string]string
}

// RenderTemplate 渲染订阅配置模板（按照原项目逻辑）
func RenderTemplate(
	templateStr string,
	outputFormat string,
	siteName string,
	subscribeName string,
	nodes []*NodeInfo,
	userSubscribe *UserSubscribe,
	userInfo UserInfo,
	params map[string]string,
) ([]byte, error) {
	// 1. 转换节点为Proxy格式
	proxies := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		node.NormalizeSimnet()
		proxyMap := structToMap(node)
		proxies = append(proxies, proxyMap)
	}

	// 2. 构建用户信息（使用传入的userInfo，其中包含订阅URL）
	if userInfo.Password == "" {
		userInfo.Password = userSubscribe.UUID
	}
	if userInfo.ExpiredAt.IsZero() && userSubscribe.ExpireTime > 0 {
		userInfo.ExpiredAt = time.UnixMilli(userSubscribe.ExpireTime)
	}

	// 3. 构建模板数据
	templateData := TemplateData{
		SiteName:      siteName,
		SubscribeName: subscribeName,
		OutputFormat:  outputFormat,
		Proxies:       proxies,
		UserInfo:      userInfo,
		Params:        params,
	}

	// 4. 解析模板
	tmpl, err := template.New("subscribe").Funcs(templateFuncMap()).Parse(templateStr)
	if err != nil {
		return nil, err
	}

	// 5. 执行模板渲染
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return nil, err
	}

	result := buf.String()

	// 6. 根据输出格式处理
	if outputFormat == "base64" {
		encoded := base64.StdEncoding.EncodeToString([]byte(result))
		return []byte(encoded), nil
	}

	return buf.Bytes(), nil
}

func templateFuncMap() template.FuncMap {
	funcs := sprig.TxtFuncMap()
	funcs["simnetHexPSK"] = simnetHexPSK
	funcs["buildOmnxtSimnetConfigs"] = buildOmnxtSimnetConfigs
	funcs["buildOmnxtProtocolLinks"] = buildOmnxtProtocolLinks
	return funcs
}

func simnetHexPSK(psk string) string {
	trimmed := strings.TrimSpace(psk)
	if trimmed == "" {
		return ""
	}
	if len(trimmed)%2 == 0 {
		if _, err := hex.DecodeString(trimmed); err == nil {
			return strings.ToLower(trimmed)
		}
	}
	return hex.EncodeToString([]byte(trimmed))
}

func buildOmnxtSimnetConfigs(proxies []map[string]interface{}, userInfo UserInfo, params map[string]string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	proxyMode := strings.TrimSpace(params["proxy_mode"])
	if proxyMode == "" {
		proxyMode = "global"
	}

	dnsServers := []string{"1.1.1.1"}
	if raw := strings.TrimSpace(params["dns_servers"]); raw != "" {
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r'
		})
		parsed := make([]string, 0, len(parts))
		for _, item := range parts {
			item = strings.TrimSpace(item)
			if item != "" {
				parsed = append(parsed, item)
			}
		}
		if len(parsed) > 0 {
			dnsServers = parsed
		}
	}

	// Per-user authentication: each user gets a unique PSK (compact UUID hex)
	// and key_id derived from their subscription ID (SID).  The Node registers
	// all per-user keys in its StaticKeyResolver so the SimNet handshake
	// authenticates the specific user via the key_id frame header field.
	// IMPORTANT: key_id MUST use SubscribeID (ProxyUserSubscribe.ID), NOT
	// UserInfo.ID (User table PK), because the Node's assembler uses
	// ServerUser.ID which is the subscribe record ID.
	userKeyID := int(userInfo.SubscribeID % (1<<31 - 1))
	if userKeyID == 0 {
		userKeyID = 1 // Avoid collision with server key_id=0
	}
	userPSK := deriveSimnetUserPSK(userInfo.Password)

	for _, proxy := range proxies {
		if mapString(proxy["Type"]) != "simnet" {
			continue
		}

		afEnabled := mapBool(proxy["SimnetAfEnabled"])
		item := map[string]interface{}{
			"tag":            mapString(proxy["Name"]),
			"server_addr":    mapString(proxy["Server"]),
			"server_port":    mapInt(proxy["Port"]),
			"protocol":       "simnet",
			"sni":            mapString(proxy["SNI"]),
			"allow_insecure": mapBool(proxy["AllowInsecure"]),
			"simnet_psk":     userPSK,
			"simnet_key_id":  userKeyID,
			// Server PSK is needed for AF path/magic/content-type derivation.
			// The Node uses credentials[0] (server PSK, key_id=0) for AF, so
			// the SDK must also use the same key material for path matching.
			"simnet_server_psk":                       mapFirstStringOrNil(proxy, "SimnetPsk", "SimnetPSK"),
			"simnet_server_key_id":                    defaultInt(mapInt(proxy["SimnetKeyID"]), 0),
			"simnet_ticket_id":                        mapStringOrNil(proxy["SimnetTicketID"]),
			"simnet_path":                             defaultString(mapString(proxy["SimnetPath"]), "/simnet/session"),
			"simnet_carrier":                          defaultString(mapString(proxy["SimnetCarrier"]), "h2"),
			"simnet_af_enabled":                       afEnabled,
			"simnet_client_max_concurrent_streams":    defaultInt(mapInt(proxy["SimnetClientMaxConcurrentStreams"]), 32),
			"simnet_client_max_streams_per_session":   defaultInt(mapInt(proxy["SimnetClientMaxStreamsPerSession"]), 512),
			"simnet_client_session_idle_timeout_secs": defaultInt(mapInt(proxy["SimnetClientSessionIdleTimeoutSecs"]), 90),
			"simnet_client_max_udp_sessions":          defaultInt(mapInt(proxy["SimnetClientMaxUDPSessions"]), 64),
			"proxy_mode":                              proxyMode,
			"dns_servers":                             dnsServers,
		}
		if afEnabled {
			item["simnet_af_path_mode"] = defaultString(mapString(proxy["SimnetAfPathMode"]), "api")
			item["simnet_af_path_prefix"] = mapStringOrNil(proxy["SimnetAfPathPrefix"])
			item["simnet_af_path_suffix"] = mapStringOrNil(proxy["SimnetAfPathSuffix"])
			item["simnet_af_magic_mode"] = defaultString(mapString(proxy["SimnetAfMagicMode"]), "derived")
			item["simnet_af_response_jitter_ms"] = defaultInt(mapInt(proxy["SimnetAfResponseJitterMs"]), 50)
			item["simnet_af_handshake_polymorphism"] = mapBool(proxy["SimnetAfHandshakePolymorphism"])
			item["simnet_af_settings_jitter"] = mapBool(proxy["SimnetAfSettingsJitter"])
			item["simnet_af_fake_header_injection"] = mapBool(proxy["SimnetAfFakeHeaderInjection"])
		}
		result = append(result, item)
	}

	return result
}

func deriveSimnetUserPSK(value string) string {
	trimmed := strings.TrimSpace(value)
	if isCanonicalUUID(trimmed) {
		return strings.ToLower(strings.ReplaceAll(trimmed, "-", ""))
	}
	if len(trimmed) == 32 && isASCIIHex(trimmed) {
		return strings.ToLower(trimmed)
	}
	return hex.EncodeToString([]byte(trimmed))
}

func isCanonicalUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for idx, ch := range value {
		switch idx {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !isASCIIHexRune(ch) {
				return false
			}
		}
	}
	return true
}

func isASCIIHex(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if !isASCIIHexRune(ch) {
			return false
		}
	}
	return true
}

func isASCIIHexRune(ch rune) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

func buildOmnxtProtocolLinks(proxies []map[string]interface{}, userInfo UserInfo, params map[string]string) []string {
	configs := buildOmnxtSimnetConfigs(proxies, userInfo, params)
	result := make([]string, 0, len(configs))

	for _, item := range configs {
		serverAddr := mapString(item["server_addr"])
		serverPort := mapInt(item["server_port"])
		tag := mapString(item["tag"])

		if serverAddr == "" || serverPort == 0 {
			continue
		}

		afEnabled := mapBool(item["simnet_af_enabled"])
		payload := map[string]interface{}{
			"protocol":                              mapString(item["protocol"]),
			"server_addr":                           serverAddr,
			"server_port":                           serverPort,
			"sni":                                   mapString(item["sni"]),
			"simnet_psk":                            mapString(item["simnet_psk"]),
			"simnet_key_id":                         mapInt(item["simnet_key_id"]),
			"simnet_server_psk":                     item["simnet_server_psk"],
			"simnet_server_key_id":                  mapInt(item["simnet_server_key_id"]),
			"simnet_ticket_id":                      item["simnet_ticket_id"],
			"simnet_path":                           item["simnet_path"],
			"simnet_carrier":                        mapString(item["simnet_carrier"]),
			"simnet_af_enabled":                     afEnabled,
			"simnet_client_max_concurrent_streams":  mapInt(item["simnet_client_max_concurrent_streams"]),
			"simnet_client_max_streams_per_session": mapInt(item["simnet_client_max_streams_per_session"]),
			"simnet_client_session_idle_timeout_secs": mapInt(item["simnet_client_session_idle_timeout_secs"]),
			"simnet_client_max_udp_sessions":          mapInt(item["simnet_client_max_udp_sessions"]),
			"proxy_mode":                              item["proxy_mode"],
			"dns_servers":                             item["dns_servers"],
		}
		if afEnabled {
			payload["simnet_af_path_mode"] = mapString(item["simnet_af_path_mode"])
			payload["simnet_af_path_prefix"] = item["simnet_af_path_prefix"]
			payload["simnet_af_path_suffix"] = item["simnet_af_path_suffix"]
			payload["simnet_af_magic_mode"] = mapString(item["simnet_af_magic_mode"])
			payload["simnet_af_response_jitter_ms"] = mapInt(item["simnet_af_response_jitter_ms"])
			payload["simnet_af_handshake_polymorphism"] = mapBool(item["simnet_af_handshake_polymorphism"])
			payload["simnet_af_settings_jitter"] = mapBool(item["simnet_af_settings_jitter"])
			payload["simnet_af_fake_header_injection"] = mapBool(item["simnet_af_fake_header_injection"])
		}

		encodedPayload := encodeProtocolPayload(payload)
		if encodedPayload == "" {
			continue
		}
		result = append(result, "simnet://"+encodedPayload+"#"+url.QueryEscape(tag))
	}

	return result
}

func findProxyByName(proxies []map[string]interface{}, name string) map[string]interface{} {
	for _, proxy := range proxies {
		if mapString(proxy["Name"]) == name {
			return proxy
		}
	}
	return nil
}

func mapFirstStringOrNil(values map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if value := mapString(values[key]); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return nil
}

func encodeProtocolPayload(payload map[string]interface{}) string {
	values := url.Values{}
	for key, value := range payload {
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(v) != "" {
				values.Set(key, v)
			}
		case bool:
			if v {
				values.Set(key, "1")
			}
		case int:
			if v != 0 {
				values.Set(key, strconv.Itoa(v))
			}
		case int32:
			if v != 0 {
				values.Set(key, strconv.FormatInt(int64(v), 10))
			}
		case int64:
			if v != 0 {
				values.Set(key, strconv.FormatInt(v, 10))
			}
		case []string:
			if len(v) > 0 {
				values.Set(key, strings.Join(v, ","))
			}
		case []interface{}:
			items := make([]string, 0, len(v))
			for _, item := range v {
				if s := mapString(item); s != "" {
					items = append(items, s)
				}
			}
			if len(items) > 0 {
				values.Set(key, strings.Join(items, ","))
			}
		default:
			if s := mapString(v); s != "" {
				values.Set(key, s)
			}
		}
	}

	if len(values) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(values.Encode()))
}

func mapString(value interface{}) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func mapBool(value interface{}) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

func mapInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func mapStringOrNil(value interface{}) interface{} {
	if s := mapString(value); s != "" {
		return s
	}
	return nil
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func defaultInt(value, fallback int) int {
	if value != 0 {
		return value
	}
	return fallback
}

// structToMap 将结构体转换为map（按照原项目逻辑）
func structToMap(obj interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			m[field.Name] = v.Field(i).Interface()
		}
	}

	return m
}
