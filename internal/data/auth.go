package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserauthmethod"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuserdevice"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	authbiz "github.com/npanel-dev/NPanel-backend/internal/biz/auth"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/captcha"
	"github.com/npanel-dev/NPanel-backend/pkg/phone"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
	"github.com/redis/go-redis/v9"
)

const (
	LogTypeLogin    int8 = 30
	LogTypeRegister int8 = 31

	DefaultJWTSecret = "your-secret-key-change-in-production"
	DefaultJWTExpire = 604800

	registerIPKeyPrefix = "register:ip:"
)

type authRepo struct {
	data   *Data
	log    *log.Helper
	config *conf.Application
}

type LoginLog struct {
	Method    string `json:"method"`
	LoginIP   string `json:"login_ip"`
	UserAgent string `json:"user_agent"`
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
}

type RegisterLog struct {
	AuthMethod string `json:"auth_method"`
	Identifier string `json:"identifier"`
	RegisterIP string `json:"register_ip"`
	UserAgent  string `json:"user_agent"`
	Timestamp  int64  `json:"timestamp"`
}

type runtimeVerifyConfig struct {
	CaptchaType                    string
	TurnstileSiteKey               string
	TurnstileSecret                string
	EnableUserLoginCaptcha         bool
	EnableUserRegisterCaptcha      bool
	EnableAdminLoginCaptcha        bool
	EnableUserResetPasswordCaptcha bool
}

type runtimeRegisterConfig struct {
	StopRegister            bool
	EnableIpRegisterLimit   bool
	IpRegisterLimit         int64
	IpRegisterLimitDuration int64
	EnableTrial             bool
	TrialSubscribe          int64
	TrialTime               int64
	TrialTimeUnit           string
}

type runtimeInviteConfig struct {
	ForcedInvite      bool
	OnlyFirstPurchase bool
}

func NewAuthRepo(data *Data, config *conf.Application, logger log.Logger) authbiz.AuthRepo {
	return &authRepo{
		data:   data,
		config: config,
		log:    log.NewHelper(log.With(logger, "module", "data/auth")),
	}
}

func (r *authRepo) GetVerifyConfig(ctx context.Context) (*authbiz.VerifyConfig, error) {
	cfg, err := r.loadVerifyConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &authbiz.VerifyConfig{
		CaptchaType:                    cfg.CaptchaType,
		TurnstileSiteKey:               cfg.TurnstileSiteKey,
		TurnstileSecret:                cfg.TurnstileSecret,
		EnableUserLoginCaptcha:         cfg.EnableUserLoginCaptcha,
		EnableUserRegisterCaptcha:      cfg.EnableUserRegisterCaptcha,
		EnableAdminLoginCaptcha:        cfg.EnableAdminLoginCaptcha,
		EnableUserResetPasswordCaptcha: cfg.EnableUserResetPasswordCaptcha,
	}, nil
}

func (r *authRepo) VerifyCaptcha(ctx context.Context, config *authbiz.VerifyConfig, meta authbiz.RequestMeta) error {
	if config == nil {
		return nil
	}

	return captcha.VerifyCaptcha(ctx, r.data.rdb, config.CaptchaType, config.TurnstileSecret, captcha.VerifyInput{
		CaptchaID:   meta.CaptchaID,
		CaptchaCode: meta.CaptchaCode,
		CfToken:     meta.CfToken,
		SliderToken: meta.SliderToken,
		IP:          meta.IP,
	})
}

func (r *authRepo) CheckUserExistByEmail(ctx context.Context, email string) (bool, error) {
	_, err := r.getAuthMethod(ctx, "email", email)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil
		}
		r.log.Errorw("CheckUserExistByEmail failed", "error", err, "email", email)
		return false, responsecode.NewDatabaseQueryError()
	}
	return true, nil
}

func (r *authRepo) CheckUserExistByTelephone(ctx context.Context, telephoneAreaCode, telephone string) (bool, error) {
	phoneNumber, err := phone.FormatToE164(telephoneAreaCode, telephone)
	if err != nil {
		return false, responsecode.NewKratosError(responsecode.ErrTelephoneError)
	}

	_, err = r.getAuthMethod(ctx, "mobile", phoneNumber)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil
		}
		r.log.Errorw("CheckUserExistByTelephone failed", "error", err, "phone", phoneNumber)
		return false, responsecode.NewDatabaseQueryError()
	}
	return true, nil
}

