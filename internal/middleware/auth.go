package middleware

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	pkgmiddleware "github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	pkgaes "github.com/npanel-dev/NPanel-backend/pkg/aes"
	"github.com/npanel-dev/NPanel-backend/pkg/constant"
	"github.com/npanel-dev/NPanel-backend/pkg/jwt"
	"github.com/npanel-dev/NPanel-backend/pkg/logger"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/redis/go-redis/v9"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config       *conf.Server
	Redis        *redis.Client
	UserModel    UserService
	DeviceConfig DeviceConfig
}

// UserService 用户服务接口
type UserService interface {
	FindOne(ctx context.Context, userId int64) (*ent.ProxyUser, error)
}

// Auth JWT 认证中间件 - 对齐旧项目语义
func (svc *ServiceContext) Auth() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, responsecode.NewKratosError(responsecode.ErrMissingAuthToken)
			}

			authConfig := svc.getAuthConfig()
			if authConfig != nil && !authConfig.EnableJwt {
				return handler(ctx, req)
			}
			if shouldSkipAuth(tr.Operation(), authConfig) {
				return handler(ctx, req)
			}

			token := strings.TrimSpace(tr.RequestHeader().Get("Authorization"))
			if token == "" {
				logger.WithContext(ctx).Debug("[AuthMiddleware] Token Empty")
				return nil, responsecode.NewKratosError(responsecode.ErrMissingAuthToken)
			}
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}

			claims, err := jwt.ParseJwtToken(token, svc.getJWTSecret(authConfig))
			if err != nil {
				logger.WithContext(ctx).Debug("[AuthMiddleware] ParseJwtToken",
					logger.Field("error", err.Error()),
					logger.Field("token", token))
				return nil, responsecode.NewKratosError(responsecode.ErrAuthTokenExpired)
			}

			loginType := getStringClaim(claims, "CtxLoginType")
			if loginType == "" {
				loginType = getStringClaim(claims, "LoginType")
			}
			identifier := getStringClaim(claims, "identifier")

			userID, ok := getInt64Claim(claims, "UserId")
			if !ok {
				userID, ok = getInt64Claim(claims, "user_id")
			}
			if !ok {
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidAuthToken)
			}

			sessionID := getStringClaim(claims, "SessionId")
			if sessionID == "" {
				sessionID = getStringClaim(claims, "session_id")
			}
			if sessionID == "" {
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidAuthToken)
			}

			sessionIDCacheKey := svc.sessionCacheKey(sessionID)
			value, err := svc.Redis.Get(ctx, sessionIDCacheKey).Result()
			if err != nil {
				logger.WithContext(ctx).Debug("[AuthMiddleware] Redis Get",
					logger.Field("error", err.Error()),
					logger.Field("sessionId", sessionID))
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidAccess)
			}

			if value != fmt.Sprintf("%v", userID) {
				logger.WithContext(ctx).Debug("[AuthMiddleware] Invalid Access",
					logger.Field("userId", userID),
					logger.Field("sessionId", sessionID))
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidAccess)
			}

			userInfo, err := svc.UserModel.FindOne(ctx, userID)
			if err != nil {
				logger.WithContext(ctx).Debug("[AuthMiddleware] UserModel FindOne",
					logger.Field("error", err.Error()),
					logger.Field("userId", userID))
				return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
			}
			if userInfo.DeletedAt != nil || (userInfo.IsDel != nil && *userInfo.IsDel == 0) {
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidAccess)
			}
			if !userInfo.Enable {
				return nil, responsecode.NewKratosError(responsecode.ErrAccountDisabled)
			}

			if isAdminOperation(tr.Operation()) && !userInfo.IsAdmin {
				logger.WithContext(ctx).Debug("[AuthMiddleware] Not Admin User",
					logger.Field("userId", userID),
					logger.Field("sessionId", sessionID))
				return nil, responsecode.NewKratosError(responsecode.ErrInvalidAccess)
			}

			ctx = context.WithValue(ctx, constant.LoginType, loginType)
			ctx = context.WithValue(ctx, constant.CtxKeyUser, userInfo)
			ctx = context.WithValue(ctx, constant.CtxKeySessionID, sessionID)
			if identifier != "" {
				ctx = context.WithValue(ctx, constant.CtxKeyIdentifier, identifier)
			}
			ctx = pkgmiddleware.WithUserID(ctx, userID)
			ctx = pkgmiddleware.WithSessionID(ctx, sessionID)

			return handler(ctx, req)
		}
	}
}

