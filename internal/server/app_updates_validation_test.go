package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	systembiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/system"
)

func TestAppUpdateValidationErrorUsesPublicBusinessEnvelope(t *testing.T) {
	uc := systembiz.NewSystemUsecase(nil, log.NewStdLogger(io.Discard))
	err := uc.UpdateSiteConfig(context.Background(), &systembiz.SiteConfig{
		CustomData: `{"app_updates":{"android":{"version":"1.0.8","download_url":"file:///invalid"}}}`,
	})
	if err == nil {
		t.Fatal("expected validation failure before repository access")
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/v1/admin/system/site_config", nil)
	CustomErrorEncoder(recorder, request, err)
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || body.Code != 400 || !strings.Contains(body.Message, "app_updates.android.download_url") {
		t.Fatalf("unexpected response status=%d code=%d message=%s", recorder.Code, body.Code, body.Message)
	}
}
