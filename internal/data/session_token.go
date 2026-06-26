package data

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyuser"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	"github.com/npanel-dev/NPanel-backend/pkg/constant"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
	"github.com/npanel-dev/NPanel-backend/pkg/uuidx"
)

type sessionTokenOptions struct {
	Identifier string
	LoginType  string
}

func (d *Data) jwtSecret() string {
	if d != nil && d.serverConf != nil && d.serverConf.Auth != nil {
		if secret := strings.TrimSpace(d.serverConf.Auth.JwtSecret); secret != "" {
			return secret
		}
	}

	if secret := strings.TrimSpace(os.Getenv("JWT_SECRET")); secret != "" {
		return secret
	}

	return DefaultJWTSecret
}

func (d *Data) jwtExpireSeconds() int64 {
	expire := int64(DefaultJWTExpire)
	if expireStr := strings.TrimSpace(os.Getenv("JWT_EXPIRE")); expireStr != "" {
		if parsed, err := strconv.ParseInt(expireStr, 10, 64); err == nil {
			expire = parsed
		}
	}
	return expire
}

func (d *Data) sessionCacheKey(sessionID string) string {
	return fmt.Sprintf("%s:%s", constant.SessionIdKey, sessionID)
}

func (d *Data) issueSessionTokenWithSessionID(ctx context.Context, userID int64, opts sessionTokenOptions) (string, string, error) {
	if err := d.ensureUserCanUseSession(ctx, userID); err != nil {
		return "", "", err
	}

	sessionID := uuidx.NewUUID().String()
	expire := d.jwtExpireSeconds()
	claims := map[string]interface{}{
		"UserId":     userID,
		"user_id":    userID,
		"SessionId":  sessionID,
		"session_id": sessionID,
	}

	if opts.Identifier != "" {
		claims["identifier"] = opts.Identifier
	}
	if opts.LoginType != "" {
		claims["CtxLoginType"] = opts.LoginType
		claims["LoginType"] = opts.LoginType
	}

	token, err := tool.GenerateJWT(d.jwtSecret(), expire, claims)
	if err != nil {
		return "", "", err
	}
	if err := d.rdb.Set(ctx, d.sessionCacheKey(sessionID), userID, time.Duration(expire)*time.Second).Err(); err != nil {
		return "", "", err
	}

	return token, sessionID, nil
}

func (d *Data) issueSessionToken(ctx context.Context, userID int64, opts sessionTokenOptions) (string, error) {
	token, _, err := d.issueSessionTokenWithSessionID(ctx, userID, opts)
	return token, err
}

func (d *Data) ensureUserCanUseSession(ctx context.Context, userID int64) error {
	userInfo, err := d.db.ProxyUser.Query().
		Where(proxyuser.IDEQ(userID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return responsecode.NewKratosError(responsecode.ErrUserNotFound)
		}
		return err
	}
	return ensureUserActive(userInfo)
}

func ensureUserActive(userInfo *ent.ProxyUser) error {
	if userInfo == nil || isDeletedUser(userInfo) {
		return responsecode.NewKratosError(responsecode.ErrUserNotFound)
	}
	if !userInfo.Enable {
		return responsecode.NewKratosError(responsecode.ErrAccountDisabled)
	}
	return nil
}