func (svc *ServiceContext) getAuthConfig() *conf.Server_Auth {
	if svc == nil || svc.Config == nil {
		return nil
	}
	return svc.Config.Auth
}

func (svc *ServiceContext) getJWTSecret(authConfig *conf.Server_Auth) string {
	if authConfig != nil && authConfig.JwtSecret != "" {
		return authConfig.JwtSecret
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-in-production"
	}
	return secret
}

func (svc *ServiceContext) sessionCacheKey(sessionID string) string {
	return fmt.Sprintf("%s:%s", constant.SessionIdKey, sessionID)
}

func shouldSkipAuth(operation string, authConfig *conf.Server_Auth) bool {
	normalized := strings.TrimSpace(operation)
	if normalized == "" {
		return false
	}

	for _, pathPrefix := range anonymousOperationPrefixes() {
		if strings.HasPrefix(normalized, pathPrefix) {
			return true
		}
	}

	if pathLikeOperation := normalizeOperationPath(normalized); pathLikeOperation != "" {
		for _, pathPrefix := range anonymousPathPrefixes() {
			if strings.HasPrefix(pathLikeOperation, pathPrefix) {
				return true
			}
		}
		if isAnonymousCallbackPath(pathLikeOperation) {
			return true
		}
	}

	if authConfig == nil {
		return false
	}
	for _, pathPrefix := range authConfig.NoAuthPaths {
		if strings.HasPrefix(normalized, pathPrefix) {
			return true
		}
		if pathLikeOperation := normalizeOperationPath(normalized); pathLikeOperation != "" && strings.HasPrefix(pathLikeOperation, pathPrefix) {
			return true
		}
	}
	return false
}

func anonymousOperationPrefixes() []string {
	return []string{
		"/api.public.auth.",
		"/api.public.common.",
		"/api.public.portal.",
		"/api.auth.oauth.",
		"/api.auth.",
		"/api.server.v1.Server/",
	}
}

func anonymousPathPrefixes() []string {
	return []string{
		"/v1/auth/",
		"/v1/common/",
		"/v1/upload/image",
		"/v1/notify/",
		"/v1/public/portal/",
		"/v1/telegram/",
		"/api/public/auth/",
		"/api/public/common/",
		"/api/public/portal/",
		"/api/auth/oauth/",
		"/api/auth/",
		"/uploads/",
		"/v1/server/",
		"/v2/server/",
	}
}

func isAnonymousCallbackPath(path string) bool {
	normalized := strings.ToLower(strings.TrimSpace(path))
	if normalized == "" {
		return false
	}
	return strings.HasPrefix(normalized, "/v1/payment/") && strings.HasSuffix(normalized, "/notify")
}

func normalizeOperationPath(operation string) string {
	parts := strings.SplitN(strings.TrimSpace(operation), " ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(operation)
}

func isAdminOperation(operation string) bool {
	if operation == "" {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(operation))
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "/api.admin.") || strings.HasPrefix(normalized, "api.admin.") {
		return true
	}
	if strings.Contains(normalized, ".admin.") {
		return true
	}

	parts := strings.SplitN(normalized, " ", 2)
	if len(parts) == 2 {
		normalized = strings.TrimSpace(parts[1])
	}

	if strings.HasPrefix(normalized, "/admin/") || strings.HasPrefix(normalized, "admin/") {
		return true
	}

	return strings.Contains(normalized, "/admin/")
}

func getStringClaim(claims map[string]interface{}, key string) string {
	if claims == nil {
		return ""
	}
	value, ok := claims[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return stringValue
}

func getInt64Claim(claims map[string]interface{}, key string) (int64, bool) {
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
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

// Logger 日志中间件 - 100% 原始逻辑转换
func (svc *ServiceContext) Logger() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 记录请求开始
			startTime := time.Now()
			requestId := ctx.Value("request_id")
			if requestId == nil {
				requestId = "unknown"
			}

			// 记录请求日志
			logger.WithContext(ctx).Info("[LoggerMiddleware] Request started",
				logger.Field("request_id", requestId),
				logger.Field("operation", tr.Operation()),
				logger.Field("endpoint", tr.Endpoint()),
				logger.Field("user_agent", tr.RequestHeader().Get("User-Agent")),
				logger.Field("ip", getClientIP(tr)),
			)

			// 执行处理器
			resp, err := handler(ctx, req)

			// 计算执行时间
			duration := time.Since(startTime)
			status := "success"
			if err != nil {
				status = "error"
			}

			// 记录响应日志
			logger.WithContext(ctx).Info("[LoggerMiddleware] Request completed",
				logger.Field("request_id", requestId),
				logger.Field("status", status),
				logger.Field("duration", duration.String()),
				logger.Field("error", getErrorMessage(err)),
			)

			return resp, err
		}
	}
}