func (r *authRepo) UserLogin(ctx context.Context, params *authbiz.UserLoginParams) (*authbiz.LoginResult, error) {
	var userID int64
	loginStatus := false

	defer func() {
		if userID != 0 {
			r.logLogin(ctx, int(userID), "email", params.Meta.IP, params.Meta.UserAgent, loginStatus)
		}
	}()

	authMethod, userInfo, err := r.getUserByAuth(ctx, "email", params.Email)
	if err != nil {
		return nil, err
	}
	userID = authMethod.UserID

	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	if !tool.MultiPasswordVerify(userInfo.Algo, stringPointerValue(userInfo.Salt), params.Password, userInfo.Password) {
		return nil, responsecode.NewKratosError(responsecode.ErrPasswordIncorrect)
	}

	r.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err := r.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (r *authRepo) TelephoneLogin(ctx context.Context, params *authbiz.TelephoneLoginParams) (*authbiz.LoginResult, error) {
	if !r.mobileEnabled() {
		return nil, responsecode.NewKratosError(responsecode.ErrSmsNotEnabled)
	}

	phoneNumber, err := phone.FormatToE164(params.TelephoneAreaCode, params.Telephone)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrTelephoneError)
	}

	var userID int64
	loginStatus := false
	defer func() {
		if userID != 0 {
			r.logLogin(ctx, int(userID), "mobile", params.Meta.IP, params.Meta.UserAgent, loginStatus)
		}
	}()

	authMethod, userInfo, err := r.getUserByAuth(ctx, "mobile", phoneNumber)
	if err != nil {
		return nil, err
	}
	userID = authMethod.UserID

	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	if params.Password == "" && params.TelephoneCode == "" {
		return nil, responsecode.NewKratosError(responsecode.ErrPasswordOrVerificationCodeRequired)
	}
	if params.TelephoneCode != "" {
		cacheKey := verifyCodeTelephoneCacheKey(verifySceneSecurity, phoneNumber)
		if err := r.checkVerificationCode(ctx, cacheKey, params.TelephoneCode); err != nil {
			return nil, err
		}
		r.data.rdb.Del(ctx, cacheKey)
	} else if !tool.MultiPasswordVerify(userInfo.Algo, stringPointerValue(userInfo.Salt), params.Password, userInfo.Password) {
		return nil, responsecode.NewKratosError(responsecode.ErrPasswordIncorrect)
	}

	r.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err := r.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (r *authRepo) UserRegister(ctx context.Context, params *authbiz.UserRegisterParams) (*authbiz.LoginResult, error) {
	var userID int64
	var token string

	defer func() {
		if userID != 0 && token != "" {
			r.logLogin(ctx, int(userID), "email", params.Meta.IP, params.Meta.UserAgent, true)
			r.logRegister(ctx, int(userID), "email", params.Email, params.Meta.IP, params.Meta.UserAgent)
		}
	}()

	registerCfg, err := r.loadRegisterConfig(ctx)
	if err != nil {
		return nil, err
	}
	if registerCfg.StopRegister {
		return nil, responsecode.NewKratosError(responsecode.ErrStopRegister)
	}

	inviteCfg, err := r.loadInviteConfig(ctx)
	if err != nil {
		return nil, err
	}
	refererID, err := r.resolveInvite(ctx, params.Invite, inviteCfg.ForcedInvite)
	if err != nil {
		return nil, err
	}
	if r.emailVerificationEnabled() {
		if err := r.checkVerificationCode(ctx, verifyCodeEmailCacheKey(verifySceneRegister, params.Email), params.Code); err != nil {
			return nil, err
		}
	}

	authMethod, err := r.getAuthMethod(ctx, "email", params.Email)
	if err != nil && !ent.IsNotFound(err) {
		r.log.Errorw("UserRegister query auth method failed", "error", err, "email", params.Email)
		return nil, responsecode.NewDatabaseQueryError()
	}
	if err == nil && authMethod != nil {
		existingUser, getUserErr := r.data.db.ProxyUser.Get(ctx, authMethod.UserID)
		if getUserErr != nil {
			r.log.Errorw("UserRegister get existing user failed", "error", getUserErr, "email", params.Email, "user_id", authMethod.UserID)
			return nil, responsecode.NewDatabaseQueryError()
		}
		if isDeletedUser(existingUser) {
			return nil, responsecode.NewKratosError(responsecode.ErrAccountDisabled)
		}
		return nil, responsecode.NewKratosError(responsecode.ErrDuplicateEmail)
	}
	if ok, err := r.checkRegisterIPLimit(ctx, registerCfg, params.Meta.IP, "email", params.Email); err != nil {
		return nil, err
	} else if !ok {
		return nil, responsecode.NewKratosError(responsecode.ErrRegisterIPLimit)
	}

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		r.log.Errorw("UserRegister start tx failed", "error", err)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	userInfo, err := tx.ProxyUser.Create().
		SetPassword(tool.EncodePassWord(params.Password)).
		SetAlgo("default").
		SetOnlyFirstPurchase(inviteCfg.OnlyFirstPurchase).
		SetNillableRefererID(refererID).
		Save(ctx)
	if err != nil {
		r.log.Errorw("UserRegister create user failed", "error", err)
		tx.Rollback()
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	referCode := tool.GenerateReferCode(userInfo.ID)
	userInfo, err = tx.ProxyUser.UpdateOneID(userInfo.ID).
		SetReferCode(referCode).
		Save(ctx)
	if err != nil {
		r.log.Errorw("UserRegister update refer code failed", "error", err, "user_id", userInfo.ID)
		tx.Rollback()
		return nil, responsecode.NewDatabaseUpdateError()
	}

	_, err = tx.ProxyUserAuthMethod.Create().
		SetUserID(userInfo.ID).
		SetAuthType("email").
		SetAuthIdentifier(params.Email).
		SetVerified(r.emailVerificationEnabled()).
		Save(ctx)
	if err != nil {
		r.log.Errorw("UserRegister create auth method failed", "error", err, "user_id", userInfo.ID)
		tx.Rollback()
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	if err := tx.Commit(); err != nil {
		r.log.Errorw("UserRegister commit failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	userID = userInfo.ID
	if registerCfg.EnableTrial {
		if trialSub, err := r.createTrialSubscription(ctx, userInfo.ID, registerCfg); err != nil {
			r.log.Errorw("UserRegister create trial subscription failed", "error", err, "user_id", userInfo.ID)
		} else if trialSub != nil {
			r.clearTrialCaches(ctx, trialSub)
			r.triggerGroupRecalculation(ctx)
		}
	}

	r.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err = r.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	return &authbiz.LoginResult{Token: token}, nil
}

func (r *authRepo) TelephoneRegister(ctx context.Context, params *authbiz.TelephoneRegisterParams) (*authbiz.LoginResult, error) {
	if !r.mobileEnabled() {
		return nil, responsecode.NewKratosError(responsecode.ErrSmsNotEnabled)
	}

	registerCfg, err := r.loadRegisterConfig(ctx)
	if err != nil {
		return nil, err
	}
	if registerCfg.StopRegister {
		return nil, responsecode.NewKratosError(responsecode.ErrStopRegister)
	}

	phoneNumber, err := phone.FormatToE164(params.TelephoneAreaCode, params.Telephone)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrTelephoneError)
	}

	var userID int64
	var token string
	defer func() {
		if userID != 0 && token != "" {
			r.logLogin(ctx, int(userID), "mobile", params.Meta.IP, params.Meta.UserAgent, true)
			r.logRegister(ctx, int(userID), "mobile", phoneNumber, params.Meta.IP, params.Meta.UserAgent)
		}
	}()

	cacheKey := verifyCodeTelephoneCacheKey(verifySceneRegister, phoneNumber)
	if err := r.checkVerificationCode(ctx, cacheKey, params.Code); err != nil {
		return nil, err
	}
	r.data.rdb.Del(ctx, cacheKey)

	authMethod, err := r.getAuthMethod(ctx, "mobile", phoneNumber)
	if err != nil && !ent.IsNotFound(err) {
		r.log.Errorw("TelephoneRegister query auth method failed", "error", err, "phone", phoneNumber)
		return nil, responsecode.NewDatabaseQueryError()
	}
	if err == nil && authMethod != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrTelephoneExist)
	}

	inviteCfg, err := r.loadInviteConfig(ctx)
	if err != nil {
		return nil, err
	}
	refererID, err := r.resolveInvite(ctx, params.Invite, inviteCfg.ForcedInvite)
	if err != nil {
		return nil, err
	}
	if ok, err := r.checkRegisterIPLimit(ctx, registerCfg, params.Meta.IP, "mobile", phoneNumber); err != nil {
		return nil, err
	} else if !ok {
		return nil, responsecode.NewKratosError(responsecode.ErrRegisterIPLimit)
	}

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		r.log.Errorw("TelephoneRegister start tx failed", "error", err)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	userInfo, err := tx.ProxyUser.Create().
		SetPassword(tool.EncodePassWord(params.Password)).
		SetAlgo("default").
		SetOnlyFirstPurchase(inviteCfg.OnlyFirstPurchase).
		SetNillableRefererID(refererID).
		Save(ctx)
	if err != nil {
		r.log.Errorw("TelephoneRegister create user failed", "error", err)
		tx.Rollback()
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	referCode := tool.GenerateReferCode(userInfo.ID)
	userInfo, err = tx.ProxyUser.UpdateOneID(userInfo.ID).
		SetReferCode(referCode).
		Save(ctx)
	if err != nil {
		r.log.Errorw("TelephoneRegister update refer code failed", "error", err, "user_id", userInfo.ID)
		tx.Rollback()
		return nil, responsecode.NewDatabaseUpdateError()
	}

	_, err = tx.ProxyUserAuthMethod.Create().
		SetUserID(userInfo.ID).
		SetAuthType("mobile").
		SetAuthIdentifier(phoneNumber).
		SetVerified(true).
		Save(ctx)
	if err != nil {
		r.log.Errorw("TelephoneRegister create auth method failed", "error", err, "user_id", userInfo.ID)
		tx.Rollback()
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
	}

	var trialSub *ent.ProxyUserSubscribe
	if registerCfg.EnableTrial {
		trialSub, err = r.createTrialSubscriptionTx(ctx, tx, userInfo.ID, registerCfg)
		if err != nil {
			r.log.Errorw("TelephoneRegister create trial subscription failed", "error", err, "user_id", userInfo.ID)
			tx.Rollback()
			return nil, responsecode.NewKratosError(responsecode.ErrDatabaseInsert)
		}
	}

	if err := tx.Commit(); err != nil {
		r.log.Errorw("TelephoneRegister commit failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	userID = userInfo.ID
	if trialSub != nil {
		r.clearTrialCaches(ctx, trialSub)
	}

	r.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err = r.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	return &authbiz.LoginResult{Token: token}, nil
}

func (r *authRepo) ResetPassword(ctx context.Context, params *authbiz.ResetPasswordParams) (*authbiz.LoginResult, error) {
	var userID int64
	loginStatus := false

	defer func() {
		if userID != 0 && loginStatus {
			r.logLogin(ctx, int(userID), "email", params.Meta.IP, params.Meta.UserAgent, true)
		}
	}()

	if err := r.checkVerificationCode(ctx, verifyCodeEmailCacheKey(verifySceneSecurity, params.Email), params.Code); err != nil {
		return nil, err
	}

	authMethod, userInfo, err := r.getUserByAuth(ctx, "email", params.Email)
	if err != nil {
		return nil, err
	}
	userID = authMethod.UserID
	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	_, err = r.data.db.ProxyUser.UpdateOneID(userInfo.ID).
		SetPassword(tool.EncodePassWord(params.Password)).
		SetAlgo("default").
		ClearSalt().
		Save(ctx)
	if err != nil {
		r.log.Errorw("ResetPassword update password failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	r.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err := r.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (r *authRepo) TelephoneResetPassword(ctx context.Context, params *authbiz.TelephoneResetPasswordParams) (*authbiz.LoginResult, error) {
	if !r.mobileEnabled() {
		return nil, responsecode.NewKratosError(responsecode.ErrSmsNotEnabled)
	}

	phoneNumber, err := phone.FormatToE164(params.TelephoneAreaCode, params.Telephone)
	if err != nil {
		return nil, responsecode.NewKratosError(responsecode.ErrTelephoneError)
	}

	var userID int64
	loginStatus := false
	defer func() {
		if userID != 0 && loginStatus {
			r.logLogin(ctx, int(userID), "mobile", params.Meta.IP, params.Meta.UserAgent, true)
		}
	}()

	if err := r.checkVerificationCode(ctx, verifyCodeTelephoneCacheKey(verifySceneSecurity, phoneNumber), params.Code); err != nil {
		return nil, err
	}

	authMethod, userInfo, err := r.getUserByAuth(ctx, "mobile", phoneNumber)
	if err != nil {
		return nil, err
	}
	userID = authMethod.UserID
	if err := ensureUserActive(userInfo); err != nil {
		return nil, err
	}
	_, err = r.data.db.ProxyUser.UpdateOneID(userInfo.ID).
		SetPassword(tool.EncodePassWord(params.Password)).
		SetAlgo("default").
		ClearSalt().
		Save(ctx)
	if err != nil {
		r.log.Errorw("TelephoneResetPassword update password failed", "error", err, "user_id", userInfo.ID)
		return nil, responsecode.NewDatabaseUpdateError()
	}

	r.bindDeviceSafely(ctx, params.Meta, userInfo.ID)

	token, err := r.issueSessionToken(ctx, userInfo.ID, params.Meta)
	if err != nil {
		return nil, err
	}

	loginStatus = true
	return &authbiz.LoginResult{Token: token}, nil
}

func (r *authRepo) getAuthMethod(ctx context.Context, authType, identifier string) (*ent.ProxyUserAuthMethod, error) {
	return r.data.db.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthTypeEQ(authType),
			proxyuserauthmethod.AuthIdentifierEQ(identifier),
		).
		Only(ctx)
}

func (r *authRepo) getUserByAuth(ctx context.Context, authType, identifier string) (*ent.ProxyUserAuthMethod, *ent.ProxyUser, error) {
	authMethod, userInfo, err := r.queryUserByAuth(ctx, authType, identifier)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, responsecode.NewKratosError(responsecode.ErrUserNotFound)
		}
		r.log.Errorw("getUserByAuth query failed", "error", err, "auth_type", authType, "identifier", identifier)
		return nil, nil, responsecode.NewDatabaseQueryError()
	}

	return authMethod, userInfo, nil
}

func (r *authRepo) queryUserByAuth(ctx context.Context, authType, identifier string) (*ent.ProxyUserAuthMethod, *ent.ProxyUser, error) {
	authMethod, err := r.getAuthMethod(ctx, authType, identifier)
	if err != nil {
		return nil, nil, err
	}

	userInfo, err := r.data.db.ProxyUser.Get(ctx, authMethod.UserID)
	if err != nil {
		return nil, nil, err
	}

	return authMethod, userInfo, nil
}

func (r *authRepo) resolveInvite(ctx context.Context, invite string, forced bool) (*int64, error) {
	if invite == "" {
		if forced {
			return nil, responsecode.NewKratosError(responsecode.ErrInviteCodeError)
		}
		return nil, nil
	}

	referer, err := r.data.db.ProxyUser.Query().
		Where(proxyuser.ReferCodeEQ(invite)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewKratosError(responsecode.ErrInviteCodeError)
		}
		r.log.Errorw("resolveInvite query failed", "error", err, "invite", invite)
		return nil, responsecode.NewDatabaseQueryError()
	}

	refererID := referer.ID
	return &refererID, nil
}

func (r *authRepo) checkVerificationCode(ctx context.Context, cacheKey, code string) error {
	if strings.TrimSpace(code) == "" {
		return responsecode.NewKratosError(responsecode.ErrVerifyCodeError)
	}

	value, err := r.data.rdb.Get(ctx, cacheKey).Result()
	if err != nil {
		if err != redis.Nil {
			r.log.Warnw("checkVerificationCode redis get failed", "error", err, "cache_key", cacheKey)
		}
		return responsecode.NewKratosError(responsecode.ErrVerifyCodeError)
	}

	var payload CacheKeyPayload
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		r.log.Warnw("checkVerificationCode unmarshal failed", "error", err, "cache_key", cacheKey)
		return responsecode.NewKratosError(responsecode.ErrVerifyCodeError)
	}
	if payload.Code != code {
		return responsecode.NewKratosError(responsecode.ErrVerifyCodeError)
	}

	return nil
}

