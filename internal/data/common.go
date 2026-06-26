package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/hibiken/asynq"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyads"
	"github.com/npanel-dev/NPanel-backend/ent/proxyauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	"github.com/npanel-dev/NPanel-backend/ent/proxysystem"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	v1 "github.com/npanel-dev/NPanel-backend/internal/biz/common"
	authmodel "github.com/npanel-dev/NPanel-backend/internal/model/auth"
	"github.com/npanel-dev/NPanel-backend/internal/queue/types"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/limit"
	"github.com/npanel-dev/NPanel-backend/pkg/phone"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type commonRepo struct {
	data *Data
	log  *log.Helper
}

// CacheKeyPayload stores verification code in Redis
type CacheKeyPayload struct {
	Code   string `json:"code"`
	LastAt int64  `json:"lastAt"`
}

// NewCommonRepo creates a new common repository
func NewCommonRepo(data *Data, logger log.Logger) v1.CommonRepo {
	return &commonRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/common")),
	}
}

// GetAdsList gets ads list by status
func (r *commonRepo) GetAdsList(ctx context.Context, status int) ([]*v1.Ads, error) {
	// Query ads with status filter, limit to 200 items
	entAds, err := r.data.db.ProxyAds.Query().
		Where(
			proxyads.Status(status),
		).
		Order(func(s *sql.Selector) {
			s.OrderBy(
				sql.Desc(proxyads.FieldStartTime),
				sql.Desc(proxyads.FieldCreatedAt),
				sql.Desc(proxyads.FieldID),
			)
		}).
		Limit(200).
		All(ctx)
	if err != nil {
		r.log.Errorw("GetAdsList query failed", "error", err, "status", status)
		return nil, err
	}

	// Convert ent objects to biz objects
	list := make([]*v1.Ads, len(entAds))
	for i, entAd := range entAds {
		list[i] = &v1.Ads{
			ID:          entAd.ID,
			Title:       entAd.Title,
			Type:        entAd.Type,
			Content:     entAd.Content,
			Description: entAd.Description,
			TargetURL:   entAd.TargetURL,
			StartTime:   entAd.StartTime.Unix(),
			EndTime:     entAd.EndTime.Unix(),
			Status:      int(entAd.Status),
			CreatedAt:   entAd.CreatedAt.Unix(),
			UpdatedAt:   entAd.UpdatedAt.Unix(),
		}
	}

	return list, nil
}

// GetClientList retrieves subscribe application list
func (r *commonRepo) GetClientList(ctx context.Context) ([]*v1.SubscribeClient, error) {
	entClients, err := r.data.db.ProxySubscribeApplication.Query().
		All(ctx)
	if err != nil {
		r.log.Errorw("GetClientList query failed", "error", err)
		return nil, err
	}

	result := make([]*v1.SubscribeClient, 0, len(entClients))
	for _, entClient := range entClients {
		// Parse download_link JSON
		var downloadLink v1.DownloadLink
		if entClient.DownloadLink != "" {
			if err := json.Unmarshal([]byte(entClient.DownloadLink), &downloadLink); err != nil {
				r.log.Warnw("Failed to unmarshal download_link", "error", err, "id", entClient.ID)
			}
		}

		client := &v1.SubscribeClient{
			ID:           int64(entClient.ID),
			Name:         entClient.Name,
			Scheme:       entClient.Scheme,
			IsDefault:    entClient.IsDefault,
			DownloadLink: downloadLink,
		}
		if entClient.Description != nil {
			client.Description = *entClient.Description
		}
		if entClient.Icon != nil {
			client.Icon = *entClient.Icon
		}
		result = append(result, client)
	}

	return result, nil
}

// GetTosConfig retrieves TOS/Privacy config from proxy_system table
func (r *commonRepo) GetTosConfig(ctx context.Context, key string) (string, error) {
	values, err := loadSystemConfigMap(ctx, r.data.db, "tos")
	if err != nil {
		r.log.Warnw("GetTosConfig query failed", "error", err, "key", key)
		return "", err
	}

	return systemConfigString(values, key), nil
}

// GetSystemConfigByCategory retrieves system config by category and returns as map
func (r *commonRepo) GetSystemConfigByCategory(ctx context.Context, category string) (map[string]string, error) {
	result, err := loadSystemConfigMap(ctx, r.data.db, category)
	if err != nil {
		r.log.Warnw("GetSystemConfigByCategory query failed", "error", err, "category", category)
		// Return empty map if not found (not an error)
		return make(map[string]string), nil
	}
	return result, nil
}

