package data

import (
	"encoding/json"
	"testing"

	"github.com/npanel-dev/NPanel-backend/ent"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
)

func TestCleanLegacyNodeProtocolsKeepsSimnetAfClientFields(t *testing.T) {
	raw, err := json.Marshal([]*servermodel.Protocol{
		{
			Type:                               "simnet",
			Port:                               443,
			Enable:                             true,
			SimnetPsk:                          " server-psk ",
			SimnetKeyID:                        0,
			SimnetCarrier:                      "",
			SimnetAfEnabled:                    true,
			SimnetAfPathMode:                   " random ",
			SimnetAfMagicMode:                  " derived ",
			SimnetAfResponseJitterMs:           0,
			SimnetAfHandshakePolymorphism:      true,
			SimnetAfSettingsJitter:             true,
			SimnetAfFakeHeaderInjection:        true,
			SimnetClientMaxConcurrentStreams:   0,
			SimnetClientMaxStreamsPerSession:   0,
			SimnetClientSessionIdleTimeoutSecs: 0,
			SimnetClientMaxUDPSessions:         0,
		},
		{
			Type:                             "vless",
			Port:                             8443,
			Enable:                           true,
			SimnetClientMaxConcurrentStreams: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cleaned := cleanLegacyNodeProtocols(string(raw))

	var protocols []*servermodel.Protocol
	if err := json.Unmarshal([]byte(cleaned), &protocols); err != nil {
		t.Fatalf("cleaned protocols should be valid json: %v\n%s", err, cleaned)
	}
	if len(protocols) != 2 {
		t.Fatalf("expected two protocols, got %d", len(protocols))
	}
	protocol := protocols[0]
	if !protocol.SimnetAfEnabled {
		t.Fatal("expected simnet_af_enabled to stay true")
	}
	if protocol.SimnetAfPathMode != "random" {
		t.Fatalf("expected trimmed path mode, got %q", protocol.SimnetAfPathMode)
	}
	if protocol.SimnetAfMagicMode != "derived" {
		t.Fatalf("expected trimmed magic mode, got %q", protocol.SimnetAfMagicMode)
	}
	if protocol.SimnetAfResponseJitterMs != 50 {
		t.Fatalf("expected default jitter 50, got %d", protocol.SimnetAfResponseJitterMs)
	}
	if !protocol.SimnetAfHandshakePolymorphism {
		t.Fatal("expected simnet_af_handshake_polymorphism to be present/true")
	}
	if !protocol.SimnetAfSettingsJitter {
		t.Fatal("expected simnet_af_settings_jitter to be present/true")
	}
	if !protocol.SimnetAfFakeHeaderInjection {
		t.Fatal("expected simnet_af_fake_header_injection to be present/true")
	}
	if protocol.SimnetCarrier != "h2" {
		t.Fatalf("expected default carrier h2, got %q", protocol.SimnetCarrier)
	}
	if protocol.SimnetPath != "/simnet/session" {
		t.Fatalf("expected default path, got %q", protocol.SimnetPath)
	}
	if protocol.SimnetPsk != "server-psk" {
		t.Fatalf("expected trimmed server psk, got %q", protocol.SimnetPsk)
	}
	if protocol.SimnetClientMaxConcurrentStreams != 32 ||
		protocol.SimnetClientMaxStreamsPerSession != 512 ||
		protocol.SimnetClientSessionIdleTimeoutSecs != 90 ||
		protocol.SimnetClientMaxUDPSessions != 64 {
		t.Fatalf("expected simnet client defaults, got %d/%d/%d/%d", protocol.SimnetClientMaxConcurrentStreams, protocol.SimnetClientMaxStreamsPerSession, protocol.SimnetClientSessionIdleTimeoutSecs, protocol.SimnetClientMaxUDPSessions)
	}
	if protocols[1].SimnetClientMaxConcurrentStreams != 0 || protocols[1].SimnetClientMaxUDPSessions != 0 {
		t.Fatalf("non-simnet protocol should not receive simnet defaults: %+v", protocols[1])
	}
}

func TestCleanLegacyNodeProtocolsDefaultsEnabledAfSubFeatures(t *testing.T) {
	raw, err := json.Marshal([]*servermodel.Protocol{
		{
			Type:                     "simnet",
			Port:                     443,
			Enable:                   true,
			SimnetAfEnabled:          true,
			SimnetAfPathMode:         "random",
			SimnetAfMagicMode:        "derived",
			SimnetAfResponseJitterMs: 50,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cleaned := cleanLegacyNodeProtocols(string(raw))

	var protocols []*servermodel.Protocol
	if err := json.Unmarshal([]byte(cleaned), &protocols); err != nil {
		t.Fatalf("cleaned protocols should be valid json: %v\n%s", err, cleaned)
	}
	if len(protocols) != 1 {
		t.Fatalf("expected one protocol, got %d", len(protocols))
	}
	protocol := protocols[0]
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

func TestCleanLegacyNodeProtocolInstanceKeepsOnlyMatchedPort(t *testing.T) {
	raw, err := json.Marshal([]*servermodel.Protocol{
		{
			Type:   "shadowsocks",
			Port:   443,
			Enable: true,
			Cipher: "chacha20-ietf-poly1305",
		},
		{
			Type:   "shadowsocks",
			Port:   8443,
			Enable: true,
			Cipher: "aes-256-gcm",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cleaned := cleanLegacyNodeProtocolInstance(string(raw), "shadowsocks", 8443)

	var protocols []*servermodel.Protocol
	if err := json.Unmarshal([]byte(cleaned), &protocols); err != nil {
		t.Fatalf("cleaned protocol instance should be valid json: %v\n%s", err, cleaned)
	}
	if len(protocols) != 1 {
		t.Fatalf("expected one matched protocol instance, got %d", len(protocols))
	}
	if protocols[0].Port != 8443 || protocols[0].Cipher != "aes-256-gcm" {
		t.Fatalf("matched protocol = %+v, want shadowsocks:8443/aes-256-gcm", protocols[0])
	}
}

func TestInjectLegacySimnetUserCredentialsForClient(t *testing.T) {
	uuid := "019EF9DF-8AEC-7EFF-AEA8-2B2F4860622E"
	raw, err := json.Marshal([]*servermodel.Protocol{
		{
			Type:        "simnet",
			Port:        443,
			Enable:      true,
			SimnetPsk:   " server-psk ",
			SimnetKeyID: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cleaned := cleanLegacyNodeProtocolInstance(string(raw), "simnet", 443)
	enriched := injectLegacySimnetUserCredentialsForClient(cleaned, &ent.ProxyUserSubscribe{
		ID:   5,
		UUID: &uuid,
	})

	var protocols []map[string]any
	if err := json.Unmarshal([]byte(enriched), &protocols); err != nil {
		t.Fatalf("enriched protocols should be valid json: %v\n%s", err, enriched)
	}
	if len(protocols) != 1 {
		t.Fatalf("expected one protocol, got %d", len(protocols))
	}
	protocol := protocols[0]
	if protocol["simnet_psk"] != "server-psk" {
		t.Fatalf("expected legacy simnet_psk to remain server psk, got %#v", protocol["simnet_psk"])
	}
	if protocol["simnet_server_psk"] != "server-psk" {
		t.Fatalf("expected server psk alias, got %#v", protocol["simnet_server_psk"])
	}
	if protocol["simnet_server_key_id"].(float64) != 0 {
		t.Fatalf("expected server key id 0, got %#v", protocol["simnet_server_key_id"])
	}
	if protocol["simnet_user_psk"] != "019ef9df8aec7effaea82b2f4860622e" {
		t.Fatalf("expected compact uuid user psk, got %#v", protocol["simnet_user_psk"])
	}
	if protocol["simnet_user_key_id"].(float64) != 5 {
		t.Fatalf("expected user subscribe id as user key id, got %#v", protocol["simnet_user_key_id"])
	}
}