func (r *authRepo) issueSessionToken(ctx context.Context, userID int64, meta authbiz.RequestMeta) (string, error) {
	token, err := r.data.issueSessionToken(ctx, userID, sessionTokenOptions{
		Identifier: meta.Identifier,
		LoginType:  meta.LoginType,
	})
	if err != nil {
		r.log.Errorw("issueSessionToken failed", "error", err, "user_id", userID)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}

	return token, nil
}

func (r *authRepo) issueDeviceSessionToken(ctx context.Context, userID int64, meta authbiz.RequestMeta) (string, error) {
	token, sessionID, err := r.data.issueSessionTokenWithSessionID(ctx, userID, sessionTokenOptions{
		Identifier: meta.Identifier,
		LoginType:  meta.LoginType,
	})
	if err != nil {
		r.log.Errorw("issueDeviceSessionToken failed", "error", err, "user_id", userID)
		return "", responsecode.NewKratosError(responsecode.ErrInternalError)
	}
	if strings.TrimSpace(meta.Identifier) != "" {
		deviceKey := fmt.Sprintf("%s:%s", legacyDeviceIdentifierSessionPrefix, meta.Identifier)
		if setErr := r.data.rdb.Set(ctx, deviceKey, sessionID, time.Duration(r.data.jwtExpireSeconds())*time.Second).Err(); setErr != nil {
			r.log.Errorw("issueDeviceSessionToken set device cache failed", "error", setErr, "user_id", userID, "identifier", meta.Identifier)
			return "", responsecode.NewKratosError(responsecode.ErrInternalError)
		}
	}
	return token, nil
}

