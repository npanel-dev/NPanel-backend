package server

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Protocol represents a server protocol configuration - 完全按照原项目定义
type Protocol struct {
	Type                                   string   `json:"type"`
	Port                                   int32    `json:"port"`
	Enable                                 bool     `json:"enable"`
	Security                               string   `json:"security,omitempty"`
	SNI                                    string   `json:"sni,omitempty"`
	AllowInsecure                          bool     `json:"allow_insecure,omitempty"`
	Fingerprint                            string   `json:"fingerprint,omitempty"`
	RealityServerAddr                      string   `json:"reality_server_addr,omitempty"`
	RealityServerPort                      int32    `json:"reality_server_port,omitempty"`
	RealityPrivateKey                      string   `json:"reality_private_key,omitempty"`
	RealityPublicKey                       string   `json:"reality_public_key,omitempty"`
	RealityShortId                         string   `json:"reality_short_id,omitempty"`
	Transport                              string   `json:"transport,omitempty"`
	Host                                   string   `json:"host,omitempty"`
	Path                                   string   `json:"path,omitempty"`
	ServiceName                            string   `json:"service_name,omitempty"`
	Mc1Mode                                string   `json:"mc1_mode,omitempty"`
	Mc1CidrSegments                        []string `json:"mc1_cidr_segments,omitempty"`
	MundoUsername                          string   `json:"mundo_username,omitempty"`
	MundoCertificateFingerprint            string   `json:"mundo_certificate_fingerprint,omitempty"`
	MundoFakeTitle                         string   `json:"mundo_fake_title,omitempty"`
	MundoFakeMessage                       string   `json:"mundo_fake_message,omitempty"`
	MundoAcceptProxyProtocol               bool     `json:"mundo_accept_proxy_protocol,omitempty"`
	MundoUseTLSCertificate                 bool     `json:"mundo_use_tls_certificate,omitempty"`
	ProxyProtocol                          bool     `json:"proxy_protocol,omitempty"`
	Cipher                                 string   `json:"cipher,omitempty"`
	ServerKey                              string   `json:"server_key,omitempty"`
	Flow                                   string   `json:"flow,omitempty"`
	HopPorts                               string   `json:"hop_ports,omitempty"`
	HopInterval                            int32    `json:"hop_interval,omitempty"`
	ObfsPassword                           string   `json:"obfs_password,omitempty"`
	DisableSNI                             bool     `json:"disable_sni,omitempty"`
	ReduceRtt                              bool     `json:"reduce_rtt,omitempty"`
	UDPRelayMode                           string   `json:"udp_relay_mode,omitempty"`
	CongestionController                   string   `json:"congestion_controller,omitempty"`
	Multiplex                              string   `json:"multiplex,omitempty"`
	PaddingScheme                          string   `json:"padding_scheme,omitempty"`
	UpMbps                                 int32    `json:"up_mbps,omitempty"`
	DownMbps                               int32    `json:"down_mbps,omitempty"`
	Obfs                                   string   `json:"obfs,omitempty"`
	ObfsHost                               string   `json:"obfs_host,omitempty"`
	ObfsPath                               string   `json:"obfs_path,omitempty"`
	XhttpMode                              string   `json:"xhttp_mode,omitempty"`
	XhttpExtra                             string   `json:"xhttp_extra,omitempty"`
	Encryption                             string   `json:"encryption,omitempty"`
	EncryptionMode                         string   `json:"encryption_mode,omitempty"`
	EncryptionRtt                          string   `json:"encryption_rtt,omitempty"`
	EncryptionTicket                       string   `json:"encryption_ticket,omitempty"`
	EncryptionServerPadding                string   `json:"encryption_server_padding,omitempty"`
	EncryptionPrivateKey                   string   `json:"encryption_private_key,omitempty"`
	EncryptionClientPadding                string   `json:"encryption_client_padding,omitempty"`
	EncryptionPassword                     string   `json:"encryption_password,omitempty"`
	Ratio                                  float64  `json:"ratio,omitempty"`
	CertMode                               string   `json:"cert_mode,omitempty"`
	CertDNSProvider                        string   `json:"cert_dns_provider,omitempty"`
	CertDNSEnv                             string   `json:"cert_dns_env,omitempty"`
	SimnetPsk                              string   `json:"simnet_psk,omitempty"`
	SimnetKeyID                            int32    `json:"simnet_key_id,omitempty"`
	SimnetTicketID                         string   `json:"simnet_ticket_id,omitempty"`
	SimnetPath                             string   `json:"simnet_path,omitempty"`
	SimnetCarrier                          string   `json:"simnet_carrier,omitempty"`
	SimnetAfEnabled                        bool     `json:"simnet_af_enabled,omitempty"`
	SimnetAfPathMode                       string   `json:"simnet_af_path_mode,omitempty"`
	SimnetAfPathPrefix                     string   `json:"simnet_af_path_prefix,omitempty"`
	SimnetAfPathSuffix                     string   `json:"simnet_af_path_suffix,omitempty"`
	SimnetAfMagicMode                      string   `json:"simnet_af_magic_mode,omitempty"`
	SimnetAfResponseJitterMs               int32    `json:"simnet_af_response_jitter_ms,omitempty"`
	SimnetAfHandshakePolymorphism          bool     `json:"simnet_af_handshake_polymorphism,omitempty"`
	SimnetAfSettingsJitter                 bool     `json:"simnet_af_settings_jitter,omitempty"`
	SimnetAfFakeHeaderInjection            bool     `json:"simnet_af_fake_header_injection,omitempty"`
	SimnetReverseEnabled                   bool     `json:"simnet_reverse_enabled,omitempty"`
	SimnetReverseListenAddr                string   `json:"simnet_reverse_listen_addr,omitempty"`
	SimnetReverseListenPort                int32    `json:"simnet_reverse_listen_port,omitempty"`
	SimnetReverseTargetHost                string   `json:"simnet_reverse_target_host,omitempty"`
	SimnetReverseTargetPort                int32    `json:"simnet_reverse_target_port,omitempty"`
	SimnetFallbackEnabled                  bool     `json:"simnet_fallback_enabled,omitempty"`
	SimnetFallbackTargetScheme             string   `json:"simnet_fallback_target_scheme,omitempty"`
	SimnetFallbackTargetHost               string   `json:"simnet_fallback_target_host,omitempty"`
	SimnetFallbackTargetPort               int32    `json:"simnet_fallback_target_port,omitempty"`
	SimnetFallbackHostHeader               string   `json:"simnet_fallback_host_header,omitempty"`
	SimnetFallbackTLSSNI                   string   `json:"simnet_fallback_tls_sni,omitempty"`
	SimnetInboundMaxStreamsPerSession      int32    `json:"simnet_inbound_max_streams_per_session,omitempty"`
	SimnetInboundMaxUDPStreamsPerSession   int32    `json:"simnet_inbound_max_udp_streams_per_session,omitempty"`
	SimnetInboundMaxHandlerTasksPerSession int32    `json:"simnet_inbound_max_handler_tasks_per_session,omitempty"`
	SimnetStreamEventChannelCapacity       int32    `json:"simnet_stream_event_channel_capacity,omitempty"`
	SimnetStreamDataChannelCapacity        int32    `json:"simnet_stream_data_channel_capacity,omitempty"`
	SimnetTargetDialTimeoutMs              int32    `json:"simnet_target_dial_timeout_ms,omitempty"`
	SimnetTargetMaxConcurrentDials         int32    `json:"simnet_target_max_concurrent_dials,omitempty"`
	SimnetEgressBlockLoopback              bool     `json:"simnet_egress_block_loopback,omitempty"`
	SimnetEgressBlockPrivate               bool     `json:"simnet_egress_block_private,omitempty"`
	SimnetEgressBlockLinkLocal             bool     `json:"simnet_egress_block_link_local,omitempty"`
	SimnetEgressBlockMetadata              bool     `json:"simnet_egress_block_metadata,omitempty"`
	SimnetSendWindow                       int32    `json:"simnet_send_window,omitempty"`
	SimnetRecvWindow                       int32    `json:"simnet_recv_window,omitempty"`
	SimnetMaxConcurrentStreams             int32    `json:"simnet_max_concurrent_streams,omitempty"`
	SimnetInitialWindowSize                int32    `json:"simnet_initial_window_size,omitempty"`
	SimnetMaxFrameSize                     int32    `json:"simnet_max_frame_size,omitempty"`
	SimnetClientMaxConcurrentStreams       int32    `json:"simnet_client_max_concurrent_streams,omitempty"`
	SimnetClientMaxStreamsPerSession       int32    `json:"simnet_client_max_streams_per_session,omitempty"`
	SimnetClientSessionIdleTimeoutSecs     int32    `json:"simnet_client_session_idle_timeout_secs,omitempty"`
	SimnetClientMaxUDPSessions             int32    `json:"simnet_client_max_udp_sessions,omitempty"`

	// OmniFlow 基础配置
	OmniflowCarrier     string `json:"omniflow_carrier,omitempty"`
	OmniflowPath        string `json:"omniflow_path,omitempty"`
	OmniflowContentType string `json:"omniflow_content_type,omitempty"`
	OmniflowProfilePath string `json:"omniflow_profile_path,omitempty"`
	OmniflowProfileJson string `json:"omniflow_profile_json,omitempty"`
	OmniflowServerHost  string `json:"omniflow_server_host,omitempty"`
	OmniflowServerPort  int32  `json:"omniflow_server_port,omitempty"`
	OmniflowCaCertPath  string `json:"omniflow_ca_cert_path,omitempty"`
	OmniflowTargetMeta  string `json:"omniflow_target_meta,omitempty"`
	OmniflowSpkiPin     string `json:"omniflow_spki_pin,omitempty"`

	// OmniFlow H3 Fallback 策略
	OmniflowH3FallbackEnabled          bool   `json:"omniflow_h3_fallback_enabled,omitempty"`
	OmniflowH3FallbackPolicy           string `json:"omniflow_h3_fallback_policy,omitempty"`
	OmniflowH3FallbackTimeoutMs        int32  `json:"omniflow_h3_fallback_timeout_ms,omitempty"`
	OmniflowH3FallbackRetryBudget      int32  `json:"omniflow_h3_fallback_retry_budget,omitempty"`
	OmniflowH3FallbackSmokeEnabled     bool   `json:"omniflow_h3_fallback_smoke_enabled,omitempty"`
	OmniflowH3FallbackSmokeIntervalSec int32  `json:"omniflow_h3_fallback_smoke_interval_sec,omitempty"`
	OmniflowH3FallbackSmokeTimeoutMs   int32  `json:"omniflow_h3_fallback_smoke_timeout_ms,omitempty"`

	// OmniFlow 连接管理
	OmniflowMaxAgeSec      int32 `json:"omniflow_max_age_sec,omitempty"`
	OmniflowIdleTimeoutSec int32 `json:"omniflow_idle_timeout_sec,omitempty"`
	OmniflowMaxConnections int32 `json:"omniflow_max_connections,omitempty"`

	// OmniFlow 抗指纹
	OmniflowAdaptiveTlsEnabled    bool   `json:"omniflow_adaptive_tls_enabled,omitempty"`
	OmniflowTlsFingerprint        string `json:"omniflow_tls_fingerprint,omitempty"`
	OmniflowSniMode               string `json:"omniflow_sni_mode,omitempty"`
	OmniflowPaddingMode           string `json:"omniflow_padding_mode,omitempty"`
	OmniflowTrafficShapingEnabled bool   `json:"omniflow_traffic_shaping_enabled,omitempty"`
	OmniflowAfEnabled             bool   `json:"omniflow_af_enabled,omitempty"`
	OmniflowAfPathMode            string `json:"omniflow_af_path_mode,omitempty"`
	OmniflowAfPathPrefix          string `json:"omniflow_af_path_prefix,omitempty"`
	OmniflowAfPathSuffix          string `json:"omniflow_af_path_suffix,omitempty"`
	OmniflowAfPathRotationSecs    int32  `json:"omniflow_af_path_rotation_secs,omitempty"`
	OmniflowAfPathSkewSlots       int32  `json:"omniflow_af_path_skew_slots,omitempty"`

	// OmniFlow 同端口浏览器 Fallback 反向代理
	OmniflowFallbackEnabled      bool   `json:"omniflow_fallback_enabled,omitempty"`
	OmniflowFallbackTargetScheme string `json:"omniflow_fallback_target_scheme,omitempty"`
	OmniflowFallbackTargetHost   string `json:"omniflow_fallback_target_host,omitempty"`
	OmniflowFallbackTargetPort   int32  `json:"omniflow_fallback_target_port,omitempty"`
	OmniflowFallbackHostHeader   string `json:"omniflow_fallback_host_header,omitempty"`
	OmniflowFallbackTLSSNI       string `json:"omniflow_fallback_tls_sni,omitempty"`

	// OmniFlow 回退 Carrier
	OmniflowFallbackCarrierEnabled bool `json:"omniflow_fallback_carrier_enabled,omitempty"`
	OmniflowFallbackConnectTunnel  bool `json:"omniflow_fallback_connect_tunnel,omitempty"`
	OmniflowFallbackWssEnabled     bool `json:"omniflow_fallback_wss_enabled,omitempty"`
}

