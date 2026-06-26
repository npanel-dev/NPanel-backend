package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	"github.com/npanel-dev/NPanel-backend/internal/data"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/constant"
	pkgjwt "github.com/npanel-dev/NPanel-backend/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

const (
	defaultUploadDir     = "uploads"
	defaultUploadURLPath = "/uploads"
	defaultMaxUploadSize = 5 << 20
)

type uploadImageResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    uploadImageData `json:"data"`
}

type uploadImageData struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

type httpRouteRegistrar interface {
	HandleFunc(pattern string, handler nethttp.HandlerFunc)
	Handle(pattern string, handler nethttp.Handler)
}

func registerUploadRoutes(srv httpRouteRegistrar, c *conf.Server, d *data.Data) {
	if srv == nil {
		return
	}
	srv.HandleFunc("/v1/upload/image", handleUploadImage(c, d))
	srv.Handle(strings.TrimRight(defaultUploadURLPath, "/")+"/", uploadStaticHandler())
}

func handleUploadImage(c *conf.Server, d *data.Data) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method == nethttp.MethodOptions {
			w.WriteHeader(nethttp.StatusNoContent)
			return
		}
		if r.Method != nethttp.MethodPost {
			writeUploadError(w, nethttp.StatusMethodNotAllowed, responsecode.ErrInvalidParameter)
			return
		}
		if err := authenticateUploadRequest(c, d, r); err != nil {
			writeUploadError(w, nethttp.StatusOK, responsecode.ErrInvalidAccess)
			return
		}

		maxSize := uploadMaxSize()
		r.Body = nethttp.MaxBytesReader(w, r.Body, maxSize)
		if err := r.ParseMultipartForm(maxSize); err != nil {
			writeUploadError(w, nethttp.StatusOK, responsecode.ErrInvalidParameter)
			return
		}

		file, header, err := multipartFile(r)
		if err != nil {
			writeUploadError(w, nethttp.StatusOK, responsecode.ErrInvalidParameter)
			return
		}
		defer file.Close()

		relativePath, err := saveUploadedImage(file, header)
		if err != nil {
			writeUploadError(w, nethttp.StatusOK, responsecode.ErrInvalidParameter)
			return
		}

		writeUploadJSON(w, nethttp.StatusOK, uploadImageResponse{
			Code:    200,
			Message: "Success",
			Data: uploadImageData{
				URL:  absoluteUploadURL(r, relativePath),
				Path: relativePath,
			},
		})
	}
}

func authenticateUploadRequest(c *conf.Server, d *data.Data, r *nethttp.Request) error {
	if c != nil && c.Auth != nil && !c.Auth.EnableJwt {
		return nil
	}
	if d == nil || d.DB() == nil || d.Redis() == nil {
		return fmt.Errorf("upload auth dependencies unavailable")
	}

	token := strings.TrimSpace(r.Header.Get("Authorization"))
	if token == "" {
		return fmt.Errorf("missing token")
	}
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))

	claims, err := pkgjwt.ParseJwtToken(token, uploadJWTSecret(c))
	if err != nil {
		return err
	}
	userID, ok := jwtInt64Claim(claims, "UserId")
	if !ok {
		userID, ok = jwtInt64Claim(claims, "user_id")
	}
	if !ok || userID <= 0 {
		return fmt.Errorf("invalid user id")
	}

	sessionID := jwtStringClaim(claims, "SessionId")
	if sessionID == "" {
		sessionID = jwtStringClaim(claims, "session_id")
	}
	if sessionID == "" {
		return fmt.Errorf("invalid session id")
	}

	value, err := d.Redis().Get(r.Context(), fmt.Sprintf("%s:%s", constant.SessionIdKey, sessionID)).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if value != strconv.FormatInt(userID, 10) {
		return fmt.Errorf("invalid session")
	}

	userInfo, err := d.DB().ProxyUser.Query().Where(proxyuser.IDEQ(userID)).Only(r.Context())
	if err != nil {
		return err
	}
	if userInfo.DeletedAt != nil || (userInfo.IsDel != nil && *userInfo.IsDel == 0) || !userInfo.Enable {
		return fmt.Errorf("inactive user")
	}
	return nil
}

func multipartFile(r *nethttp.Request) (multipart.File, *multipart.FileHeader, error) {
	for _, field := range []string{"file", "image"} {
		file, header, err := r.FormFile(field)
		if err == nil {
			return file, header, nil
		}
	}
	return nil, nil, fmt.Errorf("missing upload file")
}

func saveUploadedImage(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("missing file header")
	}
	if header.Size <= 0 || header.Size > uploadMaxSize() {
		return "", fmt.Errorf("invalid file size")
	}

	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	contentType := nethttp.DetectContentType(head[:n])
	ext, ok := imageExtension(contentType)
	if !ok {
		return "", fmt.Errorf("unsupported image type")
	}

	datePath := time.Now().Format("2006/01/02")
	dstDir := filepath.Join(uploadRootDir(), "images", datePath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return "", err
	}

	name, err := randomFilename(ext)
	if err != nil {
		return "", err
	}
	dstPath := filepath.Join(dstDir, name)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(dstPath)
		return "", err
	}

	return strings.TrimRight(defaultUploadURLPath, "/") + "/images/" + datePath + "/" + name, nil
}

func uploadStaticHandler() nethttp.Handler {
	root := nethttp.Dir(uploadRootDir())
	prefix := strings.TrimRight(defaultUploadURLPath, "/") + "/"
	return nethttp.StripPrefix(prefix, nethttp.FileServer(root))
}

func imageExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func randomFilename(ext string) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf) + ext, nil
}

func uploadRootDir() string {
	if dir := strings.TrimSpace(os.Getenv("NPANEL_UPLOAD_DIR")); dir != "" {
		return dir
	}
	return defaultUploadDir
}

func uploadMaxSize() int64 {
	if value := strings.TrimSpace(os.Getenv("NPANEL_UPLOAD_MAX_MB")); value != "" {
		if mb, err := strconv.ParseInt(value, 10, 64); err == nil && mb > 0 {
			return mb << 20
		}
	}
	return defaultMaxUploadSize
}

func uploadJWTSecret(c *conf.Server) string {
	if c != nil && c.Auth != nil && strings.TrimSpace(c.Auth.JwtSecret) != "" {
		return strings.TrimSpace(c.Auth.JwtSecret)
	}
	if secret := strings.TrimSpace(os.Getenv("JWT_SECRET")); secret != "" {
		return secret
	}
	return "your-secret-key-change-in-production"
}

func jwtStringClaim(claims jwt.MapClaims, key string) string {
	if claims == nil {
		return ""
	}
	value, ok := claims[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func jwtInt64Claim(claims jwt.MapClaims, key string) (int64, bool) {
	if claims == nil {
		return 0, false
	}
	value, ok := claims[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case json.Number:
		parsed, err := v.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func absoluteUploadURL(r *nethttp.Request, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	host := firstForwardedValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	scheme := firstForwardedValue(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	if host == "" {
		return path
	}
	return scheme + "://" + host + path
}

func firstForwardedValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, ","); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

func writeUploadError(w nethttp.ResponseWriter, httpStatus int, code int) {
	message := responsecode.CodeMessages[code]
	if message == "" {
		message = "Error"
	}
	writeUploadJSON(w, httpStatus, map[string]any{
		"code":    code,
		"message": message,
	})
}

func writeUploadJSON(w nethttp.ResponseWriter, httpStatus int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(value)
}
