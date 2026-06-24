package subscription

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	v1 "github.com/npanel-dev/NPanel-backend/api/public/subscription/v1"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

var ErrLegacyAccessDenied = errors.New("access denied")

// SubscriptionRepo 订阅配置数据仓库接口
type SubscriptionRepo interface {
	// ValidateTokenAndGetSubscribe 验证token并获取用户订阅信息
	ValidateTokenAndGetSubscribe(ctx context.Context, token string) (*UserSubscribe, error)

	// GetAvailableNodes 获取可用节点列表
	GetAvailableNodes(ctx context.Context, userSubscribe *UserSubscribe) ([]*NodeInfo, error)

	// GetUserInfo 获取用户信息
	GetUserInfo(ctx context.Context, userID int64) (*UserInfo, error)

	// GetSubscribeInfo 获取订阅信息头
	GetSubscribeInfo(ctx context.Context, userSubscribe *UserSubscribe) string

	// UpdateSubscribeLog 更新订阅活动日志
	UpdateSubscribeLog(ctx context.Context, userSubscribe *UserSubscribe, userAgent, clientIP string) error

	// GetSubscribeApplications 获取所有订阅应用配置
	GetSubscribeApplications(ctx context.Context) ([]*SubscribeApplication, error)

	// GetSubscribeDomain 获取订阅域名配置（用于生成订阅URL）
	GetSubscribeDomain(ctx context.Context) string

	// GetSubscribePath 获取订阅路径配置（用于生成订阅URL）
	GetSubscribePath(ctx context.Context) string

	// GetSiteName 获取站点名称
	GetSiteName(ctx context.Context) string

	// GetSubscribeRuntimeConfig 获取订阅运行时配置
	GetSubscribeRuntimeConfig(ctx context.Context) (*SubscribeRuntimeConfig, error)
}

// SubscribeApplication 订阅应用配置
type SubscribeApplication struct {
	ID                int64
	Name              string
	Icon              string
	Description       string
	Scheme            string
	UserAgent         string
	IsDefault         bool
	SubscribeTemplate string
	OutputFormat      string
	DownloadLink      string
}

type SubscribeRuntimeConfig struct {
	PanDomain      bool
	UserAgentLimit bool
	UserAgentList  string
}

// SubscriptionUseCase 订阅配置用例
type SubscriptionUseCase struct {
	repo SubscriptionRepo
}

// NewSubscriptionUseCase 创建订阅配置用例
func NewSubscriptionUseCase(repo SubscriptionRepo) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		repo: repo,
	}
}

func (uc *SubscriptionUseCase) GetSubscribeApplications(ctx context.Context) ([]*SubscribeApplication, error) {
	return uc.repo.GetSubscribeApplications(ctx)
}

func (uc *SubscriptionUseCase) matchSubscribeApplication(userAgent string, clients []*SubscribeApplication) (*SubscribeApplication, error) {
	userAgentLower := strings.ToLower(userAgent)
	var targetApp, defaultApp *SubscribeApplication

	for _, item := range clients {
		ua := strings.ToLower(item.UserAgent)
		if item.IsDefault {
			defaultApp = item
		}

		if strings.Contains(userAgentLower, ua) {
			if strings.Contains(userAgentLower, "stash") && !strings.Contains(ua, "stash") {
				continue
			}
			targetApp = item
			break
		}
	}

	if targetApp == nil {
		if defaultApp == nil {
			return nil, fmt.Errorf("no matching client found")
		}
		targetApp = defaultApp
	}

	return targetApp, nil
}

// getSubscribeV2URL 生成订阅URL - 按照原项目逻辑实现
// requestURI: 请求的URI（例如：/v1/subscribe?token=xxx）
// requestHost: 请求的Host（例如：example.com）
// gatewayMode: 是否为网关模式（如果为true，添加/sub前缀）
func (uc *SubscriptionUseCase) getSubscribeV2URL(ctx context.Context, token, requestURI, requestHost string, gatewayMode bool) string {
	uri := strings.TrimSpace(requestURI)
	if subscribePath := strings.TrimSpace(uc.repo.GetSubscribePath(ctx)); subscribePath != "" {
		if !strings.HasPrefix(subscribePath, "/") {
			subscribePath = "/" + subscribePath
		}
		subscribePath = strings.TrimRight(subscribePath, "/")
		if subscribePath == "" {
			subscribePath = "/api/subscribe"
		}
		if token != "" {
			uri = subscribePath + "/" + token
		} else {
			uri = subscribePath
		}
	}

	if uri == "" {
		uri = "/api/subscribe"
		if token != "" {
			uri += "/" + token
		}
	}

	// 如果是网关模式，添加 /sub 前缀
	if gatewayMode && !strings.HasPrefix(uri, "/sub/") {
		uri = "/sub" + uri
	}

	// 使用自定义域名（如果配置了）
	subscribeDomain := uc.repo.GetSubscribeDomain(ctx)
	if subscribeDomain != "" {
		domains := strings.Split(subscribeDomain, "\n")
		if len(domains) > 0 {
			return fmt.Sprintf("https://%s%s", strings.TrimSpace(domains[0]), uri)
		}
	}

	// 使用当前请求的host
	return fmt.Sprintf("https://%s%s", requestHost, uri)
}

