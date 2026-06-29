package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/hibiken/asynq"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/internal/conf"
	"github.com/npanel-dev/NPanel-backend/internal/data"
	serverservice "github.com/npanel-dev/NPanel-backend/internal/service/server"
	"github.com/npanel-dev/NPanel-backend/pkg/httpform"
	"github.com/redis/go-redis/v9"
)

type compatLegacyServerCommon struct {
	Protocol  string `form:"protocol" json:"protocol"`
	ServerID  int64  `form:"server_id" json:"server_id"`
	Port      uint16 `form:"port" json:"port"`
	SecretKey string `form:"secret_key" json:"secret_key"`
}

func (c *compatLegacyServerCommon) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Protocol = compatLegacyStringField(raw, "protocol")
	c.ServerID = compatLegacyInt64Field(raw, "server_id", "serverId")
	if port := compatLegacyInt64Field(raw, "port"); port > 0 && port <= 65535 {
		c.Port = uint16(port)
	}
	c.SecretKey = compatLegacyStringField(raw, "secret_key", "secretKey")
	return nil
}

type compatLegacyGetServerConfigRequest struct{ compatLegacyServerCommon }
type compatLegacyGetServerUserListRequest struct{ compatLegacyServerCommon }

type compatLegacyUserTraffic struct {
	SID      int64 `json:"uid"`
	Upload   int64 `json:"upload"`
	Download int64 `json:"download"`
}

func (t *compatLegacyUserTraffic) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	t.SID = compatLegacyInt64Field(raw, "uid", "sid")
	t.Upload = compatLegacyInt64Field(raw, "upload")
	t.Download = compatLegacyInt64Field(raw, "download")
	return nil
}

type compatLegacyPushUserTrafficRequest struct {
	compatLegacyServerCommon
	Traffic []compatLegacyUserTraffic `json:"traffic"`
}

func (r *compatLegacyPushUserTrafficRequest) UnmarshalJSON(data []byte) error {
	type payload struct {
		Traffic []compatLegacyUserTraffic `json:"traffic"`
	}
	var body payload
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	if err := r.compatLegacyServerCommon.UnmarshalJSON(data); err != nil {
		return err
	}
	r.Traffic = body.Traffic
	return nil
}

type compatLegacyOnlineUser struct {
	SID int64  `json:"UID"`
	IP  string `json:"IP"`
}

func (u *compatLegacyOnlineUser) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	u.SID = compatLegacyInt64Field(raw, "uid", "sid")
	u.IP = compatLegacyStringField(raw, "ip")
	return nil
}

type compatLegacyPushOnlineUsersRequest struct {
	compatLegacyServerCommon
	Users []compatLegacyOnlineUser `json:"users"`
}

func (r *compatLegacyPushOnlineUsersRequest) UnmarshalJSON(data []byte) error {
	type payload struct {
		Users []compatLegacyOnlineUser `json:"users"`
	}
	var body payload
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	if err := r.compatLegacyServerCommon.UnmarshalJSON(data); err != nil {
		return err
	}
	r.Users = body.Users
	return nil
}

type compatLegacyPushServerStatusRequest struct {
	compatLegacyServerCommon
	CPU       float64 `json:"cpu"`
	Mem       float64 `json:"mem"`
	Disk      float64 `json:"disk"`
	UpdatedAt int64   `json:"updated_at"`
}

func (r *compatLegacyPushServerStatusRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := r.compatLegacyServerCommon.UnmarshalJSON(data); err != nil {
		return err
	}
	r.CPU = compatLegacyFloat64Field(raw, "cpu")
	r.Mem = compatLegacyFloat64Field(raw, "mem")
	r.Disk = compatLegacyFloat64Field(raw, "disk")
	r.UpdatedAt = compatLegacyInt64Field(raw, "updated_at", "updatedAt")
	return nil
}

type compatLegacyQueryServerConfigRequest struct {
	ServerID  int64
	SecretKey string   `form:"secret_key"`
	Protocols []string `form:"protocols,omitempty"`
}

func compatLegacyStringField(raw map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed
			}
		default:
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", typed)); trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return ""
}