// GetWebAdConfig retrieves WebAD config
func (r *commonRepo) GetWebAdConfig(ctx context.Context) (bool, error) {
	entSystem, err := r.data.db.ProxySystem.Query().
		Where(
			proxysystem.Key("WebAD"),
		).
		First(ctx)
	if err != nil {
		r.log.Warnw("GetWebAdConfig query failed", "error", err)
		// Return false if not found (not an error)
		return false, nil
	}

	return entSystem.Value == "true", nil
}

// GetEnabledAuthMethods retrieves enabled auth methods
func (r *commonRepo) GetEnabledAuthMethods(ctx context.Context) ([]string, error) {
	entMethods, err := r.data.db.ProxyAuthMethod.Query().
		Where(
			proxyauthmethod.Enabled(true),
		).
		All(ctx)
	if err != nil {
		r.log.Warnw("GetEnabledAuthMethods query failed", "error", err)
		return []string{}, nil
	}

	var methods []string
	for _, method := range entMethods {
		methods = append(methods, method.Method)
	}

	return methods, nil
}

// GetStatistics retrieves system statistics
func (r *commonRepo) GetStatistics(ctx context.Context) (*v1.Statistics, error) {
	userCount, err := r.data.db.ProxyUser.Query().
		Where(proxyuser.EnableEQ(true)).
		Count(ctx)
	if err != nil {
		r.log.Errorw("GetStatistics user count failed", "error", err)
		return nil, err
	}

	roundedUserCount := int64(userCount)
	if roundedUserCount > 100 {
		roundedUserCount -= roundedUserCount % 100
	} else if roundedUserCount > 10 {
		roundedUserCount -= roundedUserCount % 10
	} else {
		roundedUserCount = 1
	}

	nodeCount, err := r.data.db.ProxyNode.Query().
		Where(proxynode.EnabledEQ(true)).
		Count(ctx)
	if err != nil {
		r.log.Errorw("GetStatistics node count failed", "error", err)
		return nil, err
	}

	serverAddresses, err := r.data.db.ProxyServer.Query().
		Select(proxyserver.FieldServerAddr).
		Strings(ctx)
	if err != nil {
		r.log.Errorw("GetStatistics server addresses failed", "error", err)
		return nil, err
	}

	nodeProtocols, err := r.data.db.ProxyNode.Query().
		Where(proxynode.EnabledEQ(true)).
		Select(proxynode.FieldProtocol).
		Strings(ctx)
	if err != nil {
		r.log.Errorw("GetStatistics protocol list failed", "error", err)
		return nil, err
	}

	countryCount := int64(len(fetchCountryCodes(ctx, serverAddresses)))
	protocols := collectProtocolTypes(nodeProtocols)

	stat := &v1.Statistics{
		User:     roundedUserCount,
		Node:     int64(nodeCount),
		Country:  countryCount,
		Protocol: protocols,
	}

	return stat, nil
}

func (r *commonRepo) ensureEmailChannelReady(ctx context.Context) error {
	method, err := r.data.db.ProxyAuthMethod.Query().
		Where(proxyauthmethod.MethodEQ("email")).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			r.log.Warnw("SendEmailVerificationCode email auth method not found")
			return responsecode.NewKratosError(responsecode.ErrEmailNotEnabled)
		}
		r.log.Errorw("SendEmailVerificationCode load email auth method error", "error", err)
		return responsecode.NewDatabaseQueryError()
	}

	var emailCfg authmodel.EmailAuthConfig
	emailCfg.Unmarshal(method.Config)
	if !method.Enabled || emailCfg.Platform != "smtp" || !isSMTPConfigured(&emailCfg) {
		r.log.Warnw("SendEmailVerificationCode email channel not ready", "enabled", method.Enabled, "platform", emailCfg.Platform)
		return responsecode.NewKratosError(responsecode.ErrEmailNotEnabled)
	}
	return nil
}

func isSMTPConfigured(cfg *authmodel.EmailAuthConfig) bool {
	if cfg == nil || cfg.PlatformConfig == nil {
		return false
	}
	raw, err := json.Marshal(cfg.PlatformConfig)
	if err != nil {
		return false
	}
	var smtpCfg authmodel.SMTPConfig
	if err := json.Unmarshal(raw, &smtpCfg); err != nil {
		return false
	}
	return smtpCfg.Host != "" && smtpCfg.Port > 0 && smtpCfg.From != ""
}