func (uc *SubscriptionUseCase) ValidateLegacyRequest(ctx context.Context, token, requestHost, userAgent string, clients []*SubscribeApplication) error {
	runtimeConfig, err := uc.repo.GetSubscribeRuntimeConfig(ctx)
	if err != nil || runtimeConfig == nil {
		return err
	}

	if runtimeConfig.PanDomain {
		domainArr := strings.Split(requestHost, ".")
		if len(domainArr) > 0 {
			short, err := tool.FixedUniqueString(token, 8, "")
			if err != nil {
				return err
			}
			if !strings.EqualFold(short, domainArr[0]) {
				return ErrLegacyAccessDenied
			}
		}
	}

	if runtimeConfig.UserAgentLimit {
		if strings.TrimSpace(userAgent) == "" {
			return ErrLegacyAccessDenied
		}

		clientUserAgents := tool.RemoveDuplicateElements(strings.Split(runtimeConfig.UserAgentList, "\n")...)
		for _, item := range clients {
			clientUserAgents = append(clientUserAgents, strings.ToLower(strings.TrimSpace(item.UserAgent)))
		}

		allow := false
		for _, keyword := range clientUserAgents {
			keyword = strings.ToLower(strings.TrimSpace(keyword))
			if keyword == "" {
				continue
			}
			if strings.Contains(strings.ToLower(userAgent), keyword) {
				allow = true
			}
		}

		if !allow {
			return ErrLegacyAccessDenied
		}
	}

	return nil
}

func (uc *SubscriptionUseCase) ResolveDownloadMeta(ctx context.Context, _ *v1.GetSubscribeConfigRequest, userAgent string) (string, string, error) {
	clients, err := uc.repo.GetSubscribeApplications(ctx)
	if err != nil {
		return "", "", err
	}

	targetApp, err := uc.matchSubscribeApplication(userAgent, clients)
	if err != nil {
		return "", "", err
	}

	siteName := uc.repo.GetSiteName(ctx)
	switch strings.ToLower(targetApp.OutputFormat) {
	case "json", "yaml", "conf":
		return "application/octet-stream; charset=UTF-8", url.QueryEscape(siteName) + "." + strings.ToLower(targetApp.OutputFormat), nil
	default:
		return "", "", nil
	}
}

