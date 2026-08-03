package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/transport/http"
	publicrouting "github.com/npanel-dev/NPanel-backend/internal/biz/public/routing"
	appmiddleware "github.com/npanel-dev/NPanel-backend/internal/middleware"
	"github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	adminroutingservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/routing"
)

type publicRoutingCache struct {
	mu    sync.Mutex
	items map[string]publicRoutingCacheItem
}

type publicRoutingCacheItem struct {
	envelope  publicrouting.Envelope
	expiresAt time.Time
}

func registerRoutingPreviewRoutes(srv *http.Server, routing *adminroutingservice.RoutingService, authMiddleware *appmiddleware.ServiceContext) {
	cache := &publicRoutingCache{}
	srv.HandleFunc("/v1/public/routing/config", requireRoutingAuthentication(authMiddleware, handleRoutingConfig(routing, cache)))
	srv.HandleFunc("/v1/public/routing/preview", requireRoutingAuthentication(authMiddleware, handleRoutingPreview(routing, cache)))
	srv.HandleFunc("/v1/public/routing/health_report", requireRoutingAuthentication(authMiddleware, handleRoutingHealthReport(routing)))
	srv.HandleFunc("/v1/public/routing/route_event", requireRoutingAuthentication(authMiddleware, handleRoutingRouteEvent(routing)))
}

func handleRoutingConfig(routing *adminroutingservice.RoutingService, cache *publicRoutingCache) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			writeRoutingError(w, nethttp.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}

		cfg, fallback, unsupported := loadPublicRoutingConfig(routing, cache, r, publicrouting.PreviewRequest{})
		if fallback != "" {
			w.Header().Set("X-Routing-Fallback", fallback)
		}
		writeRoutingCapabilityHeaders(w, unsupported)

		if r.Header.Get("If-None-Match") == cfg.RoutingHash {
			w.WriteHeader(nethttp.StatusNotModified)
			return
		}

		writeRoutingOK(w, cfg)
	}
}

func handleRoutingPreview(routing *adminroutingservice.RoutingService, cache *publicRoutingCache) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			writeRoutingError(w, nethttp.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}

		var req publicrouting.PreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeRoutingError(w, nethttp.StatusBadRequest, 400, "invalid preview request")
			return
		}
		if len(req.SupportedFeatures) == 0 {
			req.SupportedFeatures = publicrouting.ParseFeatureList(r.Header.Get("X-Routing-Features"))
		}
		req.Domain = strings.TrimSpace(req.Domain)
		if req.Domain == "" && req.IP == "" {
			writeRoutingError(w, nethttp.StatusBadRequest, 422, "domain or ip is required")
			return
		}

		cfg, fallback, unsupported := loadPublicRoutingConfig(routing, cache, r, req)
		if fallback != "" {
			w.Header().Set("X-Routing-Fallback", fallback)
		}
		writeRoutingCapabilityHeaders(w, unsupported)
		result := publicrouting.PreviewRouteConfig(cfg, req)
		writeRoutingOK(w, result)
	}
}

func handleRoutingHealthReport(routing *adminroutingservice.RoutingService) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			writeRoutingError(w, nethttp.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}

		var req publicrouting.HealthReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeRoutingError(w, nethttp.StatusBadRequest, 400, "invalid health report request")
			return
		}
		if req.ReporterID == "" {
			req.ReporterID = firstNonEmptyString(r.Header.Get("X-Device-ID"), r.Header.Get("X-Reporter-ID"), r.RemoteAddr)
		}
		if req.ReporterType == "" {
			req.ReporterType = firstNonEmptyString(r.Header.Get("X-Reporter-Type"), "client")
		}
		if err := routing.RecordHealthReport(r.Context(), req); err != nil {
			writeRoutingError(w, nethttp.StatusBadRequest, 400, "invalid health report request")
			return
		}
		writeRoutingOK(w, map[string]bool{"accepted": true})
	}
}

func handleRoutingRouteEvent(routing *adminroutingservice.RoutingService) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			writeRoutingError(w, nethttp.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}

		var req publicrouting.RouteEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeRoutingError(w, nethttp.StatusBadRequest, 400, "invalid route event request")
			return
		}
		if req.ReporterID == "" {
			req.ReporterID = firstNonEmptyString(r.Header.Get("X-Device-ID"), r.Header.Get("X-Reporter-ID"), r.RemoteAddr)
		}
		if req.ReporterType == "" {
			req.ReporterType = firstNonEmptyString(r.Header.Get("X-Reporter-Type"), "client")
		}
		if err := routing.RecordRouteEvent(r.Context(), req); err != nil {
			writeRoutingError(w, nethttp.StatusBadRequest, 400, "invalid route event request")
			return
		}
		writeRoutingOK(w, map[string]bool{"accepted": true})
	}
}