const (
	defaultSimnetInboundMaxStreamsPerSession      int32 = 128
	defaultSimnetInboundMaxUDPStreamsPerSession   int32 = 64
	defaultSimnetInboundMaxHandlerTasksPerSession int32 = 128
	defaultSimnetStreamEventChannelCapacity       int32 = 256
	defaultSimnetStreamDataChannelCapacity        int32 = 128
	defaultSimnetTargetDialTimeoutMs              int32 = 12_000
	defaultSimnetTargetMaxConcurrentDials         int32 = 256
	defaultSimnetSessionWindow                    int32 = 4 * 1024 * 1024
	defaultSimnetMaxConcurrentStreams             int32 = 100
	defaultSimnetInitialWindowSize                int32 = 65_535
	defaultSimnetMaxFrameSize                     int32 = 16_384
	defaultSimnetClientMaxConcurrentStreams       int32 = 32
	defaultSimnetClientMaxStreamsPerSession       int32 = 512
	defaultSimnetClientSessionIdleTimeoutSecs     int32 = 90
	defaultSimnetClientMaxUDPSessions             int32 = 64
)

func (p *Protocol) UnmarshalJSON(data []byte) error {
	type protocolAlias Protocol
	normalized := data
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		changed := false
		for _, key := range []string{"mc1_cidr_segments", "cidrSegments", "mc1CidrSegments"} {
			values, ok := decodeMC1StringSlice(raw[key])
			if !ok {
				continue
			}
			b, err := json.Marshal(values)
			if err != nil {
				return err
			}
			raw[key] = b
			changed = true
		}
		if changed {
			b, err := json.Marshal(raw)
			if err != nil {
				return err
			}
			normalized = b
		}
	}
	aux := struct {
		*protocolAlias
		Mode                 string         `json:"mode,omitempty"`
		CidrSegments         mc1StringSlice `json:"cidrSegments,omitempty"`
		Mc1ModeCamel         string         `json:"mc1Mode,omitempty"`
		Mc1CidrSegmentsCamel mc1StringSlice `json:"mc1CidrSegments,omitempty"`
		Username             string         `json:"username,omitempty"`
		MundoUsernameCamel   string         `json:"mundoUsername,omitempty"`
		CertFingerprint      string         `json:"certificateFingerprint,omitempty"`
		MundoCertFPCamel     string         `json:"mundoCertificateFingerprint,omitempty"`
		FakeTitle            string         `json:"fakeTitle,omitempty"`
		MundoFakeTitleCamel  string         `json:"mundoFakeTitle,omitempty"`
		FakeMessage          string         `json:"fakeMessage,omitempty"`
		MundoFakeMsgCamel    string         `json:"mundoFakeMessage,omitempty"`
		AcceptProxyProtocol  bool           `json:"acceptProxyProtocol,omitempty"`
		MundoAcceptPPCamel   bool           `json:"mundoAcceptProxyProtocol,omitempty"`
		ProxyProtocolCamel   bool           `json:"proxyProtocol,omitempty"`
		UseTLSCertificate    bool           `json:"useTLSCertificate,omitempty"`
		MundoUseTLSCertCamel bool           `json:"mundoUseTLSCertificate,omitempty"`
	}{
		protocolAlias: (*protocolAlias)(p),
	}
	if err := json.Unmarshal(normalized, &aux); err != nil {
		return err
	}
	if strings.TrimSpace(p.Mc1Mode) == "" {
		if strings.TrimSpace(aux.Mc1ModeCamel) != "" {
			p.Mc1Mode = aux.Mc1ModeCamel
		} else {
			p.Mc1Mode = aux.Mode
		}
	}
	if len(p.Mc1CidrSegments) == 0 {
		if len(aux.Mc1CidrSegmentsCamel) > 0 {
			p.Mc1CidrSegments = []string(aux.Mc1CidrSegmentsCamel)
		} else {
			p.Mc1CidrSegments = []string(aux.CidrSegments)
		}
	}
	if strings.TrimSpace(p.MundoUsername) == "" {
		if strings.TrimSpace(aux.MundoUsernameCamel) != "" {
			p.MundoUsername = aux.MundoUsernameCamel
		} else {
			p.MundoUsername = aux.Username
		}
	}
	if strings.TrimSpace(p.MundoCertificateFingerprint) == "" {
		if strings.TrimSpace(aux.MundoCertFPCamel) != "" {
			p.MundoCertificateFingerprint = aux.MundoCertFPCamel
		} else {
			p.MundoCertificateFingerprint = aux.CertFingerprint
		}
	}
	if strings.TrimSpace(p.MundoFakeTitle) == "" {
		if strings.TrimSpace(aux.MundoFakeTitleCamel) != "" {
			p.MundoFakeTitle = aux.MundoFakeTitleCamel
		} else {
			p.MundoFakeTitle = aux.FakeTitle
		}
	}
	if strings.TrimSpace(p.MundoFakeMessage) == "" {
		if strings.TrimSpace(aux.MundoFakeMsgCamel) != "" {
			p.MundoFakeMessage = aux.MundoFakeMsgCamel
		} else {
			p.MundoFakeMessage = aux.FakeMessage
		}
	}
	if !p.MundoAcceptProxyProtocol {
		p.MundoAcceptProxyProtocol = aux.MundoAcceptPPCamel || aux.AcceptProxyProtocol
	}
	if !p.ProxyProtocol {
		p.ProxyProtocol = aux.ProxyProtocolCamel
	}
	if !p.MundoUseTLSCertificate {
		p.MundoUseTLSCertificate = aux.MundoUseTLSCertCamel || aux.UseTLSCertificate
	}
	return nil
}

