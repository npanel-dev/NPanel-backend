package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/npanel-dev/NPanel-backend/api/server/v1"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
)

// ServerNodeRepo 节点服务器仓储接口
type ServerNodeRepo interface {
	GetNodeSecret(ctx context.Context) (string, error)
	GetServerConfig(ctx context.Context, serverID int64, protocol string) (*ServerConfig, error)
	GetServerUserList(ctx context.Context, serverID int64, protocol string) ([]*ServerUser, error)
	PushUserTraffic(ctx context.Context, req *PushUserTrafficRequest) error
	PushServerStatus(ctx context.Context, req *PushServerStatusRequest) error
	PushOnlineUsers(ctx context.Context, req *PushOnlineUsersRequest) error
	QueryServerProtocolConfig(ctx context.Context, serverID int64) (*ProtocolConfig, error)
	SessionCheck(ctx context.Context, serverID int64, userID int64, identifier string, deviceLimit int64) (allowed bool, current int64, err error)
	SessionRelease(ctx context.Context, userID int64, identifier string) error
	GetDeviceCountMode(ctx context.Context) (string, error)
	GetDeviceAdmissionEnabled(ctx context.Context) (bool, error)
	GetUserDeviceLimit(ctx context.Context, userID int64) (int64, error)
}

// ServerConfig 服务器配置
type ServerConfig struct {
	PushInterval int64
	PullInterval int64
	Protocol     string
	Config       string
}

// ServerUser 服务器用户
type ServerUser struct {
	ID          int64
	UUID        string
	SpeedLimit  int64
	DeviceLimit int64
}

// UserTraffic 用户流量
type UserTraffic struct {
	SID      int64
	Upload   int64
	Download int64
}

// PushUserTrafficRequest 推送用户流量请求
type PushUserTrafficRequest struct {
	ServerID  int64
	Protocol  string
	SecretKey string
	Traffic   []*UserTraffic
}

// PushServerStatusRequest 推送服务器状态请求
type PushServerStatusRequest struct {
	ServerID  int64
	Protocol  string
	SecretKey string
	CPU       float64
	Mem       float64
	Disk      float64
	UpdatedAt int64
}

// OnlineUser 在线用户
type OnlineUser struct {
	SID int64
	IP  string
}

// PushOnlineUsersRequest 推送在线用户请求
type PushOnlineUsersRequest struct {
	ServerID  int64
	Protocol  string
	SecretKey string
	Users     []*OnlineUser
}

