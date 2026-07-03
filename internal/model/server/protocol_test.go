package server

import (
	"errors"
	"testing"
)

func TestUnmarshalProtocolsMxMc1Aliases(t *testing.T) {
	for _, tc := range []struct {
		name string
		json string
	}{
		{
			name: "array alias",
			json: `[{"type":"mx","port":443,"enable":true,"transport":"mc1","mode":"auto","cidrSegments":["127.0.0.0/24","10.0.0.0/8"]}]`,
		},
		{
			name: "comma string alias",
			json: `[{"type":"mx","port":443,"enable":true,"transport":"mc1","mode":"auto","cidrSegments":"127.0.0.0/24, 10.0.0.0/8"}]`,
		},
		{
			name: "snake string",
			json: `[{"type":"mx","port":443,"enable":true,"transport":"mc1","mc1_mode":"auto","mc1_cidr_segments":"127.0.0.0/24, 10.0.0.0/8"}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			protocols, err := UnmarshalProtocols(tc.json)
			if err != nil {
				t.Fatalf("UnmarshalProtocols returned error: %v", err)
			}
			if len(protocols) != 1 {
				t.Fatalf("protocol count = %d, want 1", len(protocols))
			}
			if got := protocols[0].Mc1Mode; got != "auto" {
				t.Fatalf("Mc1Mode = %q, want auto", got)
			}
			if got := protocols[0].Mc1CidrSegments; len(got) != 2 || got[0] != "127.0.0.0/24" || got[1] != "10.0.0.0/8" {
				t.Fatalf("Mc1CidrSegments = %#v, want aliases preserved", got)
			}
		})
	}
}

func TestUnmarshalProtocolsMundoAliases(t *testing.T) {
	protocols, err := UnmarshalProtocols(`[{"type":"mx","port":443,"enable":true,"transport":"mundordp","username":"alice","certificateFingerprint":"fp","fakeTitle":"Login","fakeMessage":"Denied","acceptProxyProtocol":true,"useTLSCertificate":true}]`)
	if err != nil {
		t.Fatalf("UnmarshalProtocols returned error: %v", err)
	}
	if len(protocols) != 1 {
		t.Fatalf("protocol count = %d, want 1", len(protocols))
	}
	protocol := protocols[0]
	if protocol.MundoUsername != "alice" {
		t.Fatalf("MundoUsername = %q, want alice", protocol.MundoUsername)
	}
	if protocol.MundoCertificateFingerprint != "fp" {
		t.Fatalf("MundoCertificateFingerprint = %q, want fp", protocol.MundoCertificateFingerprint)
	}
	if protocol.MundoFakeTitle != "Login" || protocol.MundoFakeMessage != "Denied" {
		t.Fatalf("mundo fake fields = %q/%q, want Login/Denied", protocol.MundoFakeTitle, protocol.MundoFakeMessage)
	}
	if !protocol.MundoAcceptProxyProtocol || !protocol.MundoUseTLSCertificate {
		t.Fatalf("mundo bool fields were not preserved: %+v", protocol)
	}
}

func TestUnmarshalProtocolsProxyProtocolAliases(t *testing.T) {
	for _, tc := range []struct {
		name string
		json string
	}{
		{
			name: "snake",
			json: `[{"type":"vless","port":443,"enable":true,"transport":"tcp","proxy_protocol":true}]`,
		},
		{
			name: "camel",
			json: `[{"type":"vless","port":443,"enable":true,"transport":"tcp","proxyProtocol":true}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			protocols, err := UnmarshalProtocols(tc.json)
			if err != nil {
				t.Fatalf("UnmarshalProtocols returned error: %v", err)
			}
			if len(protocols) != 1 {
				t.Fatalf("protocol count = %d, want 1", len(protocols))
			}
			if !protocols[0].ProxyProtocol {
				t.Fatalf("ProxyProtocol = false, want true: %+v", protocols[0])
			}
			if protocols[0].MundoAcceptProxyProtocol {
				t.Fatalf("ProxyProtocol should not set MundoAcceptProxyProtocol: %+v", protocols[0])
			}
		})
	}
}