type mc1StringSlice []string

func (s *mc1StringSlice) UnmarshalJSON(data []byte) error {
	values, ok := decodeMC1StringSlice(data)
	if !ok {
		var direct []string
		if err := json.Unmarshal(data, &direct); err != nil {
			return err
		}
		values = direct
	}
	*s = sanitizeMC1StringSlice(values)
	return nil
}

func decodeMC1StringSlice(data []byte) ([]string, bool) {
	if len(data) == 0 || string(data) == "null" {
		return nil, false
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		return splitMC1CIDRString(single), true
	}
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		return sanitizeMC1StringSlice(values), true
	}
	return nil, false
}

func sanitizeMC1StringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func splitMC1CIDRString(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return sanitizeMC1StringSlice(strings.Split(value, ","))
}

func (p *Protocol) NormalizeSimnet() {
	if p == nil || p.Type != "simnet" {
		return
	}
	if strings.TrimSpace(p.SimnetPath) == "" {
		p.SimnetPath = "/simnet/session"
	}
	p.applySimnetResourceDefaults()
	if !p.SimnetFallbackEnabled || strings.TrimSpace(p.SimnetFallbackTargetHost) == "" {
		p.SimnetFallbackEnabled = false
		p.SimnetFallbackTargetScheme = ""
		p.SimnetFallbackTargetHost = ""
		p.SimnetFallbackTargetPort = 0
		p.SimnetFallbackHostHeader = ""
		p.SimnetFallbackTLSSNI = ""
	} else {
		p.SimnetFallbackTargetHost = strings.TrimSpace(p.SimnetFallbackTargetHost)
		p.SimnetFallbackHostHeader = strings.TrimSpace(p.SimnetFallbackHostHeader)
		p.SimnetFallbackTLSSNI = strings.TrimSpace(p.SimnetFallbackTLSSNI)
		switch strings.ToLower(strings.TrimSpace(p.SimnetFallbackTargetScheme)) {
		case "http", "https":
			p.SimnetFallbackTargetScheme = strings.ToLower(strings.TrimSpace(p.SimnetFallbackTargetScheme))
		default:
			p.SimnetFallbackTargetScheme = "https"
		}
	}
	if !p.SimnetAfEnabled {
		p.SimnetAfPathMode = ""
		p.SimnetAfMagicMode = ""
		p.SimnetAfPathPrefix = ""
		p.SimnetAfPathSuffix = ""
		p.SimnetAfResponseJitterMs = 0
		p.SimnetAfHandshakePolymorphism = false
		p.SimnetAfSettingsJitter = false
		p.SimnetAfFakeHeaderInjection = false
		return
	}
	if p.SimnetAfPathMode == "" {
		p.SimnetAfPathMode = "api"
	}
	if p.SimnetAfMagicMode == "" {
		p.SimnetAfMagicMode = "derived"
	}
	if p.SimnetAfResponseJitterMs == 0 {
		p.SimnetAfResponseJitterMs = 50
	}
	if !p.SimnetAfHandshakePolymorphism {
		p.SimnetAfHandshakePolymorphism = true
	}
	if !p.SimnetAfSettingsJitter {
		p.SimnetAfSettingsJitter = true
	}
	if !p.SimnetAfFakeHeaderInjection {
		p.SimnetAfFakeHeaderInjection = true
	}
}