// SendEmailVerificationCode sends email verification code
func (r *commonRepo) SendEmailVerificationCode(ctx context.Context, email string, verifyType int32) error {
	if err := r.ensureEmailChannelReady(ctx); err != nil {
		return err
	}

	scene := parseVerifyType(verifyType)
	cacheKey := verifyCodeEmailCacheKey(scene, email)
	verifyCfg := r.loadVerifyCodeConfig(ctx)

	// Old project uses a fixed 60-second interval limiter with the legacy Redis key format.
	intervalLimiter := limit.NewPeriodLimit(60, 1, r.data.rdb, fmt.Sprintf("%s:%s:%s", SendIntervalKeyPrefix, "email", scene))
	permit, err := intervalLimiter.Take(email)
	if err != nil {
		r.log.Errorw("SendEmailVerificationCode interval limiter error", "error", err, "email", email)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if !intervalLimiter.ParsePermitState(permit) {
		r.log.Warnw("SendEmailVerificationCode interval limit exceeded", "email", email)
		return responsecode.NewKratosError(responsecode.ErrTooManyRequests)
	}

	dailyLimiter := limit.NewPeriodLimit(86400, int(verifyCfg.Limit), r.data.rdb, SendCountLimitKeyPrefix, limit.Align())
	permit, err = dailyLimiter.Take(fmt.Sprintf("%s:%s:%s", "email", scene, email))
	if err != nil {
		r.log.Errorw("SendEmailVerificationCode daily limiter error", "error", err, "email", email)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if !dailyLimiter.ParsePermitState(permit) {
		r.log.Warnw("SendEmailVerificationCode daily limit exceeded", "email", email)
		return responsecode.NewKratosError(responsecode.ErrTodaySendCountExceedsLimit)
	}

	authMethod, err := r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthType("email"),
			proxyuserauthmethod.AuthIdentifier(email),
		).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		r.log.Errorw("SendEmailVerificationCode query user error", "error", err, "email", email)
		return responsecode.NewDatabaseQueryError()
	}

	if scene == verifySceneRegister && authMethod != nil {
		r.log.Warnw("SendEmailVerificationCode user already exists", "email", email)
		return responsecode.NewKratosError(responsecode.ErrUserAlreadyExists)
	} else if scene == verifySceneSecurity && authMethod == nil {
		r.log.Warnw("SendEmailVerificationCode user not found", "email", email)
		return responsecode.NewKratosError(responsecode.ErrUserNotExist)
	}

	code := tool.KeyNew(6, 0)
	cachePayload := CacheKeyPayload{
		Code:   code,
		LastAt: time.Now().Unix(),
	}

	val, err := json.Marshal(cachePayload)
	if err != nil {
		r.log.Errorw("SendEmailVerificationCode marshal error", "error", err, "email", email)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	if err := r.data.rdb.Set(ctx, cacheKey, string(val), 5*time.Minute).Err(); err != nil {
		r.log.Errorw("SendEmailVerificationCode Redis error", "error", err, "cache_key", cacheKey)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	siteLogo := ""
	siteName := ""
	if appConf := r.data.AppConf(); appConf != nil && appConf.Site != nil {
		siteLogo = appConf.Site.SiteLogo
		siteName = appConf.Site.SiteName
	}
	emailPayload := types.SendEmailPayload{
		Type:    types.EmailTypeVerify,
		Email:   email,
		Subject: "Verification code",
		Content: map[string]interface{}{
			"Type":     verifyType,
			"SiteLogo": siteLogo,
			"SiteName": siteName,
			"Expire":   5,
			"Code":     code,
		},
	}

	payloadBytes, err := json.Marshal(emailPayload)
	if err != nil {
		r.log.Errorw("SendEmailVerificationCode marshal task payload error", "error", err, "email", email)
		return responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	task := asynq.NewTask(types.ForthwithSendEmail, payloadBytes, asynq.MaxRetry(3))

	taskInfo, err := r.data.queue.Enqueue(task)
	if err != nil {
		r.log.Errorw("SendEmailVerificationCode enqueue error", "error", err, "payload", string(payloadBytes))
		return responsecode.NewKratosError(responsecode.ErrQueueEnqueueError)
	}

	r.log.Infow("Email verification code sent", "email", email, "code", code, "task_id", taskInfo.ID, "cache_key", cacheKey)

	return nil
}

// SendSmsVerificationCode sends SMS verification code
func (r *commonRepo) SendSmsVerificationCode(ctx context.Context, telephone, telephoneArea string, verifyType int32) (string, error) {
	phoneNumber, err := phone.FormatToE164(telephoneArea, telephone)
	if err != nil {
		r.log.Errorw("SendSmsVerificationCode invalid phone number", "error", err, "telephone", telephone, "area", telephoneArea)
		return "", responsecode.NewKratosError(responsecode.ErrTelephoneError)
	}

	scene := parseVerifyType(verifyType)
	cacheKey := verifyCodeTelephoneCacheKey(scene, phoneNumber)
	verifyCfg := r.loadVerifyCodeConfig(ctx)

	intervalLimiter := limit.NewPeriodLimit(60, 1, r.data.rdb, fmt.Sprintf("%s:%s:%s", SendIntervalKeyPrefix, "mobile", scene))
	permit, err := intervalLimiter.Take(phoneNumber)
	if err != nil {
		r.log.Errorw("SendSmsVerificationCode interval limiter error", "error", err, "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if !intervalLimiter.ParsePermitState(permit) {
		r.log.Warnw("SendSmsVerificationCode interval limit exceeded", "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrTooManyRequests)
	}

	dailyLimiter := limit.NewPeriodLimit(86400, int(verifyCfg.Limit), r.data.rdb, SendCountLimitKeyPrefix, limit.Align())
	permit, err = dailyLimiter.Take(fmt.Sprintf("%s:%s:%s", "mobile", scene, phoneNumber))
	if err != nil {
		r.log.Errorw("SendSmsVerificationCode daily limiter error", "error", err, "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if !dailyLimiter.ParsePermitState(permit) {
		r.log.Warnw("SendSmsVerificationCode daily limit exceeded", "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrTodaySendCountExceedsLimit)
	}

	authMethod, err := r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthType("mobile"),
			proxyuserauthmethod.AuthIdentifier(phoneNumber),
		).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		r.log.Errorw("SendSmsVerificationCode query user error", "error", err, "phone", phoneNumber)
		return "", responsecode.NewDatabaseQueryError()
	}

	if scene == verifySceneRegister && authMethod != nil {
		r.log.Warnw("SendSmsVerificationCode user already exists", "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrUserAlreadyExists)
	} else if scene == verifySceneSecurity && authMethod == nil {
		r.log.Warnw("SendSmsVerificationCode user not found", "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrUserNotExist)
	}

	code := tool.KeyNew(6, 0)
	cachePayload := CacheKeyPayload{
		Code:   code,
		LastAt: time.Now().Unix(),
	}

	val, err := json.Marshal(cachePayload)
	if err != nil {
		r.log.Errorw("SendSmsVerificationCode marshal error", "error", err, "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	if err := r.data.rdb.Set(ctx, cacheKey, string(val), time.Second*time.Duration(verifyCfg.ExpireTime)).Err(); err != nil {
		r.log.Errorw("SendSmsVerificationCode Redis error", "error", err, "cache_key", cacheKey)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	smsPayload := types.SendSmsPayload{
		Type:          verifyType,
		Telephone:     telephone,
		TelephoneArea: telephoneArea,
		Content:       code,
	}

	payloadBytes, err := json.Marshal(smsPayload)
	if err != nil {
		r.log.Errorw("SendSmsVerificationCode marshal task payload error", "error", err, "phone", phoneNumber)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	task := asynq.NewTask(types.ForthwithSendSms, payloadBytes)

	taskInfo, err := r.data.queue.Enqueue(task)
	if err != nil {
		r.log.Errorw("SendSmsVerificationCode enqueue error", "error", err, "payload", string(payloadBytes))
		return "", responsecode.NewKratosError(responsecode.ErrQueueEnqueueError)
	}

	r.log.Infow("SMS verification code sent", "phone", phoneNumber, "code", code, "task_id", taskInfo.ID, "cache_key", cacheKey)

	return code, nil
}

// CheckVerificationCode checks verification code
func (r *commonRepo) CheckVerificationCode(ctx context.Context, method, account, code string, verifyType int32) (bool, error) {
	var cacheKey string

	if method == "email" {
		cacheKey = verifyCodeEmailCacheKey(parseVerifyType(verifyType), account)
	} else if method == "mobile" {
		if !phone.CheckPhone(account) {
			return false, responsecode.NewKratosError(responsecode.ErrTelephoneError)
		}
		cacheKey = verifyCodeTelephoneCacheKey(parseVerifyType(verifyType), "+"+account)
	} else {
		r.log.Warnw("CheckVerificationCode invalid method", "method", method)
		return false, responsecode.NewKratosError(responsecode.ErrInvalidParameter)
	}

	// Get from Redis
	value, err := r.data.rdb.Get(ctx, cacheKey).Result()
	if err != nil {
		r.log.Warnw("CheckVerificationCode Redis get error", "error", err, "cache_key", cacheKey)
		return false, nil
	}

	// Unmarshal payload
	var payload CacheKeyPayload
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		r.log.Warnw("CheckVerificationCode unmarshal error", "error", err)
		return false, nil
	}

	// Compare code
	if payload.Code != code {
		r.log.Warnw("CheckVerificationCode code mismatch", "expected", payload.Code, "got", code)
		return false, nil
	}

	r.log.Infow("Verification code validated", "method", method, "account", account)
	return true, nil
}

type runtimeVerifyCodeConfig struct {
	ExpireTime int64
	Limit      int64
	Interval   int64
}

func (r *commonRepo) loadVerifyCodeConfig(ctx context.Context) runtimeVerifyCodeConfig {
	result := runtimeVerifyCodeConfig{
		ExpireTime: 300,
		Limit:      15,
		Interval:   60,
	}

	values, err := loadSystemConfigMap(ctx, r.data.db, "verify_code")
	if err != nil {
		r.log.Warnw("loadVerifyCodeConfig query failed, using defaults", "error", err)
		return result
	}

	result.ExpireTime = getInt64Config(values, result.ExpireTime, "VerifyCodeExpireTime", "verify_code_expire_time")
	result.Limit = getInt64Config(values, result.Limit, "VerifyCodeLimit", "verify_code_limit")
	result.Interval = getInt64Config(values, result.Interval, "VerifyCodeInterval", "verify_code_interval")
	return result
}

type ipAPIBatchRequest struct {
	Query  string `json:"query"`
	Fields string `json:"fields"`
}

type ipAPIBatchResponse struct {
	CountryCode string `json:"countryCode"`
}

type nodeProtocolItem struct {
	Type string `json:"type"`
}

func fetchCountryCodes(ctx context.Context, addresses []string) map[string]struct{} {
	countries := make(map[string]struct{})
	requests := make([]ipAPIBatchRequest, 0, len(addresses))

	for _, address := range addresses {
		address = resolveAddressToIP(address)
		if address == "" {
			continue
		}
		requests = append(requests, ipAPIBatchRequest{
			Query:  address,
			Fields: "countryCode",
		})
	}

	if len(requests) == 0 {
		return countries
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for start := 0; start < len(requests); start += 100 {
		end := start + 100
		if end > len(requests) {
			end = len(requests)
		}

		body, err := json.Marshal(requests[start:end])
		if err != nil {
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://ip-api.com/batch", bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var payload []ipAPIBatchResponse
		if err := json.Unmarshal(respBody, &payload); err != nil {
			continue
		}

		for _, item := range payload {
			if item.CountryCode == "" {
				continue
			}
			countries[item.CountryCode] = struct{}{}
		}
	}

	return countries
}

func resolveAddressToIP(address string) string {
	if ip := net.ParseIP(address); ip != nil {
		return ip.String()
	}

	records, err := net.LookupIP(address)
	if err != nil {
		return ""
	}

	for _, record := range records {
		if record == nil {
			continue
		}
		if ipv4 := record.To4(); ipv4 != nil {
			return ipv4.String()
		}
		return record.String()
	}

	return ""
}

func collectProtocolTypes(protocolValues []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, raw := range protocolValues {
		for _, item := range parseNodeProtocolTypes(raw) {
			if item == "" {
				continue
			}
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func parseNodeProtocolTypes(raw string) []string {
	if raw == "" {
		return nil
	}

	var list []nodeProtocolItem
	if err := json.Unmarshal([]byte(raw), &list); err == nil && len(list) > 0 {
		result := make([]string, 0, len(list))
		for _, item := range list {
			if item.Type != "" {
				result = append(result, item.Type)
			}
		}
		return result
	}

	return []string{raw}
}