func TestUnmarshalProtocolsAcceptProxyProtocolRemainsMundoAlias(t *testing.T) {
	protocols, err := UnmarshalProtocols(`[{"type":"vless","port":443,"enable":true,"transport":"tcp","acceptProxyProtocol":true}]`)
	if err != nil {
		t.Fatalf("UnmarshalProtocols returned error: %v", err)
	}
	if len(protocols) != 1 {
		t.Fatalf("protocol count = %d, want 1", len(protocols))
	}
	if protocols[0].ProxyProtocol {
		t.Fatalf("acceptProxyProtocol should not set generic ProxyProtocol: %+v", protocols[0])
	}
	if !protocols[0].MundoAcceptProxyProtocol {
		t.Fatalf("existing Mundo acceptProxyProtocol alias was not preserved: %+v", protocols[0])
	}
}

func TestValidateProtocolsAllowsSameTypeDifferentPorts(t *testing.T) {
	err := ValidateProtocols([]*Protocol{
		{Type: "mx", Port: 443, Transport: "mc1", Enable: true},
		{Type: "mx", Port: 8443, Transport: "mundordp", Enable: true},
		{Type: "mx", Port: 9443, Transport: "mundosql", Enable: true},
		{Type: "vless", Port: 10443, Transport: "mc1", Enable: true},
	})
	if err != nil {
		t.Fatalf("ValidateProtocols returned error: %v", err)
	}
}

func TestValidateProtocolsRejectsDuplicateTypeAndPort(t *testing.T) {
	err := ValidateProtocols([]*Protocol{
		{Type: "mx", Port: 443, Transport: "mc1"},
		{Type: "mx", Port: 443, Transport: "mundordp"},
	})
	if !errors.Is(err, ErrDuplicateProtocolType) {
		t.Fatalf("ValidateProtocols error = %v, want ErrDuplicateProtocolType", err)
	}
}

func TestValidateProtocolsRejectsEnabledPortCollision(t *testing.T) {
	err := ValidateProtocols([]*Protocol{
		{Type: "vless", Port: 443, Transport: "mc1", Enable: true},
		{Type: "shadowsocks", Port: 443, Transport: "tcp", Enable: true},
	})
	if !errors.Is(err, ErrDuplicateProtocolPort) {
		t.Fatalf("ValidateProtocols error = %v, want ErrDuplicateProtocolPort", err)
	}
}

func TestValidateProtocolsAllowsDisabledPortCollision(t *testing.T) {
	err := ValidateProtocols([]*Protocol{
		{Type: "vless", Port: 443, Transport: "mc1", Enable: true},
		{Type: "shadowsocks", Port: 443, Transport: "tcp", Enable: false},
	})
	if err != nil {
		t.Fatalf("ValidateProtocols returned error: %v", err)
	}
}

func TestProtocolNormalizeSimnetClearsDisabledFallback(t *testing.T) {
	protocol := &Protocol{
		Type:                          "simnet",
		SimnetFallbackEnabled:         false,
		SimnetFallbackTargetScheme:    "https",
		SimnetFallbackTargetHost:      "www.example.com",
		SimnetFallbackTargetPort:      443,
		SimnetFallbackHostHeader:      "www.example.com",
		SimnetFallbackTLSSNI:          "www.example.com",
		SimnetAfEnabled:               false,
		SimnetAfPathMode:              "random",
		SimnetAfMagicMode:             "random",
		SimnetAfResponseJitterMs:      20,
		SimnetAfHandshakePolymorphism: true,
		SimnetAfSettingsJitter:        true,
		SimnetAfFakeHeaderInjection:   true,
	}

	protocol.NormalizeSimnet()

	if protocol.SimnetFallbackEnabled {
		t.Fatal("fallback should remain disabled")
	}
	if protocol.SimnetFallbackTargetScheme != "" ||
		protocol.SimnetFallbackTargetHost != "" ||
		protocol.SimnetFallbackTargetPort != 0 ||
		protocol.SimnetFallbackHostHeader != "" ||
		protocol.SimnetFallbackTLSSNI != "" {
		t.Fatalf("disabled fallback fields were not cleared: %+v", protocol)
	}
	if protocol.SimnetPath != "/simnet/session" {
		t.Fatalf("expected default simnet path, got %q", protocol.SimnetPath)
	}
	if protocol.SimnetAfPathMode != "" || protocol.SimnetAfMagicMode != "" {
		t.Fatalf("expected disabled AF fields to be cleared, got path=%q magic=%q", protocol.SimnetAfPathMode, protocol.SimnetAfMagicMode)
	}
}