func (p *Protocol) applySimnetResourceDefaults() {
	if p.SimnetInboundMaxStreamsPerSession <= 0 {
		p.SimnetInboundMaxStreamsPerSession = defaultSimnetInboundMaxStreamsPerSession
	}
	if p.SimnetInboundMaxUDPStreamsPerSession <= 0 {
		p.SimnetInboundMaxUDPStreamsPerSession = defaultSimnetInboundMaxUDPStreamsPerSession
	}
	if p.SimnetInboundMaxHandlerTasksPerSession <= 0 {
		p.SimnetInboundMaxHandlerTasksPerSession = defaultSimnetInboundMaxHandlerTasksPerSession
	}
	if p.SimnetStreamEventChannelCapacity <= 0 {
		p.SimnetStreamEventChannelCapacity = defaultSimnetStreamEventChannelCapacity
	}
	if p.SimnetStreamDataChannelCapacity <= 0 {
		p.SimnetStreamDataChannelCapacity = defaultSimnetStreamDataChannelCapacity
	}
	if p.SimnetTargetDialTimeoutMs <= 0 {
		p.SimnetTargetDialTimeoutMs = defaultSimnetTargetDialTimeoutMs
	}
	if p.SimnetTargetMaxConcurrentDials <= 0 {
		p.SimnetTargetMaxConcurrentDials = defaultSimnetTargetMaxConcurrentDials
	}
	if p.SimnetSendWindow <= 0 {
		p.SimnetSendWindow = defaultSimnetSessionWindow
	}
	if p.SimnetRecvWindow <= 0 {
		p.SimnetRecvWindow = defaultSimnetSessionWindow
	}
	if p.SimnetMaxConcurrentStreams <= 0 {
		p.SimnetMaxConcurrentStreams = defaultSimnetMaxConcurrentStreams
	}
	if p.SimnetInitialWindowSize <= 0 {
		p.SimnetInitialWindowSize = defaultSimnetInitialWindowSize
	}
	if p.SimnetMaxFrameSize <= 0 {
		p.SimnetMaxFrameSize = defaultSimnetMaxFrameSize
	}
	if p.SimnetClientMaxConcurrentStreams <= 0 {
		p.SimnetClientMaxConcurrentStreams = defaultSimnetClientMaxConcurrentStreams
	}
	if p.SimnetClientMaxStreamsPerSession <= 0 {
		p.SimnetClientMaxStreamsPerSession = defaultSimnetClientMaxStreamsPerSession
	}
	if p.SimnetClientSessionIdleTimeoutSecs <= 0 {
		p.SimnetClientSessionIdleTimeoutSecs = defaultSimnetClientSessionIdleTimeoutSecs
	}
	if p.SimnetClientMaxUDPSessions <= 0 {
		p.SimnetClientMaxUDPSessions = defaultSimnetClientMaxUDPSessions
	}
}