// Protocol 协议配置
type Protocol struct {
	Type                                   string
	Port                                   int32
	Enable                                 bool
	Security                               string
	SNI                                    string
	AllowInsecure                          bool
	Fingerprint                            string
	RealityServerAddr                      string
	RealityServerPort                      int32
	RealityPrivateKey                      string
	RealityPublicKey                       string
	RealityShortId                         string
	Transport                              string
	Host                                   string
	Path                                   string
	ServiceName                            string
	Mc1Mode                                string
	Mc1CidrSegments                        []string
	MundoUsername                          string
	MundoCertificateFingerprint            string
	MundoFakeTitle                         string
	MundoFakeMessage                       string
	MundoAcceptProxyProtocol               bool
	MundoUseTLSCertificate                 bool
	Cipher                                 string
	ServerKey                              string
	Flow                                   string
	HopPorts                               string
	HopInterval                            int32
	ObfsPassword                           string
	DisableSNI                             bool
	ReduceRtt                              bool
	UDPRelayMode                           string
	CongestionController                   string
	Multiplex                              string
	PaddingScheme                          string
	UpMbps                                 int32
	DownMbps                               int32
	Obfs                                   string
	ObfsHost                               string
	ObfsPath                               string
	XhttpMode                              string
	XhttpExtra                             string
	Encryption                             string
	EncryptionMode                         string
	EncryptionRtt                          string
	EncryptionTicket                       string
	EncryptionServerPadding                string
	EncryptionPrivateKey                   string
	EncryptionClientPadding                string
	EncryptionPassword                     string
	Ratio                                  float64
	CertMode                               string
	CertDNSProvider                        string
	CertDNSEnv                             string
	SimnetPsk                              string
	SimnetKeyID                            int32
	SimnetTicketID                         string
	SimnetPath                             string
	SimnetCarrier                          string
	SimnetAfEnabled                        bool
	SimnetAfPathMode                       string
	SimnetAfPathPrefix                     string
	SimnetAfPathSuffix                     string
	SimnetAfMagicMode                      string
	SimnetAfResponseJitterMs               int32
	SimnetAfHandshakePolymorphism          bool
	SimnetAfSettingsJitter                 bool
	SimnetAfFakeHeaderInjection            bool
	SimnetReverseEnabled                   bool
	SimnetReverseListenAddr                string
	SimnetReverseListenPort                int32
	SimnetReverseTargetHost                string
	SimnetReverseTargetPort                int32
	SimnetFallbackEnabled                  bool
	SimnetFallbackTargetScheme             string
	SimnetFallbackTargetHost               string
	SimnetFallbackTargetPort               int32
	SimnetFallbackHostHeader               string
	SimnetFallbackTLSSNI                   string
	SimnetInboundMaxStreamsPerSession      int32
	SimnetInboundMaxUDPStreamsPerSession   int32
	SimnetInboundMaxHandlerTasksPerSession int32
	SimnetStreamEventChannelCapacity       int32
	SimnetStreamDataChannelCapacity        int32
	SimnetTargetDialTimeoutMs              int32
	SimnetTargetMaxConcurrentDials         int32
	SimnetEgressBlockLoopback              bool
	SimnetEgressBlockPrivate               bool
	SimnetEgressBlockLinkLocal             bool
	SimnetEgressBlockMetadata              bool
	SimnetSendWindow                       int32
	SimnetRecvWindow                       int32
	SimnetMaxConcurrentStreams             int32
	SimnetInitialWindowSize                int32
	SimnetMaxFrameSize                     int32
	SimnetClientMaxConcurrentStreams       int32
	SimnetClientMaxStreamsPerSession       int32
	SimnetClientSessionIdleTimeoutSecs     int32
	SimnetClientMaxUDPSessions             int32

	// OmniFlow 基础配置
	OmniflowCarrier     string
	OmniflowPath        string
	OmniflowContentType string
	OmniflowProfilePath string
	OmniflowProfileJson string
	OmniflowServerHost  string
	OmniflowServerPort  int32
	OmniflowCaCertPath  string
	OmniflowTargetMeta  string
	OmniflowSpkiPin     string

	// OmniFlow H3 Fallback 策略
	OmniflowH3FallbackEnabled          bool
	OmniflowH3FallbackPolicy           string
	OmniflowH3FallbackTimeoutMs        int32
	OmniflowH3FallbackRetryBudget      int32
	OmniflowH3FallbackSmokeEnabled     bool
	OmniflowH3FallbackSmokeIntervalSec int32
	OmniflowH3FallbackSmokeTimeoutMs   int32

	// OmniFlow 连接管理
	OmniflowMaxAgeSec      int32
	OmniflowIdleTimeoutSec int32
	OmniflowMaxConnections int32

	// OmniFlow 抗指纹
	OmniflowAdaptiveTlsEnabled    bool
	OmniflowTlsFingerprint        string
	OmniflowSniMode               string
	OmniflowPaddingMode           string
	OmniflowTrafficShapingEnabled bool
	OmniflowAfEnabled             bool
	OmniflowAfPathMode            string
	OmniflowAfPathPrefix          string
	OmniflowAfPathSuffix          string
	OmniflowAfPathRotationSecs    int32
	OmniflowAfPathSkewSlots       int32

	// OmniFlow 同端口浏览器 Fallback 反向代理
	OmniflowFallbackEnabled      bool
	OmniflowFallbackTargetScheme string
	OmniflowFallbackTargetHost   string
	OmniflowFallbackTargetPort   int32
	OmniflowFallbackHostHeader   string
	OmniflowFallbackTLSSNI       string

	// OmniFlow 回退 Carrier
	OmniflowFallbackCarrierEnabled bool
	OmniflowFallbackConnectTunnel  bool
	OmniflowFallbackWssEnabled     bool
}

// ProtocolConfig 协议配置响应
type ProtocolConfig struct {
	TrafficReportThreshold int64
	IPStrategy             string
	DNS                    []*DNSConfig
	Block                  []string
	Outbound               []*OutboundConfig
	Protocols              []*Protocol
	Total                  int32
}

// DNSConfig DNS配置
type DNSConfig struct {
	Server string
	Domain string
	Port   int64
}

// OutboundConfig 出站配置
type OutboundConfig struct {
	Tag      string
	Protocol string
	Settings map[string]string
}

// ServerNodeUsecase 节点服务器用例
type ServerNodeUsecase struct {
	repo   ServerNodeRepo
	logger *log.Helper
}