func TestProtocolNormalizeSimnetKeepsEnabledFallback(t *testing.T) {
	protocol := &Protocol{
		Type:                       "simnet",
		SimnetFallbackEnabled:      true,
		SimnetFallbackTargetScheme: " HTTPS ",
		SimnetFallbackTargetHost:   " www.example.com ",
		SimnetFallbackHostHeader:   " fallback.example.com ",
		SimnetFallbackTLSSNI:       " tls.example.com ",
	}

	protocol.NormalizeSimnet()

	if !protocol.SimnetFallbackEnabled {
		t.Fatal("fallback should stay enabled")
	}
	if protocol.SimnetFallbackTargetScheme != "https" {
		t.Fatalf("expected normalized scheme, got %q", protocol.SimnetFallbackTargetScheme)
	}
	if protocol.SimnetFallbackTargetHost != "www.example.com" {
		t.Fatalf("expected trimmed target host, got %q", protocol.SimnetFallbackTargetHost)
	}
	if protocol.SimnetFallbackHostHeader != "fallback.example.com" {
		t.Fatalf("expected trimmed host header, got %q", protocol.SimnetFallbackHostHeader)
	}
	if protocol.SimnetFallbackTLSSNI != "tls.example.com" {
		t.Fatalf("expected trimmed TLS SNI, got %q", protocol.SimnetFallbackTLSSNI)
	}
}

func TestProtocolNormalizeSimnetDefaultsEnabledAfSubFeatures(t *testing.T) {
	protocol := &Protocol{
		Type:                     "simnet",
		SimnetAfEnabled:          true,
		SimnetAfPathMode:         "random",
		SimnetAfMagicMode:        "derived",
		SimnetAfResponseJitterMs: 50,
	}

	protocol.NormalizeSimnet()

	if !protocol.SimnetAfHandshakePolymorphism {
		t.Fatal("expected enabled AF to default handshake polymorphism on")
	}
	if !protocol.SimnetAfSettingsJitter {
		t.Fatal("expected enabled AF to default settings jitter on")
	}
	if !protocol.SimnetAfFakeHeaderInjection {
		t.Fatal("expected enabled AF to default fake header injection on")
	}
}