func (p *Protocol) NormalizeOmniflow() {
	if p == nil || (p.Type != "omniflow" && p.Type != "omniflow-h3") {
		return
	}
	if !p.OmniflowFallbackEnabled || strings.TrimSpace(p.OmniflowFallbackTargetHost) == "" {
		p.OmniflowFallbackEnabled = false
		p.OmniflowFallbackTargetScheme = ""
		p.OmniflowFallbackTargetHost = ""
		p.OmniflowFallbackTargetPort = 0
		p.OmniflowFallbackHostHeader = ""
		p.OmniflowFallbackTLSSNI = ""
	} else {
		p.OmniflowFallbackTargetHost = strings.TrimSpace(p.OmniflowFallbackTargetHost)
		p.OmniflowFallbackHostHeader = strings.TrimSpace(p.OmniflowFallbackHostHeader)
		p.OmniflowFallbackTLSSNI = strings.TrimSpace(p.OmniflowFallbackTLSSNI)
		switch strings.ToLower(strings.TrimSpace(p.OmniflowFallbackTargetScheme)) {
		case "http", "https":
			p.OmniflowFallbackTargetScheme = strings.ToLower(strings.TrimSpace(p.OmniflowFallbackTargetScheme))
		default:
			p.OmniflowFallbackTargetScheme = "https"
		}
	}
	if !p.OmniflowAfEnabled {
		p.OmniflowAfPathMode = ""
		p.OmniflowAfPathPrefix = ""
		p.OmniflowAfPathSuffix = ""
		p.OmniflowAfPathRotationSecs = 0
		p.OmniflowAfPathSkewSlots = 0
		return
	}
	if strings.TrimSpace(p.OmniflowAfPathMode) == "" {
		p.OmniflowAfPathMode = "random"
	} else {
		p.OmniflowAfPathMode = strings.ToLower(strings.TrimSpace(p.OmniflowAfPathMode))
	}
	p.OmniflowAfPathPrefix = strings.TrimSpace(p.OmniflowAfPathPrefix)
	p.OmniflowAfPathSuffix = strings.TrimSpace(p.OmniflowAfPathSuffix)
	if p.OmniflowAfPathRotationSecs <= 0 {
		p.OmniflowAfPathRotationSecs = 300
	}
	if p.OmniflowAfPathSkewSlots <= 0 {
		p.OmniflowAfPathSkewSlots = 1
	}
}