func (r *authRepo) loadVerifyConfig(ctx context.Context) (*runtimeVerifyConfig, error) {
	configs, err := r.getSystemConfigMap(ctx, "verify")
	if err != nil {
		return nil, err
	}

	result := &runtimeVerifyConfig{
		CaptchaType:      getStringConfig(configs, "CaptchaType", "captcha_type"),
		TurnstileSiteKey: getStringConfig(configs, "TurnstileSiteKey", "turnstile_site_key"),
		TurnstileSecret:  getStringConfig(configs, "TurnstileSecret", "turnstile_secret"),
	}
	if r.config != nil && r.config.Verify != nil {
		result.TurnstileSiteKey = firstNonEmpty(result.TurnstileSiteKey, r.config.Verify.TurnstileSiteKey)
		result.EnableUserLoginCaptcha = r.config.Verify.EnableLoginVerify
		result.EnableUserRegisterCaptcha = r.config.Verify.EnableRegisterVerify
		result.EnableUserResetPasswordCaptcha = r.config.Verify.EnableResetPasswordVerify
	}
	result.EnableUserLoginCaptcha = getBoolConfig(configs, result.EnableUserLoginCaptcha, "EnableUserLoginCaptcha", "enable_user_login_captcha", "EnableLoginVerify", "enable_login_verify")
	result.EnableUserRegisterCaptcha = getBoolConfig(configs, result.EnableUserRegisterCaptcha, "EnableUserRegisterCaptcha", "enable_user_register_captcha", "EnableRegisterVerify", "enable_register_verify")
	result.EnableAdminLoginCaptcha = getBoolConfig(configs, result.EnableAdminLoginCaptcha, "EnableAdminLoginCaptcha", "enable_admin_login_captcha")
	result.EnableUserResetPasswordCaptcha = getBoolConfig(configs, result.EnableUserResetPasswordCaptcha, "EnableUserResetPasswordCaptcha", "enable_user_reset_password_captcha", "EnableResetPasswordVerify", "enable_reset_password_verify")
	if result.CaptchaType == "" {
		if result.TurnstileSiteKey != "" || result.TurnstileSecret != "" {
			result.CaptchaType = string(captcha.CaptchaTypeTurnstile)
		}
	}
	return result, nil
}