func TestProtocolNormalizeSimnetDefaultsResourceLimits(t *testing.T) {
	protocol := &Protocol{Type: "simnet"}

	protocol.NormalizeSimnet()

	if protocol.SimnetInboundMaxStreamsPerSession != defaultSimnetInboundMaxStreamsPerSession {
		t.Fatalf("inbound max streams = %d, want %d", protocol.SimnetInboundMaxStreamsPerSession, defaultSimnetInboundMaxStreamsPerSession)
	}
	if protocol.SimnetInboundMaxUDPStreamsPerSession != defaultSimnetInboundMaxUDPStreamsPerSession {
		t.Fatalf("inbound max udp streams = %d, want %d", protocol.SimnetInboundMaxUDPStreamsPerSession, defaultSimnetInboundMaxUDPStreamsPerSession)
	}
	if protocol.SimnetInboundMaxHandlerTasksPerSession != defaultSimnetInboundMaxHandlerTasksPerSession {
		t.Fatalf("handler task limit = %d, want %d", protocol.SimnetInboundMaxHandlerTasksPerSession, defaultSimnetInboundMaxHandlerTasksPerSession)
	}
	if protocol.SimnetStreamEventChannelCapacity != defaultSimnetStreamEventChannelCapacity {
		t.Fatalf("event channel capacity = %d, want %d", protocol.SimnetStreamEventChannelCapacity, defaultSimnetStreamEventChannelCapacity)
	}
	if protocol.SimnetStreamDataChannelCapacity != defaultSimnetStreamDataChannelCapacity {
		t.Fatalf("data channel capacity = %d, want %d", protocol.SimnetStreamDataChannelCapacity, defaultSimnetStreamDataChannelCapacity)
	}
	if protocol.SimnetTargetDialTimeoutMs != defaultSimnetTargetDialTimeoutMs {
		t.Fatalf("dial timeout = %d, want %d", protocol.SimnetTargetDialTimeoutMs, defaultSimnetTargetDialTimeoutMs)
	}
	if protocol.SimnetTargetMaxConcurrentDials != defaultSimnetTargetMaxConcurrentDials {
		t.Fatalf("dial limit = %d, want %d", protocol.SimnetTargetMaxConcurrentDials, defaultSimnetTargetMaxConcurrentDials)
	}
	if protocol.SimnetSendWindow != defaultSimnetSessionWindow || protocol.SimnetRecvWindow != defaultSimnetSessionWindow {
		t.Fatalf("windows = %d/%d, want %d", protocol.SimnetSendWindow, protocol.SimnetRecvWindow, defaultSimnetSessionWindow)
	}
	if protocol.SimnetMaxConcurrentStreams != defaultSimnetMaxConcurrentStreams ||
		protocol.SimnetInitialWindowSize != defaultSimnetInitialWindowSize ||
		protocol.SimnetMaxFrameSize != defaultSimnetMaxFrameSize {
		t.Fatalf("h2 defaults = %d/%d/%d", protocol.SimnetMaxConcurrentStreams, protocol.SimnetInitialWindowSize, protocol.SimnetMaxFrameSize)
	}
	if protocol.SimnetClientMaxConcurrentStreams != defaultSimnetClientMaxConcurrentStreams ||
		protocol.SimnetClientMaxStreamsPerSession != defaultSimnetClientMaxStreamsPerSession ||
		protocol.SimnetClientSessionIdleTimeoutSecs != defaultSimnetClientSessionIdleTimeoutSecs ||
		protocol.SimnetClientMaxUDPSessions != defaultSimnetClientMaxUDPSessions {
		t.Fatalf("client defaults = %d/%d/%d/%d", protocol.SimnetClientMaxConcurrentStreams, protocol.SimnetClientMaxStreamsPerSession, protocol.SimnetClientSessionIdleTimeoutSecs, protocol.SimnetClientMaxUDPSessions)
	}
}

func TestProtocolNormalizeSimnetDoesNotPolluteOtherProtocols(t *testing.T) {
	protocol := &Protocol{Type: "vless"}

	protocol.NormalizeSimnet()

	if protocol.SimnetPath != "" {
		t.Fatalf("non-simnet path changed to %q", protocol.SimnetPath)
	}
	if protocol.SimnetInboundMaxStreamsPerSession != 0 ||
		protocol.SimnetInboundMaxUDPStreamsPerSession != 0 ||
		protocol.SimnetTargetDialTimeoutMs != 0 ||
		protocol.SimnetSendWindow != 0 ||
		protocol.SimnetClientMaxConcurrentStreams != 0 ||
		protocol.SimnetClientMaxUDPSessions != 0 {
		t.Fatalf("non-simnet resource fields were populated: %+v", protocol)
	}
}