// GetSubscribeConfig 获取订阅配置 - 按照原项目逻辑实现
func (uc *SubscriptionUseCase) GetSubscribeConfig(ctx context.Context, req *v1.GetSubscribeConfigRequest, userAgent, clientIP, requestURI, requestHost string, gatewayMode bool, queryParams map[string]string) (*v1.GetSubscribeConfigReply, error) {
	// 按照原项目逻辑，使用defer记录日志，并只在成功时记录
	var subscribeStatus = false
	var userSubscribe *UserSubscribe

	defer func() {
		if subscribeStatus && userSubscribe != nil {
			_ = uc.repo.UpdateSubscribeLog(ctx, userSubscribe, userAgent, clientIP)
		}
	}()

	// 1. 查询客户端应用列表
	clients, err := uc.repo.GetSubscribeApplications(ctx)
	if err != nil {
		return nil, err
	}

	if err := uc.ValidateLegacyRequest(ctx, req.Token, requestHost, userAgent, clients); err != nil {
		return nil, err
	}

	// 2. 根据User-Agent匹配客户端
	targetApp, err := uc.matchSubscribeApplication(userAgent, clients)
	if err != nil {
		return nil, err
	}

	// 3. 验证token并获取用户订阅
	userSubscribe, err = uc.repo.ValidateTokenAndGetSubscribe(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	// 4. 获取用户信息
	userInfo, err := uc.repo.GetUserInfo(ctx, userSubscribe.UserID)
	if err != nil {
		return nil, err
	}

	// 5. 构建请求参数（URL参数）
	params := make(map[string]string, len(req.Params)+len(queryParams)+2)
	for key, value := range req.Params {
		params[key] = value
	}
	for key, value := range queryParams {
		params[key] = value
	}
	if req.Flag != "" {
		params["flag"] = req.Flag
	}
	if req.Type != "" {
		params["type"] = req.Type
	}

	// 6. 获取可用节点列表（包含分组过滤、过期检查等）
	nodes, err := uc.repo.GetAvailableNodes(ctx, userSubscribe)
	if err != nil {
		return nil, err
	}
	nodes = filterSubscriptionNodesByUserAgent(nodes, userAgent)

	// 7. 生成订阅URL（按照原项目逻辑）
	subscribeURL := uc.getSubscribeV2URL(ctx, req.Token, requestURI, requestHost, gatewayMode)

	// 8. 更新用户信息中的订阅URL
	userInfo.SubscribeURL = subscribeURL
	userInfo.Password = userSubscribe.UUID
	userInfo.Download = userSubscribe.Download
	userInfo.Upload = userSubscribe.Upload
	userInfo.Traffic = userSubscribe.Traffic
	userInfo.SubscribeID = userSubscribe.ID
	if userSubscribe.ExpireTime > 0 {
		userInfo.ExpiredAt = time.UnixMilli(userSubscribe.ExpireTime)
	}

	// 9. 获取站点名称
	siteName := uc.repo.GetSiteName(ctx)

	// 10. 生成配置文件（使用模板方式）
	configBytes, err := RenderTemplate(
		targetApp.SubscribeTemplate,
		targetApp.OutputFormat,
		siteName, // 从配置获取站点名称
		userSubscribe.SubscribeName,
		nodes,
		userSubscribe,
		*userInfo, // 传入包含订阅URL的用户信息
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	// 8. 获取订阅信息头
	header := uc.repo.GetSubscribeInfo(ctx, userSubscribe)

	// 10. 标记成功（defer会记录日志）
	subscribeStatus = true

	return &v1.GetSubscribeConfigReply{
		Config: configBytes,
		Header: header,
	}, nil
}

func filterSubscriptionNodesByUserAgent(nodes []*NodeInfo, userAgent string) []*NodeInfo {
	// 默认剔除实验性协议/网络（simnet/omniflow/mx/mc1/mundordp/mundosql）。
	// 仅当请求来自自有客户端/SDK（UA 命中 omnxt 或 slaglab）时才放行。
	// 开源客户端不支持这两个新协议，且必须搭配我方节点列表与 SDK 才能使用。
	if len(nodes) == 0 || isOfficialClient(userAgent) {
		return nodes
	}

	filtered := make([]*NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		if node == nil || isExperimentalSubscriptionProtocol(node.Type) || isExperimentalSubscriptionTransport(node.Transport) {
			continue
		}
		filtered = append(filtered, node)
	}
	return filtered
}

// isOfficialClient 判断请求是否来自自有客户端/SDK。
// 只有命中的客户端才允许下发 simnet/omniflow/mx/mundo 等实验性协议。
func isOfficialClient(userAgent string) bool {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return false
	}

	// 自有客户端/SDK 的 UA 关键字白名单
	officialKeywords := []string{
		"omnxt",
		"slag/",
		"slaglab",
	}
	for _, keyword := range officialKeywords {
		if strings.Contains(ua, keyword) {
			return true
		}
	}
	return false
}

func isExperimentalSubscriptionProtocol(protocolType string) bool {
	switch strings.ToLower(strings.TrimSpace(protocolType)) {
	case "simnet", "omn", "omniflow", "mx":
		return true
	default:
		return false
	}
}

func isExperimentalSubscriptionTransport(transport string) bool {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "mc1", "mundordp", "mundosql":
		return true
	default:
		return false
	}
}

// UserSubscribe 用户订阅信息
type UserSubscribe struct {
	ID          int64
	UserID      int64
	SubscribeID int64
	Token       string
	UUID        string
	StartTime   int64
	ExpireTime  int64
	Traffic     int64
	Download    int64
	Upload      int64
	Status      int

	// 过期期间流量
	ExpiredDownload int64
	ExpiredUpload   int64

	// 扩展信息
	SubscribeName string
	NodeGroups    []int64
	NodeTags      []string
	NodeGroupID   int64
}

// UserInfo 用户信息
type UserInfo struct {
	ID         int64
	Email      string
	InviteCode string

	// 扩展信息（用于模板渲染）
	Password     string    `json:"password"`
	ExpiredAt    time.Time `json:"expired_at"`
	Download     int64     `json:"download"`
	Upload       int64     `json:"upload"`
	Traffic      int64     `json:"traffic"`
	SubscribeURL string    `json:"subscribe_url"`
	// SubscribeID is the ProxyUserSubscribe table primary key (SID).
	// Used by SimNet per-user key_id derivation to match the Node's key_id.
	SubscribeID int64 `json:"subscribe_id"`
}

