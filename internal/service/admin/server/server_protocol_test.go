package server

import (
	"testing"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/server/v1"
)

func TestAdminProtocolConversionKeepsMxMundoFields(t *testing.T) {
	input := &v1.Protocol{
		Type:                        "mx",
		Port:                        443,
		Enable:                      true,
		Security:                    "tls",
		Sni:                         "sni.example.com",
		Transport:                   "mundordp",
		MundoUsername:               "admin",
		MundoCertificateFingerprint: "sha256:abc",
		MundoFakeTitle:              "Remote Desktop",
		MundoFakeMessage:            "Access denied",
		MundoAcceptProxyProtocol:    true,
		MundoUseTlsCertificate:      true,
		ProxyProtocol:               true,
	}

	model := protoToModelProtocol(input)
	if model.Type != "mx" || model.Transport != "mundordp" {
		t.Fatalf("model protocol mismatch: %+v", model)
	}
	if model.MundoUsername != "admin" ||
		model.MundoCertificateFingerprint != "sha256:abc" ||
		model.MundoFakeTitle != "Remote Desktop" ||
		model.MundoFakeMessage != "Access denied" ||
		!model.MundoAcceptProxyProtocol ||
		!model.MundoUseTLSCertificate ||
		!model.ProxyProtocol {
		t.Fatalf("model lost Mundo fields: %+v", model)
	}

	output := modelProtocolToProto(model)
	if output.MundoUsername != input.MundoUsername ||
		output.MundoCertificateFingerprint != input.MundoCertificateFingerprint ||
		output.MundoFakeTitle != input.MundoFakeTitle ||
		output.MundoFakeMessage != input.MundoFakeMessage ||
		output.MundoAcceptProxyProtocol != input.MundoAcceptProxyProtocol ||
		output.MundoUseTlsCertificate != input.MundoUseTlsCertificate ||
		output.ProxyProtocol != input.ProxyProtocol {
		t.Fatalf("proto round-trip lost Mundo fields: input=%+v output=%+v", input, output)
	}
}