func (r *authRepo) loadRegisterConfig(ctx context.Context) (*runtimeRegisterConfig, error) {
	configs, err := r.getSystemConfigMap(ctx, "register")
	if err != nil {
		return nil, err
	}

	result := &runtimeRegisterConfig{}
	if r.config != nil && r.config.Register != nil {
		result.StopRegister = r.config.Register.StopRegister
		result.EnableIpRegisterLimit = r.config.Register.EnableIpRegisterLimit
		result.IpRegisterLimit = r.config.Register.IpRegisterLimit
		result.IpRegisterLimitDuration = r.config.Register.IpRegisterLimitDuration
		result.EnableTrial = r.config.Register.EnableTrial
		result.TrialSubscribe = r.config.Register.TrialSubscribe
		result.TrialTime = r.config.Register.TrialTime
		result.TrialTimeUnit = r.config.Register.TrialTimeUnit
	}
	result.StopRegister = getBoolConfig(configs, result.StopRegister, "StopRegister", "stop_register")
	result.EnableIpRegisterLimit = getBoolConfig(configs, result.EnableIpRegisterLimit, "EnableIpRegisterLimit", "enable_ip_register_limit")
	result.IpRegisterLimit = getInt64Config(configs, result.IpRegisterLimit, "IpRegisterLimit", "ip_register_limit")
	result.IpRegisterLimitDuration = getInt64Config(configs, result.IpRegisterLimitDuration, "IpRegisterLimitDuration", "ip_register_limit_duration")
	result.EnableTrial = getBoolConfig(configs, result.EnableTrial, "EnableTrial", "enable_trial")
	result.TrialSubscribe = getInt64Config(configs, result.TrialSubscribe, "TrialSubscribe", "trial_subscribe")
	result.TrialTime = getInt64Config(configs, result.TrialTime, "TrialTime", "trial_time")
	result.TrialTimeUnit = getStringConfigWithDefault(configs, result.TrialTimeUnit, "TrialTimeUnit", "trial_time_unit")
	return result, nil
}