// NewServerNodeUsecase 创建节点服务器用例
func NewServerNodeUsecase(repo ServerNodeRepo, logger log.Logger) *ServerNodeUsecase {
	return &ServerNodeUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// validateSecretKey 验证密钥
// 老项目语义：直接比对运行时 NodeSecret，而不是按 server_id 派生。
func (uc *ServerNodeUsecase) validateSecretKey(ctx context.Context, secretKey string) (bool, error) {
	expectedKey, err := uc.repo.GetNodeSecret(ctx)
	if err != nil {
		return false, err
	}
	return expectedKey == secretKey, nil
}

// GetServerConfig 获取服务器配置
func (uc *ServerNodeUsecase) GetServerConfig(ctx context.Context, serverID int64, protocol, secretKey string) (*ServerConfig, error) {
	valid, err := uc.validateSecretKey(ctx, secretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for server %d", serverID)
		return nil, responsecode.ErrUnauthorized()
	}

	config, err := uc.repo.GetServerConfig(ctx, serverID, protocol)
	if err != nil {
		uc.logger.Errorf("GetServerConfig failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	return config, nil
}

// GetDeviceAdmissionEnabled 获取设备准入控制全局开关
func (uc *ServerNodeUsecase) GetDeviceAdmissionEnabled(ctx context.Context) (bool, error) {
	return uc.repo.GetDeviceAdmissionEnabled(ctx)
}

// GetDeviceCountMode 获取设备计数模式
func (uc *ServerNodeUsecase) GetDeviceCountMode(ctx context.Context) (string, error) {
	return uc.repo.GetDeviceCountMode(ctx)
}

// GetServerUserList 获取服务器用户列表
func (uc *ServerNodeUsecase) GetServerUserList(ctx context.Context, serverID int64, protocol, secretKey string) ([]*ServerUser, error) {
	valid, err := uc.validateSecretKey(ctx, secretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for server %d", serverID)
		return nil, responsecode.ErrUnauthorized()
	}

	users, err := uc.repo.GetServerUserList(ctx, serverID, protocol)
	if err != nil {
		uc.logger.Errorf("GetServerUserList failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	return users, nil
}

// PushUserTraffic 推送用户流量
func (uc *ServerNodeUsecase) PushUserTraffic(ctx context.Context, req *PushUserTrafficRequest) error {
	valid, err := uc.validateSecretKey(ctx, req.SecretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for server %d", req.ServerID)
		return responsecode.ErrUnauthorized()
	}

	err = uc.repo.PushUserTraffic(ctx, req)
	if err != nil {
		uc.logger.Errorf("PushUserTraffic failed: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	return nil
}

// PushServerStatus 推送服务器状态
func (uc *ServerNodeUsecase) PushServerStatus(ctx context.Context, req *PushServerStatusRequest) error {
	valid, err := uc.validateSecretKey(ctx, req.SecretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for server %d", req.ServerID)
		return responsecode.ErrUnauthorized()
	}

	err = uc.repo.PushServerStatus(ctx, req)
	if err != nil {
		uc.logger.Errorf("PushServerStatus failed: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	return nil
}

// PushOnlineUsers 推送在线用户
func (uc *ServerNodeUsecase) PushOnlineUsers(ctx context.Context, req *PushOnlineUsersRequest) error {
	valid, err := uc.validateSecretKey(ctx, req.SecretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for server %d", req.ServerID)
		return responsecode.ErrUnauthorized()
	}

	err = uc.repo.PushOnlineUsers(ctx, req)
	if err != nil {
		uc.logger.Errorf("PushOnlineUsers failed: %v", err)
		return responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	return nil
}

// SessionCheck 设备准入检查
func (uc *ServerNodeUsecase) SessionCheck(ctx context.Context, req *v1.SessionCheckRequest) (*v1.SessionCheckResponse, error) {
	// 1. 验证密钥
	valid, err := uc.validateSecretKey(ctx, req.SecretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for SessionCheck server %d", req.ServerId)
		return nil, responsecode.ErrUnauthorized()
	}

	// 检查全局开关
	enabled, err := uc.repo.GetDeviceAdmissionEnabled(ctx)
	if err != nil {
		// 开关读取失败，保守策略：放行
		return &v1.SessionCheckResponse{Allowed: true, Current: 0, Limit: 0, Reason: ""}, nil
	}
	if !enabled {
		// 功能未启用，直接放行
		return &v1.SessionCheckResponse{Allowed: true, Current: 0, Limit: 0, Reason: ""}, nil
	}

	// 2. 读取 device_count_mode
	mode, err := uc.repo.GetDeviceCountMode(ctx)
	if err != nil {
		uc.logger.Errorf("GetDeviceCountMode failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// 3. 根据模式确定 identifier
	identifier := req.ClientIp
	if mode == "connection" {
		if req.ConnectionId != "" {
			identifier = req.ConnectionId
		}
	}

	// 4. 查询用户的 device_limit
	deviceLimit, err := uc.repo.GetUserDeviceLimit(ctx, req.UserId)
	if err != nil {
		uc.logger.Errorf("GetUserDeviceLimit failed for user %d: %v", req.UserId, err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// device_limit == 0 表示不限制，直接放行
	if deviceLimit == 0 {
		return &v1.SessionCheckResponse{
			Allowed: true,
			Current: 0,
			Limit:   0,
			Reason:  "",
		}, nil
	}

	// 5. 调用 repo.SessionCheck
	allowed, current, err := uc.repo.SessionCheck(ctx, req.ServerId, req.UserId, identifier, deviceLimit)
	if err != nil {
		uc.logger.Errorf("SessionCheck failed for user %d: %v", req.UserId, err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// 6. 构造响应
	resp := &v1.SessionCheckResponse{
		Allowed: allowed,
		Current: current,
		Limit:   deviceLimit,
	}
	if !allowed {
		resp.Reason = fmt.Sprintf("device limit exceeded: current=%d, limit=%d", current, deviceLimit)
	}
	return resp, nil
}

// SessionRelease 会话释放
func (uc *ServerNodeUsecase) SessionRelease(ctx context.Context, req *v1.SessionReleaseRequest) (*v1.SessionReleaseResponse, error) {
	// 1. 验证密钥
	valid, err := uc.validateSecretKey(ctx, req.SecretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for SessionRelease server %d", req.ServerId)
		return nil, responsecode.ErrUnauthorized()
	}

	// 2. 读取 device_count_mode
	mode, err := uc.repo.GetDeviceCountMode(ctx)
	if err != nil {
		uc.logger.Errorf("GetDeviceCountMode failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	// 3. 确定 identifier
	identifier := req.ClientIp
	if mode == "connection" {
		if req.ConnectionId != "" {
			identifier = req.ConnectionId
		}
	}

	// 4. 调用 repo.SessionRelease
	err = uc.repo.SessionRelease(ctx, req.UserId, identifier)
	if err != nil {
		uc.logger.Errorf("SessionRelease failed for user %d: %v", req.UserId, err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseUpdate)
	}

	// 5. 返回成功
	return &v1.SessionReleaseResponse{Success: true}, nil
}

// QueryServerProtocolConfig 查询服务器协议配置
func (uc *ServerNodeUsecase) QueryServerProtocolConfig(ctx context.Context, serverID int64, secretKey string, protocols []string) (*ProtocolConfig, error) {
	// 验证密钥
	valid, err := uc.validateSecretKey(ctx, secretKey)
	if err != nil {
		uc.logger.Errorf("Load node secret failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}
	if !valid {
		uc.logger.Errorf("Invalid secret key for server %d", serverID)
		return nil, responsecode.ErrUnauthorized()
	}

	config, err := uc.repo.QueryServerProtocolConfig(ctx, serverID)
	if err != nil {
		uc.logger.Errorf("QueryServerProtocolConfig failed: %v", err)
		return nil, responsecode.NewKratosError(responsecode.ErrDatabaseQuery)
	}

	if config != nil {
		enabledProtocols := make([]*Protocol, 0, len(config.Protocols))
		for _, protocol := range config.Protocols {
			if protocol == nil || !protocol.Enable {
				continue
			}
			enabledProtocols = append(enabledProtocols, protocol)
		}
		config.Protocols = enabledProtocols
		config.Total = int32(len(enabledProtocols))

		if len(protocols) > 0 {
			requested := make(map[string]struct{}, len(protocols))
			for _, protocol := range protocols {
				if protocol = strings.TrimSpace(protocol); protocol != "" {
					requested[protocol] = struct{}{}
				}
			}
			if len(requested) > 0 {
				filtered := make([]*Protocol, 0, len(config.Protocols))
				for _, protocol := range config.Protocols {
					if protocol == nil {
						continue
					}
					if _, ok := requested[protocol.Type]; ok {
						filtered = append(filtered, protocol)
					}
				}
				config.Protocols = filtered
				config.Total = int32(len(filtered))
			}
		}
	}

	return config, nil
}