func compatLegacyInt64Field(raw map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int64(typed)
		case json.Number:
			if parsed, err := typed.Int64(); err == nil {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil {
				return parsed
			}
		default:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%v", typed)), 10, 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func compatLegacyFloat64Field(raw map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case json.Number:
			if parsed, err := typed.Float64(); err == nil {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
				return parsed
			}
		default:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprintf("%v", typed)), 64); err == nil {
				return parsed
			}
		}
	}
	return 0
}

type compatLegacyCodeError struct {
	code int
	msg  string
}

func (e *compatLegacyCodeError) Error() string { return e.msg }

type compatLegacyServerProvider struct {
	dataLayer *data.Data
}

func (p *compatLegacyServerProvider) DB() *ent.Client {
	if p == nil || p.dataLayer == nil {
		return nil
	}
	return p.dataLayer.DB()
}

func (p *compatLegacyServerProvider) Redis() redis.UniversalClient {
	if p == nil || p.dataLayer == nil {
		return nil
	}
	return p.dataLayer.Redis()
}

func (p *compatLegacyServerProvider) Queue() *asynq.Client {
	if p == nil || p.dataLayer == nil {
		return nil
	}
	return p.dataLayer.Queue()
}

func (p *compatLegacyServerProvider) AppNodeConfig() *conf.Node {
	if p == nil || p.dataLayer == nil {
		return nil
	}
	if appConf := p.dataLayer.AppConf(); appConf != nil {
		return appConf.Node
	}
	return nil
}

func (p *compatLegacyServerProvider) LoadNodeConfig(ctx context.Context, module string) (*serverservice.CompatLegacyNodeConfig, error) {
	if p == nil || p.dataLayer == nil {
		return nil, errors.New("data layer unavailable")
	}
	nodeConfig, err := data.LoadNodeConfigForServer(ctx, p.dataLayer, log.With(log.DefaultLogger, "module", module))
	if err != nil {
		return nil, err
	}
	return &serverservice.CompatLegacyNodeConfig{
		NodeSecret:             nodeConfig.NodeSecret,
		NodePullInterval:       int64(nodeConfig.NodePullInterval),
		NodePushInterval:       int64(nodeConfig.NodePushInterval),
		TrafficReportThreshold: int64(nodeConfig.TrafficReportThreshold),
		IPStrategy:             nodeConfig.IPStrategy,
		DNS:                    nodeConfig.DNS,
		Block:                  nodeConfig.Block,
		Outbound:               nodeConfig.Outbound,
	}, nil
}