// NodeInfo 节点信息
type NodeInfo struct {
	ID          int64
	Sort        int
	Name        string
	Server      string
	Port        uint16
	Type        string
	Tags        []string
	NodeGroupID int64

	Security                           string
	SNI                                string
	AllowInsecure                      bool
	Fingerprint                        string
	RealityServerAddr                  string
	RealityServerPort                  int
	RealityPrivateKey                  string
	RealityPublicKey                   string
	RealityShortId                     string
	Transport                          string
	Host                               string
	Path                               string
	ServiceName                        string
	Mc1Mode                            string
	Mc1CidrSegments                    []string
	MundoUsername                      string
	MundoCertificateFingerprint        string
	MundoFakeTitle                     string
	MundoFakeMessage                   string
	MundoAcceptProxyProtocol           bool
	MundoUseTLSCertificate             bool
	Method                             string
	ServerKey                          string
	Flow                               string
	HopPorts                           string
	HopInterval                        int
	ObfsPassword                       string
	UpMbps                             int
	DownMbps                           int
	DisableSNI                         bool
	ReduceRtt                          bool
	UDPRelayMode                       string
	CongestionController               string
	PaddingScheme                      string
	Multiplex                          string
	XhttpMode                          string
	XhttpExtra                         string
	Encryption                         string
	EncryptionMode                     string
	EncryptionRtt                      string
	EncryptionTicket                   string
	EncryptionServerPadding            string
	EncryptionPrivateKey               string
	EncryptionClientPadding            string
	EncryptionPassword                 string
	Ratio                              float64
	CertMode                           string
	CertDNSProvider                    string
	CertDNSEnv                         string
	SimnetPsk                          string
	SimnetKeyID                        int
	SimnetTicketID                     string
	SimnetPath                         string
	SimnetCarrier                      string
	SimnetAfEnabled                    bool
	SimnetAfPathMode                   string
	SimnetAfPathPrefix                 string
	SimnetAfPathSuffix                 string
	SimnetAfMagicMode                  string
	SimnetAfResponseJitterMs           int
	SimnetAfHandshakePolymorphism      bool
	SimnetAfSettingsJitter             bool
	SimnetAfFakeHeaderInjection        bool
	SimnetClientMaxConcurrentStreams   int
	SimnetClientMaxStreamsPerSession   int
	SimnetClientSessionIdleTimeoutSecs int
	SimnetClientMaxUDPSessions         int
	OmniflowCarrier                    string
	OmniflowPath                       string
	OmniflowContentType                string
	OmniflowProfileJson                string
	OmniflowCaCertPath                 string
	OmniflowTargetMeta                 string
	OmniflowSpkiPin                    string
	OmniflowAdaptiveTlsEnabled         bool
	OmniflowTlsFingerprint             string
	OmniflowSniMode                    string
	OmniflowPaddingMode                string
	OmniflowAfEnabled                  bool
	OmniflowAfPathMode                 string
	OmniflowAfPathPrefix               string
	OmniflowAfPathSuffix               string
	OmniflowAfPathRotationSecs         int
	OmniflowAfPathSkewSlots            int
}

func (n *NodeInfo) NormalizeSimnet() {
	if n == nil || n.Type != "simnet" {
		return
	}
	if strings.TrimSpace(n.SimnetPath) == "" {
		n.SimnetPath = "/simnet/session"
	}
	if n.SimnetClientMaxConcurrentStreams <= 0 {
		n.SimnetClientMaxConcurrentStreams = 32
	}
	if n.SimnetClientMaxStreamsPerSession <= 0 {
		n.SimnetClientMaxStreamsPerSession = 512
	}
	if n.SimnetClientSessionIdleTimeoutSecs <= 0 {
		n.SimnetClientSessionIdleTimeoutSecs = 90
	}
	if n.SimnetClientMaxUDPSessions <= 0 {
		n.SimnetClientMaxUDPSessions = 64
	}
	if !n.SimnetAfEnabled {
		n.SimnetAfPathMode = ""
		n.SimnetAfMagicMode = ""
		n.SimnetAfPathPrefix = ""
		n.SimnetAfPathSuffix = ""
		n.SimnetAfResponseJitterMs = 0
		n.SimnetAfHandshakePolymorphism = false
		n.SimnetAfSettingsJitter = false
		n.SimnetAfFakeHeaderInjection = false
		return
	}
	if n.SimnetAfPathMode == "" {
		n.SimnetAfPathMode = "api"
	}
	if n.SimnetAfMagicMode == "" {
		n.SimnetAfMagicMode = "derived"
	}
	if n.SimnetAfResponseJitterMs == 0 {
		n.SimnetAfResponseJitterMs = 50
	}
	if !n.SimnetAfHandshakePolymorphism {
		n.SimnetAfHandshakePolymorphism = true
	}
	if !n.SimnetAfSettingsJitter {
		n.SimnetAfSettingsJitter = true
	}
	if !n.SimnetAfFakeHeaderInjection {
		n.SimnetAfFakeHeaderInjection = true
	}
}
