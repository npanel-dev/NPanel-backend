package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
)

func TestUploadImageSavesLocalFile(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("NPANEL_UPLOAD_DIR", uploadDir)

	body, contentType := multipartBody(t, "file", "logo.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89,
	})
	req := httptest.NewRequest(http.MethodPost, "https://panel.example.com/v1/upload/image", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handleUploadImage(authDisabledServerConfig(), nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp uploadImageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 200 || resp.Data.Path == "" || resp.Data.URL == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if got, want := resp.Data.URL[:len("https://panel.example.com/uploads/")], "https://panel.example.com/uploads/"; got != want {
		t.Fatalf("unexpected url prefix: got %q want %q", got, want)
	}
	localPath := filepath.Join(uploadDir, filepath.FromSlash(resp.Data.Path[len("/uploads/"):]))
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("uploaded file not found: %v", err)
	}
}

func TestUploadImageRejectsNonImage(t *testing.T) {
	t.Setenv("NPANEL_UPLOAD_DIR", t.TempDir())
	body, contentType := multipartBody(t, "file", "note.txt", []byte("not an image"))
	req := httptest.NewRequest(http.MethodPost, "https://panel.example.com/v1/upload/image", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	handleUploadImage(authDisabledServerConfig(), nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["code"] == float64(200) {
		t.Fatalf("non-image upload unexpectedly succeeded: %s", rec.Body.String())
	}
}

func TestAbsoluteUploadURLUsesForwardedCustomerHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/v1/upload/image", nil)
	req.Host = "127.0.0.1:8081"
	req.Header.Set("X-Forwarded-Host", "panel.customer.example, internal.proxy")
	req.Header.Set("X-Forwarded-Proto", "https, http")

	got := absoluteUploadURL(req, "/uploads/images/2026/06/27/logo.webp")
	want := "https://panel.customer.example/uploads/images/2026/06/27/logo.webp"
	if got != want {
		t.Fatalf("unexpected upload url: got %q want %q", got, want)
	}
}

func TestUploadStaticRouteServesNestedImagePath(t *testing.T) {
	uploadDir := t.TempDir()
	t.Setenv("NPANEL_UPLOAD_DIR", uploadDir)

	relativePath := filepath.FromSlash("images/2026/06/27/logo.webp")
	localPath := filepath.Join(uploadDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		t.Fatalf("create upload dir: %v", err)
	}
	content := []byte("webp image")
	if err := os.WriteFile(localPath, content, 0644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}

	srv := kratoshttp.NewServer()
	registerUploadRoutes(srv, authDisabledServerConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/uploads/images/2026/06/27/logo.webp", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(rec.Body.Bytes(), content) {
		t.Fatalf("unexpected body: got %q want %q", rec.Body.Bytes(), content)
	}
}

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func authDisabledServerConfig() *conf.Server {
	return &conf.Server{Auth: &conf.Server_Auth{EnableJwt: false}}
}