func (r *authRepo) loadInviteConfig(ctx context.Context) (*runtimeInviteConfig, error) {
	configs, err := r.getSystemConfigMap(ctx, "invite")
	if err != nil {
		return nil, err
	}

	result := &runtimeInviteConfig{}
	if r.config != nil && r.config.Invite != nil {
		result.ForcedInvite = r.config.Invite.ForcedInvite
		result.OnlyFirstPurchase = r.config.Invite.OnlyFirstPurchase
	}
	result.ForcedInvite = getBoolConfig(configs, result.ForcedInvite, "ForcedInvite", "forced_invite")
	result.OnlyFirstPurchase = getBoolConfig(configs, result.OnlyFirstPurchase, "OnlyFirstPurchase", "only_first_purchase")
	return result, nil
}

func (r *authRepo) getSystemConfigMap(ctx context.Context, category string) (map[string]string, error) {
	cacheKey := systemCategoryCacheKey(category)
	if cacheKey != "" {
		if cached, err := r.data.rdb.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
			var result map[string]string
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				return result, nil
			}
		}
	}

	result, err := loadSystemConfigMap(ctx, r.data.db, category)
	if err != nil {
		r.log.Errorw("getSystemConfigMap query failed", "error", err, "category", category)
		return nil, responsecode.NewDatabaseQueryError()
	}

	if cacheKey != "" {
		if payload, err := json.Marshal(result); err == nil {
			r.data.rdb.Set(ctx, cacheKey, payload, 5*time.Minute)
		}
	}

	return result, nil
}

func (r *authRepo) checkRegisterIPLimit(ctx context.Context, cfg *runtimeRegisterConfig, registerIP, authType, account string) (bool, error) {
	if cfg == nil || !cfg.EnableIpRegisterLimit || cfg.IpRegisterLimit <= 0 || cfg.IpRegisterLimitDuration <= 0 || strings.TrimSpace(registerIP) == "" {
		return true, nil
	}

	cacheKey := fmt.Sprintf("%s%s", registerIPKeyPrefix, registerIP)
	now := time.Now().Unix()
	expiredBefore := now - cfg.IpRegisterLimitDuration

	if err := r.data.rdb.ZRemRangeByScore(ctx, cacheKey, "0", strconv.FormatInt(expiredBefore, 10)).Err(); err != nil {
		r.log.Warnw("checkRegisterIPLimit cleanup failed", "error", err, "cache_key", cacheKey)
		return true, nil
	}

	count, err := r.data.rdb.ZCard(ctx, cacheKey).Result()
	if err != nil {
		r.log.Warnw("checkRegisterIPLimit count failed", "error", err, "cache_key", cacheKey)
		return true, nil
	}
	if count >= cfg.IpRegisterLimit {
		return false, nil
	}

	member := fmt.Sprintf("%s:%s:%d", authType, account, now)
	if err := r.data.rdb.ZAdd(ctx, cacheKey, redis.Z{
		Score:  float64(now),
		Member: member,
	}).Err(); err != nil {
		r.log.Warnw("checkRegisterIPLimit zadd failed", "error", err, "cache_key", cacheKey)
		return true, nil
	}
	r.data.rdb.Expire(ctx, cacheKey, time.Duration(cfg.IpRegisterLimitDuration)*time.Second)

	return true, nil
}

func (r *authRepo) createTrialSubscription(ctx context.Context, userID int64, cfg *runtimeRegisterConfig) (*ent.ProxyUserSubscribe, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	userSub, err := r.createTrialSubscriptionTx(ctx, tx, userID, cfg)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return userSub, nil
}

