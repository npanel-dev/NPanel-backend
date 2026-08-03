package middleware

import (
	"testing"

	"github.com/npanel-dev/NPanel-backend/internal/conf"
)

func TestShouldSkipAuthLegacyAnonymousCoverage(t *testing.T) {
	t.Parallel()

	authConfig := &conf.Server_Auth{}

	tests := []struct {
		name      string
		operation string
		want      bool
	}{
		{
			name:      "public auth http path",
			operation: "POST /v1/auth/login",
			want:      true,
		},
		{
			name:      "public common http path",
			operation: "GET /v1/common/site/config",
			want:      true,
		},
		{
			name:      "public portal http path",
			operation: "POST /v1/public/portal/purchase",
			want:      true,
		},
		{
			name:      "payment callback http path",
			operation: "POST /v1/payment/demo-token/alipay/notify",
			want:      true,
		},
		{
			name:      "notify callback http path",
			operation: "POST /v1/notify/epay/demo-token",
			want:      true,
		},
		{
			name:      "telegram webhook http path",
			operation: "POST /v1/telegram/webhook",
			want:      true,
		},
		{
			name:      "public portal grpc operation",
			operation: "/api.public.portal.v1.Portal/Purchase",
			want:      true,
		},
		{
			name:      "server http path",
			operation: "GET /v1/server/config",
			want:      true,
		},
		{
			name:      "server grpc operation",
			operation: "/api.server.v1.Server/GetServerConfig",
			want:      true,
		},
		{
			name:      "protected public user http path",
			operation: "GET /v1/public/user/info",
			want:      false,
		},
		{
			name:      "protected public payment method path",
			operation: "GET /v1/public/payment/methods",
			want:      false,
		},
		{
			name:      "protected public routing config path",
			operation: "GET /v1/public/routing/config",
			want:      false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldSkipAuth(tc.operation, authConfig)
			if got != tc.want {
				t.Fatalf("shouldSkipAuth(%q) = %v, want %v", tc.operation, got, tc.want)
			}
		})
	}
}