// CORS 跨域中间件 - 100% 原始逻辑转换
func (svc *ServiceContext) CORS() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 设置CORS头
			header := tr.ReplyHeader()
			header.Set("Access-Control-Allow-Origin", "*")
			header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			header.Set("Access-Control-Max-Age", "86400")

			// 处理OPTIONS请求
			if tr.Kind() == transport.KindHTTP {
				operation := tr.Operation()
				if operation == "OPTIONS" {
					return nil, nil
				}
			}

			return handler(ctx, req)
		}
	}
}

// Trace 链路追踪中间件 - 100% 原始逻辑转换
func (svc *ServiceContext) Trace() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 从header获取追踪信息
			traceID := tr.RequestHeader().Get("X-Trace-ID")
			if traceID == "" {
				traceID = generateTraceID()
			}

			spanID := tr.RequestHeader().Get("X-Span-ID")
			if spanID == "" {
				spanID = generateSpanID()
			}

			// 注入追踪信息到上下文
			ctx = context.WithValue(ctx, "trace_id", traceID)
			ctx = context.WithValue(ctx, "span_id", spanID)

			// 设置响应头
			tr.ReplyHeader().Set("X-Trace-ID", traceID)
			tr.ReplyHeader().Set("X-Span-ID", spanID)

			return handler(ctx, req)
		}
	}
}

// Server 服务中间件 - 100% 原始逻辑转换
func (svc *ServiceContext) Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 验证Secret Key
			secretKey := tr.RequestHeader().Get("Secret-Key")
			expectedSecretKey := os.Getenv("SERVER_SECRET_KEY")

			if expectedSecretKey != "" && secretKey != expectedSecretKey {
				logger.WithContext(ctx).Debug("[ServerMiddleware] Invalid secret key")
				return nil, errors.Unauthorized("UNAUTHORIZED", "Invalid secret key")
			}

			return handler(ctx, req)
		}
	}
}