func registerLegacyServerCompatRoutes(r *khttp.Router, dataLayer *data.Data, serverService *serverservice.ServerService) {
	if r == nil || dataLayer == nil || dataLayer.DB() == nil || serverService == nil {
		return
	}
	provider := &compatLegacyServerProvider{dataLayer: dataLayer}

	r.GET("/v1/server/config", func(ctx khttp.Context) error {
		var req compatLegacyGetServerConfigRequest
		_ = ctx.Bind(&req)
		_ = ctx.BindQuery(&req)
		compatLegacyPopulateV1ServerCommon(ctx.Request(), &req.compatLegacyServerCommon)
		if !serverService.CompatV1ServerSecretAllowed(ctx, provider, req.SecretKey) {
			return ctx.String(http.StatusForbidden, "Forbidden")
		}
		out, etag, notModified, err := serverService.CompatGetServerConfig(ctx, provider, &serverservice.CompatLegacyGetServerConfigRequest{
			CompatLegacyServerCommon: serverservice.CompatLegacyServerCommon{
				Protocol:  req.Protocol,
				ServerID:  req.ServerID,
				Port:      req.Port,
				SecretKey: req.SecretKey,
			},
		}, compatLegacyIfNoneMatch(ctx))
		if etag != "" {
			compatLegacySetReplyHeader(ctx, "ETag", etag)
		}
		if notModified {
			return ctx.String(http.StatusNotModified, "Not Modified")
		}
		if err != nil {
			return ctx.String(http.StatusNotFound, "Not Found")
		}
		return ctx.JSON(http.StatusOK, out)
	})

	r.GET("/v1/server/user", func(ctx khttp.Context) error {
		var req compatLegacyGetServerUserListRequest
		_ = ctx.Bind(&req)
		_ = ctx.BindQuery(&req)
		compatLegacyPopulateV1ServerCommon(ctx.Request(), &req.compatLegacyServerCommon)
		if !serverService.CompatV1ServerSecretAllowed(ctx, provider, req.SecretKey) {
			return ctx.String(http.StatusForbidden, "Forbidden")
		}
		out, etag, notModified, err := serverService.CompatGetServerUserList(ctx, provider, &serverservice.CompatLegacyGetServerUserListRequest{
			CompatLegacyServerCommon: serverservice.CompatLegacyServerCommon{
				Protocol:  req.Protocol,
				ServerID:  req.ServerID,
				Port:      req.Port,
				SecretKey: req.SecretKey,
			},
		}, compatLegacyIfNoneMatch(ctx))
		if etag != "" {
			compatLegacySetReplyHeader(ctx, "ETag", etag)
		}
		if notModified {
			return ctx.String(http.StatusNotModified, "Not Modified")
		}
		if err != nil {
			return ctx.String(http.StatusNotFound, "Not Found")
		}
		return ctx.JSON(http.StatusOK, out)
	})

	r.POST("/v1/server/push", func(ctx khttp.Context) error {
		var req compatLegacyPushUserTrafficRequest
		_ = ctx.Bind(&req)
		_ = ctx.BindQuery(&req)
		if !serverService.CompatV1ServerSecretAllowed(ctx, provider, req.SecretKey) {
			return ctx.String(http.StatusForbidden, "Forbidden")
		}
		traffic := make([]serverservice.CompatLegacyUserTraffic, 0, len(req.Traffic))
		for _, item := range req.Traffic {
			traffic = append(traffic, serverservice.CompatLegacyUserTraffic{
				SID:      item.SID,
				Upload:   item.Upload,
				Download: item.Download,
			})
		}
		_, err := compatMiddleware(ctx, &req, func(inner context.Context, request interface{}) (interface{}, error) {
			return nil, serverService.CompatPushUserTraffic(inner, provider, &serverservice.CompatLegacyPushUserTrafficRequest{
				CompatLegacyServerCommon: serverservice.CompatLegacyServerCommon{
					Protocol:  req.Protocol,
					ServerID:  req.ServerID,
					Port:      req.Port,
					SecretKey: req.SecretKey,
				},
				Traffic: traffic,
			})
		})
		if err != nil {
			return compatLegacyServerJSONError(ctx, err)
		}
		return compatLegacyServerJSON(ctx, nil)
	})

	r.POST("/v1/server/status", func(ctx khttp.Context) error {
		var req compatLegacyPushServerStatusRequest
		_ = ctx.Bind(&req)
		_ = ctx.BindQuery(&req.compatLegacyServerCommon)
		if !serverService.CompatV1ServerSecretAllowed(ctx, provider, req.SecretKey) {
			return ctx.String(http.StatusForbidden, "Forbidden")
		}
		_, err := compatMiddleware(ctx, &req, func(inner context.Context, request interface{}) (interface{}, error) {
			return nil, serverService.CompatPushServerStatus(inner, provider, &serverservice.CompatLegacyPushServerStatusRequest{
				CompatLegacyServerCommon: serverservice.CompatLegacyServerCommon{
					Protocol:  req.Protocol,
					ServerID:  req.ServerID,
					Port:      req.Port,
					SecretKey: req.SecretKey,
				},
				CPU:       req.CPU,
				Mem:       req.Mem,
				Disk:      req.Disk,
				UpdatedAt: req.UpdatedAt,
			})
		})
		if err != nil {
			return compatLegacyServerJSONError(ctx, err)
		}
		return compatLegacyServerJSON(ctx, nil)
	})

	r.POST("/v1/server/online", func(ctx khttp.Context) error {
		var req compatLegacyPushOnlineUsersRequest
		if err := ctx.BindQuery(&req.compatLegacyServerCommon); err != nil {
			return compatLegacyServerJSONError(ctx, err)
		}
		compatLegacyPopulateV1ServerCommon(ctx.Request(), &req.compatLegacyServerCommon)
		var body struct {
			Users []compatLegacyOnlineUser `json:"users"`
		}
		if err := json.NewDecoder(ctx.Request().Body).Decode(&body); err != nil {
			return compatLegacyServerJSONError(ctx, err)
		}
		req.Users = body.Users
		if !serverService.CompatV1ServerSecretAllowed(ctx, provider, req.SecretKey) {
			return ctx.String(http.StatusForbidden, "Forbidden")
		}
		users := make([]serverservice.CompatLegacyOnlineUser, 0, len(req.Users))
		for _, item := range req.Users {
			users = append(users, serverservice.CompatLegacyOnlineUser{
				SID: item.SID,
				IP:  item.IP,
			})
		}
		_, err := compatMiddleware(ctx, &req, func(inner context.Context, request interface{}) (interface{}, error) {
			return nil, serverService.CompatPushOnlineUsers(inner, provider, &serverservice.CompatLegacyPushOnlineUsersRequest{
				CompatLegacyServerCommon: serverservice.CompatLegacyServerCommon{
					Protocol:  req.Protocol,
					ServerID:  req.ServerID,
					Port:      req.Port,
					SecretKey: req.SecretKey,
				},
				Users: users,
			})
		})
		if err != nil {
			return compatLegacyServerJSONError(ctx, err)
		}
		return compatLegacyServerJSON(ctx, nil)
	})

	r.GET("/v2/server/{server_id}", func(ctx khttp.Context) error {
		helper := log.NewHelper(log.With(log.DefaultLogger, "module", "server/compat/v2"))
		request := ctx.Request()
		helper.Infof(
			"[QueryServerProtocolConfig] request received method=%s path=%s raw_query=%s content_type=%s",
			request.Method,
			request.URL.Path,
			request.URL.RawQuery,
			request.Header.Get("Content-Type"),
		)

		vars := ctx.Vars()
		rawServerID := strings.TrimSpace(vars.Get("server_id"))
		helper.Infof("[QueryServerProtocolConfig] parsing path server_id=%q", rawServerID)
		serverID, err := strconv.ParseInt(rawServerID, 10, 64)
		if err != nil {
			helper.Errorf("[QueryServerProtocolConfig] invalid path server_id=%q err=%v", rawServerID, err)
			return ctx.String(http.StatusBadRequest, "Invalid Params")
		}
		var req compatLegacyQueryServerConfigRequest
		req.ServerID = serverID
		if err := ctx.BindQuery(&req); err != nil {
			helper.Errorf("[QueryServerProtocolConfig] bind query failed server_id=%d err=%v", serverID, err)
			return ctx.String(http.StatusBadRequest, "Invalid Params")
		}
		queryValues := request.URL.Query()
		if strings.TrimSpace(req.SecretKey) == "" {
			req.SecretKey = httpform.FirstNonEmpty(queryValues, "secret_key", "secretKey")
		}
		if len(req.Protocols) == 0 {
			req.Protocols = httpform.StringSlice(queryValues, "protocols", "protocols[]")
		}
		req.Protocols = compatLegacySanitizeStringList(req.Protocols)
		helper.Infof(
			"[QueryServerProtocolConfig] query parsed server_id=%d secret_present=%t secret_value=%q protocols=%v query_keys=%d",
			req.ServerID,
			strings.TrimSpace(req.SecretKey) != "",
			req.SecretKey,
			req.Protocols,
			len(queryValues),
		)
		if formValues, err := httpform.ParseGETBodyForm(ctx.Request()); err != nil {
			helper.Errorf("[QueryServerProtocolConfig] parse GET body form failed server_id=%d err=%v", serverID, err)
			return ctx.String(http.StatusBadRequest, "Invalid Params")
		} else {
			if strings.TrimSpace(req.SecretKey) == "" {
				req.SecretKey = httpform.FirstNonEmpty(formValues, "secret_key", "secretKey")
			}
			if len(req.Protocols) == 0 {
				req.Protocols = httpform.StringSlice(formValues, "protocols", "protocols[]")
			}
			req.Protocols = compatLegacySanitizeStringList(req.Protocols)
			helper.Infof(
				"[QueryServerProtocolConfig] GET body form merged server_id=%d secret_present=%t secret_value=%q protocols=%v form_keys=%d",
				req.ServerID,
				strings.TrimSpace(req.SecretKey) != "",
				req.SecretKey,
				req.Protocols,
				len(formValues),
			)
		}
		if !serverService.CompatV2ServerSecretAllowed(ctx, provider, req.SecretKey) {
			helper.Errorf(
				"[QueryServerProtocolConfig] secret validation failed server_id=%d secret_present=%t secret_value=%q protocols=%v",
				req.ServerID,
				strings.TrimSpace(req.SecretKey) != "",
				req.SecretKey,
				req.Protocols,
			)
			return ctx.String(http.StatusUnauthorized, "Unauthorized")
		}
		helper.Infof(
			"[QueryServerProtocolConfig] secret validated server_id=%d secret_value=%q protocols=%v",
			req.ServerID,
			req.SecretKey,
			req.Protocols,
		)
		out, err := compatMiddleware(ctx, &req, func(inner context.Context, request interface{}) (interface{}, error) {
			helper.Infof(
				"[QueryServerProtocolConfig] invoking compat usecase server_id=%d protocols=%v",
				req.ServerID,
				req.Protocols,
			)
			return serverService.CompatQueryServerProtocolConfig(inner, provider, &serverservice.CompatLegacyQueryServerConfigRequest{
				ServerID:  req.ServerID,
				SecretKey: req.SecretKey,
				Protocols: append([]string(nil), req.Protocols...),
			})
		})
		if err != nil {
			helper.Errorf("[QueryServerProtocolConfig] compat usecase failed server_id=%d err=%v", req.ServerID, err)
			return compatLegacyServerJSONError(ctx, err)
		}
		if typedOut, ok := out.(*serverservice.CompatLegacyQueryServerConfigResponse); ok && typedOut != nil {
			helper.Infof(
				"[QueryServerProtocolConfig] success server_id=%d total=%d dns=%d outbound=%d protocols=%d",
				req.ServerID,
				typedOut.Total,
				len(typedOut.DNS),
				len(typedOut.Outbound),
				len(typedOut.Protocols),
			)
		} else {
			helper.Infof("[QueryServerProtocolConfig] success server_id=%d response_type=%T", req.ServerID, out)
		}
		return compatLegacyServerJSON(ctx, out)
	})
}