func loadPublicRoutingConfig(routing *adminroutingservice.RoutingService, cache *publicRoutingCache, r *nethttp.Request, req publicrouting.PreviewRequest) (publicrouting.Envelope, string, []string) {
	now := time.Now()
	scopeOpts := routingConfigOptionsFromRequest(r, req)
	features := publicrouting.ParseFeatureList(r.Header.Get("X-Routing-Features"))
	if len(scopeOpts.SupportedFeatures) == 0 {
		scopeOpts.SupportedFeatures = features
	}
	cacheKey := publicRoutingCacheKey(scopeOpts)
	if cfg, ok := cache.get(cacheKey, now); ok {
		return cfg, "", publicrouting.MissingRequiredFeatures(cfg.CapabilityRequirements.RequiredFeatures, scopeOpts.SupportedFeatures)
	}

	cfg, err := routing.BuildPublicConfig(r.Context(), now, publicrouting.ConfigOptions{
		UserID:            scopeOpts.UserID,
		SubscribeID:       scopeOpts.SubscribeID,
		UserSubscribeID:   scopeOpts.UserSubscribeID,
		SubscribeToken:    scopeOpts.SubscribeToken,
		NodeID:            scopeOpts.NodeID,
		UserAgent:         r.UserAgent(),
		SupportedFeatures: scopeOpts.SupportedFeatures,
	})
	if err != nil {
		fallback := publicrouting.ApplyClientCapabilities(publicrouting.BuildPreviewConfig(now, publicrouting.ConfigOptions{
			UserID:            scopeOpts.UserID,
			SubscribeID:       scopeOpts.SubscribeID,
			UserSubscribeID:   scopeOpts.UserSubscribeID,
			SubscribeToken:    scopeOpts.SubscribeToken,
			NodeID:            scopeOpts.NodeID,
			UserAgent:         r.UserAgent(),
			SupportedFeatures: scopeOpts.SupportedFeatures,
		}), scopeOpts.SupportedFeatures)
		return fallback, "fixture", publicrouting.MissingRequiredFeatures(fallback.CapabilityRequirements.RequiredFeatures, scopeOpts.SupportedFeatures)
	}
	cache.set(cacheKey, cfg, now.Add(15*time.Second))
	return cfg, "", publicrouting.MissingRequiredFeatures(cfg.CapabilityRequirements.RequiredFeatures, scopeOpts.SupportedFeatures)
}

func routingConfigOptionsFromRequest(r *nethttp.Request, req publicrouting.PreviewRequest) publicrouting.ConfigOptions {
	token := firstNonEmptyString(r.Header.Get("token"), r.Header.Get("X-Subscribe-Token"))
	return publicrouting.ConfigOptions{
		UserID:            middleware.GetUserID(r.Context()),
		SubscribeToken:    token,
		SupportedFeatures: req.SupportedFeatures,
	}
}

func publicRoutingCacheKey(opts publicrouting.ConfigOptions) string {
	tokenHash := sha256.Sum256([]byte(opts.SubscribeToken))
	return fmt.Sprintf("u:%d|s:%d|us:%d|n:%d|t:%x|f:%s", opts.UserID, opts.SubscribeID, opts.UserSubscribeID, opts.NodeID, tokenHash[:8], strings.Join(opts.SupportedFeatures, ","))
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeRoutingCapabilityHeaders(w nethttp.ResponseWriter, unsupported []string) {
	if len(unsupported) == 0 {
		w.Header().Set("X-Routing-Execution", "eligible")
		return
	}
	w.Header().Set("X-Routing-Execution", "blocked")
	w.Header().Set("X-Routing-Unsupported-Features", strings.Join(unsupported, ","))
}

func requireRoutingAuthentication(authMiddleware *appmiddleware.ServiceContext, next nethttp.HandlerFunc) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		authenticatedRequest, ok := authenticateRoutingRequest(authMiddleware, r)
		if !ok {
			writeRoutingError(w, nethttp.StatusUnauthorized, 401, "routing authentication required")
			return
		}
		next(w, authenticatedRequest)
	}
}

// authenticateRoutingRequest supports subscription-token clients and the
// OwlClient account JWT carried in Authorization. These routes are registered
// with HandleFunc, so the shared Kratos middleware must be invoked explicitly.
func authenticateRoutingRequest(authMiddleware *appmiddleware.ServiceContext, r *nethttp.Request) (*nethttp.Request, bool) {
	if firstNonEmptyString(r.Header.Get("token"), r.Header.Get("X-Subscribe-Token")) != "" {
		return r, true
	}
	if authMiddleware == nil {
		return r, false
	}

	var authenticatedContext context.Context
	authenticate := authMiddleware.Auth()(func(ctx context.Context, _ interface{}) (interface{}, error) {
		authenticatedContext = ctx
		return nil, nil
	})
	if _, err := authenticate(r.Context(), nil); err != nil || middleware.GetUserID(authenticatedContext) <= 0 {
		return r, false
	}
	return r.WithContext(authenticatedContext), true
}

func (c *publicRoutingCache) get(key string, now time.Time) (publicrouting.Envelope, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		return publicrouting.Envelope{}, false
	}
	item, ok := c.items[key]
	if !ok || now.After(item.expiresAt) {
		return publicrouting.Envelope{}, false
	}
	return item.envelope, true
}

func (c *publicRoutingCache) set(key string, envelope publicrouting.Envelope, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil {
		c.items = map[string]publicRoutingCacheItem{}
	}
	c.items[key] = publicRoutingCacheItem{envelope: envelope, expiresAt: expiresAt}
}

func writeRoutingOK(w nethttp.ResponseWriter, data any) {
	writeRoutingJSON(w, nethttp.StatusOK, map[string]any{
		"code":    200,
		"message": "success",
		"data":    data,
	})
}

func writeRoutingError(w nethttp.ResponseWriter, httpStatus, code int, message string) {
	writeRoutingJSON(w, httpStatus, map[string]any{
		"code":    code,
		"message": message,
		"data":    map[string]any{},
	})
}

func writeRoutingJSON(w nethttp.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