// MarshalProtocols converts protocol array to JSON string
func MarshalProtocols(protocols []*Protocol) (string, error) {
	if len(protocols) == 0 {
		return "", nil
	}
	data, err := json.Marshal(protocols)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalProtocols converts JSON string to protocol array
func UnmarshalProtocols(protocolsJSON string) ([]*Protocol, error) {
	var protocols []*Protocol
	if protocolsJSON == "" {
		return protocols, nil
	}
	err := json.Unmarshal([]byte(protocolsJSON), &protocols)
	if err != nil {
		return nil, err
	}
	return protocols, nil
}

// ValidateProtocols validates protocol list
func ValidateProtocols(protocols []*Protocol) error {
	// A protocol entry represents one listener instance, so uniqueness is
	// scoped to type+port instead of type only.
	seen := make(map[string]bool)
	enabledPorts := make(map[int32]string)
	for _, p := range protocols {
		if p == nil || strings.TrimSpace(p.Type) == "" {
			return ErrProtocolTypeRequired
		}
		key := protocolUniqueKey(p)
		if seen[key] {
			return ErrDuplicateProtocolType
		}
		seen[key] = true
		if p.Enable && p.Port > 0 {
			if previous, ok := enabledPorts[p.Port]; ok && previous != "" {
				return ErrDuplicateProtocolPort
			}
			enabledPorts[p.Port] = strings.ToLower(strings.TrimSpace(p.Type))
		}
	}
	return nil
}

func protocolUniqueKey(p *Protocol) string {
	protocolType := strings.ToLower(strings.TrimSpace(p.Type))
	return fmt.Sprintf("%s:%d", protocolType, p.Port)
}