func compatLegacyServerJSON(ctx khttp.Context, data interface{}) error {
	return ctx.JSON(http.StatusOK, compatEnvelope{Code: 200, Msg: "success", Data: data})
}

func compatLegacyServerJSONError(ctx khttp.Context, err error) error {
	code := 500
	msg := "Internal Server Error"
	var typedErr *compatLegacyCodeError
	if errors.As(err, &typedErr) {
		code = typedErr.code
		msg = typedErr.msg
	}
	return ctx.JSON(http.StatusOK, compatEnvelope{Code: code, Msg: msg})
}

func compatLegacyPopulateV1ServerCommon(request *http.Request, req *compatLegacyServerCommon) {
	if request == nil || req == nil {
		return
	}
	compatLegacyMergeV1ServerCommon(req, request.URL.Query())
	if formValues, err := httpform.ParseGETBodyForm(request); err == nil {
		compatLegacyMergeV1ServerCommon(req, formValues)
	}
}

func compatLegacyMergeV1ServerCommon(req *compatLegacyServerCommon, values url.Values) {
	if req == nil || values == nil {
		return
	}
	if strings.TrimSpace(req.Protocol) == "" {
		req.Protocol = httpform.FirstNonEmpty(values, "protocol")
	}
	if req.ServerID <= 0 {
		if serverID, ok := compatLegacyInt64FromValues(values, "server_id", "serverId"); ok {
			req.ServerID = serverID
		}
	}
	if req.Port == 0 {
		if port, ok := compatLegacyInt64FromValues(values, "port"); ok && port > 0 && port <= 65535 {
			req.Port = uint16(port)
		}
	}
	if strings.TrimSpace(req.SecretKey) == "" {
		req.SecretKey = httpform.FirstNonEmpty(values, "secret_key", "secretKey")
	}
}

func compatLegacyInt64FromValues(values url.Values, keys ...string) (int64, bool) {
	raw := httpform.FirstNonEmpty(values, keys...)
	if raw == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func compatLegacyIfNoneMatch(ctx context.Context) string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		return strings.TrimSpace(tr.RequestHeader().Get("If-None-Match"))
	}
	return ""
}

func compatLegacySetReplyHeader(ctx context.Context, key, value string) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		tr.ReplyHeader().Set(key, value)
	}
}

func compatLegacySanitizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