func (r *authRepo) createTrialSubscriptionTx(ctx context.Context, tx *ent.Tx, userID int64, cfg *runtimeRegisterConfig) (*ent.ProxyUserSubscribe, error) {
	if cfg == nil || cfg.TrialSubscribe <= 0 {
		return nil, nil
	}

	subscribeInfo, err := tx.ProxySubscribe.Query().
		Where(proxysubscribe.IDEQ(cfg.TrialSubscribe)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	expireTime := tool.AddTime(cfg.TrialTimeUnit, cfg.TrialTime, startTime)
	token := uuidx.SubscribeToken(fmt.Sprintf("Trial-%v", userID))
	subscribeUUID := uuidx.NewUUID().String()

	return tx.ProxyUserSubscribe.Create().
		SetUserID(userID).
		SetOrderID(0).
		SetSubscribeID(subscribeInfo.ID).
		SetStartTime(startTime).
		SetExpireTime(expireTime).
		SetTraffic(subscribeInfo.Traffic).
		SetDownload(0).
		SetUpload(0).
		SetToken(token).
		SetUUID(subscribeUUID).
		SetStatus(1).
		Save(ctx)
}

func (r *authRepo) clearTrialCaches(ctx context.Context, userSub *ent.ProxyUserSubscribe) {
	if userSub == nil {
		return
	}

	cacheKeys := []string{
		fmt.Sprintf("cache:user:subscribe:user:%d", userSub.UserID),
		fmt.Sprintf("cache:user:subscribe:id:%d", userSub.ID),
	}
	if userSub.Token != nil && *userSub.Token != "" {
		cacheKeys = append(cacheKeys, fmt.Sprintf("cache:user:subscribe:token:%s", *userSub.Token))
	}
	if err := r.data.rdb.Del(ctx, cacheKeys...).Err(); err != nil {
		r.log.Warnw("clearTrialCaches delete user subscribe cache failed", "error", err, "user_subscribe_id", userSub.ID)
	}

	serverCacheKeys := []string{
		fmt.Sprintf("cache:subscribe:id:%d", userSub.SubscribeID),
		fmt.Sprintf("cache:subscribe:servers:%d", userSub.SubscribeID),
	}
	if err := r.data.rdb.Del(ctx, serverCacheKeys...).Err(); err != nil {
		r.log.Warnw("clearTrialCaches delete subscribe cache failed", "error", err, "subscribe_id", userSub.SubscribeID)
	}
	if err := ClearLegacyServerAllCaches(ctx, r.data.rdb); err != nil {
		r.log.Warnw("clearTrialCaches delete legacy server caches failed", "error", err, "subscribe_id", userSub.SubscribeID)
	}
}

func (r *authRepo) triggerGroupRecalculation(ctx context.Context) {
	groupConfig, err := r.getSystemConfigMap(ctx, "group")
	if err != nil {
		r.log.Warnw("triggerGroupRecalculation load config failed", "error", err)
		return
	}

	enabled := getBoolConfig(groupConfig, false, "enabled", "Enabled")
	mode := getStringConfig(groupConfig, "mode", "Mode")
	if !enabled || mode == "" {
		return
	}
	if mode != "average" && mode != "subscribe" && mode != "traffic" {
		return
	}

	go func() {
		groupRepo := &adminGroupRepo{
			data:   r.data,
			logger: r.log,
		}
		backgroundCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := groupRepo.RecalculateGroup(backgroundCtx, mode, "register"); err != nil {
			r.log.Warnw("triggerGroupRecalculation failed", "error", err, "mode", mode)
		}
	}()
}

func (r *authRepo) bindDeviceSafely(ctx context.Context, meta authbiz.RequestMeta, userID int64) {
	if strings.TrimSpace(meta.Identifier) == "" {
		return
	}
	if err := r.bindDeviceToUser(ctx, meta.Identifier, meta.IP, meta.UserAgent, userID); err != nil {
		r.log.Warnw("bindDeviceToUser failed", "error", err, "user_id", userID, "identifier", meta.Identifier)
	}
}

func (r *authRepo) bindDeviceToUser(ctx context.Context, identifier, ip, userAgent string, currentUserID int64) error {
	deviceInfo, err := r.data.db.ProxyUserDevice.Query().
		Where(proxyuserdevice.IdentifierEQ(identifier)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return r.createDeviceForUser(ctx, identifier, ip, userAgent, currentUserID)
		}
		return err
	}

	if deviceInfo.UserID == currentUserID {
		_, err = r.data.db.ProxyUserDevice.UpdateOneID(deviceInfo.ID).
			SetIP(ip).
			SetUserAgent(trimUserAgent(userAgent)).
			SetEnabled(true).
			Save(ctx)
		return err
	}

	return r.rebindDeviceToUser(ctx, deviceInfo, ip, userAgent, currentUserID)
}

func (r *authRepo) createDeviceForUser(ctx context.Context, identifier, ip, userAgent string, userID int64) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	authMethod, err := tx.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthTypeEQ("device"),
			proxyuserauthmethod.AuthIdentifierEQ(identifier),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		tx.Rollback()
		return err
	}
	if ent.IsNotFound(err) {
		_, err = tx.ProxyUserAuthMethod.Create().
			SetUserID(userID).
			SetAuthType("device").
			SetAuthIdentifier(identifier).
			SetVerified(true).
			Save(ctx)
	} else if authMethod.UserID != userID {
		_, err = tx.ProxyUserAuthMethod.UpdateOneID(authMethod.ID).
			SetUserID(userID).
			SetVerified(true).
			Save(ctx)
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ProxyUserDevice.Create().
		SetUserID(userID).
		SetIP(ip).
		SetIdentifier(identifier).
		SetUserAgent(trimUserAgent(userAgent)).
		SetEnabled(true).
		SetOnline(false).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *authRepo) rebindDeviceToUser(ctx context.Context, deviceInfo *ent.ProxyUserDevice, ip, userAgent string, newUserID int64) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	authMethod, err := tx.ProxyUserAuthMethod.Query().
		Where(
			proxyuserauthmethod.AuthTypeEQ("device"),
			proxyuserauthmethod.AuthIdentifierEQ(stringPointerValue(deviceInfo.Identifier)),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		tx.Rollback()
		return err
	}
	if ent.IsNotFound(err) {
		_, err = tx.ProxyUserAuthMethod.Create().
			SetUserID(newUserID).
			SetAuthType("device").
			SetAuthIdentifier(stringPointerValue(deviceInfo.Identifier)).
			SetVerified(true).
			Save(ctx)
	} else {
		_, err = tx.ProxyUserAuthMethod.UpdateOneID(authMethod.ID).
			SetUserID(newUserID).
			SetVerified(true).
			Save(ctx)
	}
	if err != nil {
		tx.Rollback()
		return err
	}

	count, err := tx.ProxyUserAuthMethod.Query().
		Where(proxyuserauthmethod.UserIDEQ(deviceInfo.UserID)).
		Count(ctx)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count < 1 {
		if err := r.transferUserSubscriptions(ctx, tx, deviceInfo.UserID, newUserID); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ProxyUser.UpdateOneID(deviceInfo.UserID).
			SetEnable(false).
			SetDeletedAt(time.Now()).
			SetIsDel(0).
			Save(ctx); err != nil {
			tx.Rollback()
			return err
		}
	}

	if _, err := tx.ProxyUserDevice.UpdateOneID(deviceInfo.ID).
		SetUserID(newUserID).
		SetIP(ip).
		SetUserAgent(trimUserAgent(userAgent)).
		SetEnabled(true).
		Save(ctx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *authRepo) transferUserSubscriptions(ctx context.Context, tx *ent.Tx, oldUserID, newUserID int64) error {
	oldSubs, err := tx.ProxyUserSubscribe.Query().
		Where(
			proxyusersubscribe.UserIDEQ(oldUserID),
			proxyusersubscribe.StatusIn(0, 1),
		).
		All(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, oldSub := range oldSubs {
		newSub, err := tx.ProxyUserSubscribe.Query().
			Where(
				proxyusersubscribe.UserIDEQ(newUserID),
				proxyusersubscribe.SubscribeIDEQ(oldSub.SubscribeID),
				proxyusersubscribe.StatusIn(0, 1),
			).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				return err
			}
			if _, err := tx.ProxyUserSubscribe.UpdateOneID(oldSub.ID).SetUserID(newUserID).Save(ctx); err != nil {
				return err
			}
			continue
		}

		if oldSub.ExpireTime != nil && oldSub.ExpireTime.After(now) {
			remaining := oldSub.ExpireTime.Sub(now)
			newExpire := now.Add(remaining)
			if newSub.ExpireTime != nil && newSub.ExpireTime.After(now) {
				newExpire = newSub.ExpireTime.Add(remaining)
			}
			if _, err := tx.ProxyUserSubscribe.UpdateOneID(newSub.ID).SetExpireTime(newExpire).Save(ctx); err != nil {
				return err
			}
		}

		if err := tx.ProxyUserSubscribe.DeleteOneID(oldSub.ID).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *authRepo) mobileEnabled() bool {
	return r.config != nil && r.config.Mobile != nil && r.config.Mobile.Enable
}

