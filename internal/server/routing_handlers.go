package server

import (
	"encoding/json"
	nethttp "net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/transport/http"
	publicrouting "github.com/npanel-dev/NPanel-backend/internal/biz/public/routing"
	"github.com/npanel-dev/NPanel-backend/internal/pkg/middleware"
	adminroutingservice "github.com/npanel-dev/NPanel-backend/internal/service/admin/routing"
)

type publicRoutingCache struct {
	mu        sync.Mutex
	envelope  publicrouting.Envelope
	expiresAt time.Time
	ok        bool
}

func registerRoutingPreviewRoutes(srv *http.Server, routing *adminroutingservice.RoutingService) {
	cache := &publicRoutingCache{}
	srv.HandleFunc("/v1/public/routing/config", handleRoutingConfig(routing, cache))
	srv.HandleFunc("/v1/public/routing/preview", handleRoutingPreview(routing, cache))
}

func handleRoutingConfig(routing *adminroutingservice.RoutingService, cache *publicRoutingCache) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			writeRoutingError(w, nethttp.StatusMethodNotAllowed, 405, "method not allowed")
			return
		}

		cfg, fallback := loadPublicRoutingConfig(routing, cache, r)
		if fallback != "" {
			w.Header().Set("X-Routing-Fallback", fallback)
		}

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

		cfg, fallback := loadPublicRoutingConfig(routing, cache, r)
		if fallback != "" {
			w.Header().Set("X-Routing-Fallback", fallback)
		}
		result := publicrouting.PreviewRouteConfig(cfg, req)
		writeRoutingOK(w, result)
	}
}

func loadPublicRoutingConfig(routing *adminroutingservice.RoutingService, cache *publicRoutingCache, r *nethttp.Request) (publicrouting.Envelope, string) {
	now := time.Now()
	if cfg, ok := cache.get(now); ok {
		return cfg, ""
	}

	features := publicrouting.ParseFeatureList(r.Header.Get("X-Routing-Features"))
	cfg, err := routing.BuildPublicConfig(r.Context(), now, publicrouting.ConfigOptions{
		UserID:            middleware.GetUserID(r.Context()),
		UserAgent:         r.UserAgent(),
		SupportedFeatures: features,
	})
	if err != nil {
		return publicrouting.BuildPreviewConfig(now, publicrouting.ConfigOptions{
			UserID:            middleware.GetUserID(r.Context()),
			UserAgent:         r.UserAgent(),
			SupportedFeatures: features,
		}), "fixture"
	}
	cache.set(cfg, now.Add(15*time.Second))
	return cfg, ""
}

func (c *publicRoutingCache) get(now time.Time) (publicrouting.Envelope, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ok || now.After(c.expiresAt) {
		return publicrouting.Envelope{}, false
	}
	return c.envelope, true
}

func (c *publicRoutingCache) set(envelope publicrouting.Envelope, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.envelope = envelope
	c.expiresAt = expiresAt
	c.ok = true
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