func TestProtocolNormalizeOmniflowClearsDisabledAfPath(t *testing.T) {
	protocol := &Protocol{
		Type:                         "omniflow",
		OmniflowFallbackEnabled:      false,
		OmniflowFallbackTargetScheme: "https",
		OmniflowFallbackTargetHost:   "www.example.com",
		OmniflowFallbackTargetPort:   443,
		OmniflowFallbackHostHeader:   "www.example.com",
		OmniflowFallbackTLSSNI:       "www.example.com",
		OmniflowAfEnabled:            false,
		OmniflowAfPathMode:           "random",
		OmniflowAfPathPrefix:         "/cdn",
		OmniflowAfPathSuffix:         ".woff2",
		OmniflowAfPathRotationSecs:   120,
		OmniflowAfPathSkewSlots:      2,
	}

	protocol.NormalizeOmniflow()

	if protocol.OmniflowAfPathMode != "" ||
		protocol.OmniflowAfPathPrefix != "" ||
		protocol.OmniflowAfPathSuffix != "" ||
		protocol.OmniflowAfPathRotationSecs != 0 ||
		protocol.OmniflowAfPathSkewSlots != 0 {
		t.Fatalf("expected disabled OmniFlow AF path fields to be cleared, got %+v", protocol)
	}
	if protocol.OmniflowFallbackEnabled ||
		protocol.OmniflowFallbackTargetScheme != "" ||
		protocol.OmniflowFallbackTargetHost != "" ||
		protocol.OmniflowFallbackTargetPort != 0 ||
		protocol.OmniflowFallbackHostHeader != "" ||
		protocol.OmniflowFallbackTLSSNI != "" {
		t.Fatalf("expected disabled OmniFlow fallback fields to be cleared, got %+v", protocol)
	}
}

func TestProtocolNormalizeOmniflowKeepsEnabledFallback(t *testing.T) {
	protocol := &Protocol{
		Type:                         "omniflow-h3",
		OmniflowFallbackEnabled:      true,
		OmniflowFallbackTargetScheme: " HTTP ",
		OmniflowFallbackTargetHost:   " www.example.com ",
		OmniflowFallbackHostHeader:   " fallback.example.com ",
		OmniflowFallbackTLSSNI:       " tls.example.com ",
	}

	protocol.NormalizeOmniflow()

	if !protocol.OmniflowFallbackEnabled {
		t.Fatal("fallback should stay enabled")
	}
	if protocol.OmniflowFallbackTargetScheme != "http" {
		t.Fatalf("expected normalized scheme, got %q", protocol.OmniflowFallbackTargetScheme)
	}
	if protocol.OmniflowFallbackTargetHost != "www.example.com" {
		t.Fatalf("expected trimmed target host, got %q", protocol.OmniflowFallbackTargetHost)
	}
	if protocol.OmniflowFallbackHostHeader != "fallback.example.com" {
		t.Fatalf("expected trimmed host header, got %q", protocol.OmniflowFallbackHostHeader)
	}
	if protocol.OmniflowFallbackTLSSNI != "tls.example.com" {
		t.Fatalf("expected trimmed TLS SNI, got %q", protocol.OmniflowFallbackTLSSNI)
	}
}

func TestProtocolNormalizeOmniflowDefaultsEnabledAfPath(t *testing.T) {
	protocol := &Protocol{
		Type:              "omniflow-h3",
		OmniflowAfEnabled: true,
	}

	protocol.NormalizeOmniflow()

	if protocol.OmniflowAfPathMode != "random" {
		t.Fatalf("expected default random path mode, got %q", protocol.OmniflowAfPathMode)
	}
	if protocol.OmniflowAfPathRotationSecs != 300 {
		t.Fatalf("expected default rotation 300, got %d", protocol.OmniflowAfPathRotationSecs)
	}
	if protocol.OmniflowAfPathSkewSlots != 1 {
		t.Fatalf("expected default skew slots 1, got %d", protocol.OmniflowAfPathSkewSlots)
	}
}