func (r *authRepo) emailVerificationEnabled() bool {
	return r.config != nil && r.config.Email != nil && r.config.Email.EnableVerify
}

func (r *authRepo) logLogin(ctx context.Context, userID int, method, ip, userAgent string, success bool) {
	loginLog := LoginLog{
		Method:    method,
		LoginIP:   ip,
		UserAgent: userAgent,
		Success:   success,
		Timestamp: time.Now().UnixMilli(),
	}

	content, err := json.Marshal(loginLog)
	if err != nil {
		r.log.Errorw("logLogin marshal failed", "error", err, "user_id", userID)
		return
	}

	if _, err := r.data.db.ProxySystemLog.Create().
		SetType(LogTypeLogin).
		SetDate(time.Now().Format("2006-01-02")).
		SetObjectID(int64(userID)).
		SetContent(string(content)).
		Save(ctx); err != nil {
		r.log.Errorw("logLogin save failed", "error", err, "user_id", userID)
	}
}

func (r *authRepo) logRegister(ctx context.Context, userID int, authMethod, identifier, ip, userAgent string) {
	registerLog := RegisterLog{
		AuthMethod: authMethod,
		Identifier: identifier,
		RegisterIP: ip,
		UserAgent:  userAgent,
		Timestamp:  time.Now().UnixMilli(),
	}

	content, err := json.Marshal(registerLog)
	if err != nil {
		r.log.Errorw("logRegister marshal failed", "error", err, "user_id", userID)
		return
	}

	if _, err := r.data.db.ProxySystemLog.Create().
		SetType(LogTypeRegister).
		SetDate(time.Now().Format("2006-01-02")).
		SetObjectID(int64(userID)).
		SetContent(string(content)).
		Save(ctx); err != nil {
		r.log.Errorw("logRegister save failed", "error", err, "user_id", userID)
	}
}

func ensureEmailLoginAllowed(userInfo *ent.ProxyUser) error {
	return ensureUserActive(userInfo)
}

func isDeletedUser(userInfo *ent.ProxyUser) bool {
	if userInfo == nil {
		return true
	}
	if userInfo.DeletedAt != nil {
		return true
	}
	return userInfo.IsDel != nil && *userInfo.IsDel == 0
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func trimUserAgent(userAgent string) string {
	if len(userAgent) <= 64 {
		return userAgent
	}
	return userAgent[:64]
}

func systemCategoryCacheKey(category string) string {
	switch category {
	case "invite":
		return InviteConfigKey
	case "register":
		return RegisterConfigKey
	case "verify":
		return VerifyConfigKey
	default:
		return ""
	}
}

func getStringConfig(configs map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := configs[key]; ok && value != "" {
			return value
		}
	}
	return ""
}

func getStringConfigWithDefault(configs map[string]string, defaultValue string, keys ...string) string {
	if value := getStringConfig(configs, keys...); value != "" {
		return value
	}
	return defaultValue
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func getBoolConfig(configs map[string]string, defaultValue bool, keys ...string) bool {
	for _, key := range keys {
		if value, ok := configs[key]; ok {
			if parsed, err := strconv.ParseBool(value); err == nil {
				return parsed
			}
		}
	}
	return defaultValue
}

func getInt64Config(configs map[string]string, defaultValue int64, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := configs[key]; ok {
			if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
				return parsed
			}
		}
	}
	return defaultValue
}