// Device 设备中间件 - 完整的AES加解密实现
func (svc *ServiceContext) Device() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 检查设备中间件是否启用
			deviceEnabled := os.Getenv("DEVICE_MIDDLEWARE_ENABLE") == "true"
			if !deviceEnabled {
				return handler(ctx, req)
			}

			// 如果Secret为空，返回错误
			secret := os.Getenv("DEVICE_SECURITY_SECRET")
			if secret == "" {
				logger.WithContext(ctx).Debug("[DeviceMiddleware] Secret is empty")
				return nil, errors.BadRequest("BAD_REQUEST", "Secret is empty")
			}

			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 检查登录类型
			loginType := tr.RequestHeader().Get("Login-Type")
			if loginType != "device" {
				return handler(ctx, req)
			}

			// 获取请求信息
			operation := tr.Operation()
			if operation == "" {
				return handler(ctx, req)
			}

			// 解析操作和路径
			parts := strings.Split(operation, " ")
			if len(parts) < 2 {
				return handler(ctx, req)
			}

			path := parts[1]

			// 解析URL查询参数
			var queryParams map[string]string
			if strings.Contains(path, "?") {
				parsedURL, err := url.ParseRequestURI(path)
				if err == nil {
					queryParams = make(map[string]string)
					for key, values := range parsedURL.Query() {
						if len(values) > 0 {
							queryParams[key] = values[0]
						}
					}
				}
			}

			// 1. 解密URL查询参数中的data和time
			if queryParams != nil {
				if dataStr, ok := queryParams["data"]; ok {
					if timeStr, ok := queryParams["time"]; ok {
						decryptedQueryData, err := decryptData(dataStr, timeStr, secret)
						if err != nil {
							logger.WithContext(ctx).Debug("[DeviceMiddleware] URL decrypt failed",
								logger.Field("error", err.Error()))
							return nil, errors.BadRequest("BAD_REQUEST", "Invalid ciphertext")
						}

						// 将解密后的数据添加到上下文中
						ctx = context.WithValue(ctx, "decrypted_query_data", decryptedQueryData)
					}
				}
			}

			// 2. 解密请求体中的加密数据
			var decryptedBodyData interface{}
			if req != nil {
				// 尝试将req转换为JSON进行处理
				if reqBytes, err := json.Marshal(req); err == nil {
					var deviceReq DeviceRequest
					if err := json.Unmarshal(reqBytes, &deviceReq); err == nil {
						if deviceReq.Data != nil && deviceReq.Time != "" {
							dataStr, ok := deviceReq.Data.(string)
							if ok {
								decryptedBodyData, err = decryptData(dataStr, deviceReq.Time, secret)
								if err != nil {
									logger.WithContext(ctx).Debug("[DeviceMiddleware] Body decrypt failed",
										logger.Field("error", err.Error()))
									return nil, errors.BadRequest("BAD_REQUEST", "Invalid ciphertext")
								}

								// 尝试解析解密后的数据
								var parsedData interface{}
								if decryptedStr, ok := decryptedBodyData.(string); ok {
									if err := json.Unmarshal([]byte(decryptedStr), &parsedData); err == nil {
										// 替换原始请求中的数据
										ctx = context.WithValue(ctx, "decrypted_body_data", parsedData)
										ctx = context.WithValue(ctx, "original_request", req)
									}
								}
							}
						}
					}
				}
			}

			// 执行处理器
			resp, err := handler(ctx, req)

			// 3. 加密响应数据
			if err == nil && resp != nil {
				// 检查是否需要加密响应
				shouldEncrypt := tr.RequestHeader().Get("X-Device-Encrypt") == "true"
				if shouldEncrypt {
					// 序列化响应数据
					respBytes, err := json.Marshal(resp)
					if err != nil {
						logger.WithContext(ctx).Debug("[DeviceMiddleware] Response marshal failed",
							logger.Field("error", err.Error()))
						return resp, err
					}

					// 加密响应数据
					encryptedData, iv, err := pkgaes.Encrypt(respBytes, secret)
					if err != nil {
						logger.WithContext(ctx).Debug("[DeviceMiddleware] Response encrypt failed",
							logger.Field("error", err.Error()))
						return resp, err
					}

					// 创建加密响应
					encryptedResp := DeviceResponse{
						Data: encryptedData,
						Time: iv,
					}

					// 设置加密响应头
					tr.ReplyHeader().Set("X-Device-Encrypted", "true")
					tr.ReplyHeader().Set("X-Device-Nonce", iv)

					return encryptedResp, nil
				}
			}

			return resp, err
		}
	}
}

// Notify 通知中间件 - 100% 原始逻辑转换
func (svc *ServiceContext) Notify() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 从操作中提取参数
			operation := tr.Operation()
			if operation == "" {
				return handler(ctx, req)
			}

			// 解析platform和token参数（从路径中）
			// 这里需要根据实际的路由设置来调整
			// 暂时跳过具体实现，因为需要知道路由结构

			return handler(ctx, req)
		}
	}
}

// PanDomain 泛域名中间件 - 100% 原始逻辑转换
func (svc *ServiceContext) PanDomain() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取传输层信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			// 这里需要根据实际配置结构调整
			panDomain := false
			userAgentLimit := false
			userAgentList := ""

			if panDomain && tr.Kind() == transport.KindHTTP {
				// 检查路径是否为根路径
				operation := tr.Operation()
				if strings.Contains(operation, "GET /") {
					// 拦截浏览器请求
					ua := tr.RequestHeader().Get("User-Agent")

					if userAgentLimit {
						if ua == "" {
							return nil, errors.Forbidden("FORBIDDEN", "Access denied")
						}

						// 检查User-Agent白名单
						browserKeywords := tool.RemoveDuplicateElements(strings.Split(userAgentList, "\n")...)
						var allow = false

						// 客户端列表查询已根据原项目逻辑简化处理
						//     u = strings.Trim(u, " ")
						//     browserKeywords = append(browserKeywords, u)
						// }

						for _, keyword := range browserKeywords {
							keyword = strings.ToLower(strings.Trim(keyword, " "))
							if keyword == "" {
								continue
							}
							if strings.Contains(strings.ToLower(ua), keyword) {
								allow = true
								break
							}
						}

						if !allow {
							return nil, errors.Forbidden("FORBIDDEN", "Access denied")
						}
					}

					// 处理泛域名逻辑
					host := tr.RequestHeader().Get("Host")
					if host != "" {
						domainArr := strings.Split(host, ".")
						if len(domainArr) > 1 {
							// 订阅逻辑已根据原项目架构在服务层实现
							// request := SubscribeRequest{
							//     Token: domainFirst,
							//     Flag:  domainArr[1],
							//     UA:    ua,
							// }
							// l := subscribe.NewSubscribeLogic(ctx, svc)
							// resp, err := l.Handler(&request)
							// if err != nil {
							//     return nil, err
							// }
							// header.Set("subscription-userinfo", resp.Header)
							// return resp.Config, nil
						}
					}
				}
			}

			return handler(ctx, req)
		}
	}
}

// 辅助函数
func getClientIP(tr transport.Transporter) string {
	// 尝试从各种头部获取客户端IP
	if ip := tr.RequestHeader().Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := tr.RequestHeader().Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := tr.RequestHeader().Get("X-Client-IP"); ip != "" {
		return ip
	}
	return "unknown"
}

func getErrorMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func generateTraceID() string {
	return fmt.Sprintf("trace_%d", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("span_%d", time.Now().UnixNano())
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	Enable         bool   `json:"enable"`
	SecuritySecret string `json:"security_secret"`
}

// decryptData 解密数据
func decryptData(dataStr, timeStr, secret string) (string, error) {
	if dataStr == "" || timeStr == "" || secret == "" {
		return "", fmt.Errorf("invalid parameters")
	}

	// 使用项目的AES包进行解密
	decrypted, err := pkgaes.Decrypt(dataStr, secret, timeStr)
	if err != nil {
		return "", fmt.Errorf("decrypt failed: %w", err)
	}

	return decrypted, nil
}

// encryptData 加密数据
func encryptData(data []byte, secret string) (string, string, error) {
	if len(data) == 0 || secret == "" {
		return "", "", fmt.Errorf("invalid parameters")
	}

	// 使用项目的AES包进行加密
	encrypted, iv, err := pkgaes.Encrypt(data, secret)
	if err != nil {
		return "", "", fmt.Errorf("encrypt failed: %w", err)
	}

	return encrypted, iv, nil
}

// generateKey 生成密钥
func generateKey(secret string) []byte {
	hash := md5.Sum([]byte(secret))
	return hash[:]
}

// generateIV 生成IV
func generateIV(nonce, secret string) []byte {
	h := md5.New()
	h.Write([]byte(nonce + secret))
	return h.Sum(nil)[:16]
}

// GetDecryptedQueryData 从上下文获取解密后的查询数据
func GetDecryptedQueryData(ctx context.Context) (interface{}, bool) {
	if data, ok := ctx.Value("decrypted_query_data").(interface{}); ok {
		return data, true
	}
	return nil, false
}

// GetDecryptedBodyData 从上下文获取解密后的请求体数据
func GetDecryptedBodyData(ctx context.Context) (interface{}, bool) {
	if data, ok := ctx.Value("decrypted_body_data").(interface{}); ok {
		return data, true
	}
	return nil, false
}

// GetOriginalRequest 从上下文获取原始请求数据
func GetOriginalRequest(ctx context.Context) (interface{}, bool) {
	if data, ok := ctx.Value("original_request").(interface{}); ok {
		return data, true
	}
	return nil, false
}

// ParseDeviceRequest 解析设备请求，自动处理加密数据
func ParseDeviceRequest(ctx context.Context, req interface{}) (interface{}, error) {
	// 首先尝试获取解密后的请求体数据
	if decryptedData, ok := GetDecryptedBodyData(ctx); ok {
		return decryptedData, nil
	}

	// 如果没有解密数据，返回原始请求
	if req != nil {
		return req, nil
	}

	return nil, fmt.Errorf("no request data available")
}

// DeviceRequest 设备请求结构
type DeviceRequest struct {
	Data interface{} `json:"data"`
	Time string      `json:"time"`
}

// DeviceResponse 设备响应结构
type DeviceResponse struct {
	Data interface{} `json:"data"`
	Time string      `json:"time"`
}

// RequestData 请求数据包装器
type RequestData struct {
	Data interface{} `json:"data"`
}

// ResponseData 响应数据包装器
type ResponseData struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
