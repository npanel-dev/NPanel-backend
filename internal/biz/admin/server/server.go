package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/ip"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

// Server represents a proxy server
type Server struct {
	ID             int64
	Name           string
	Country        string
	City           string
	Address        string
	Sort           int
	Protocols      []*server.Protocol
	LastReportedAt int64
	Status         *ServerStatus
	CreatedAt      int64
	UpdatedAt      int64
}

type ServerStatus struct {
	Cpu      float64
	Mem      float64
	Disk     float64
	Protocol string
	Online   []*ServerOnlineUser
	Status   string
}

type ServerOnlineUser struct {
	IP          []*ServerOnlineIP
	UserID      int64
	Subscribe   string
	SubscribeID int64
	Traffic     int64
	ExpiredAt   int64
}

type ServerOnlineIP struct {
	IP       string
	Protocol string
}

type Node struct {
	ID           int64
	Name         string
	Tags         []string
	Port         uint16
	Address      string
	ServerID     int64
	Protocol     string
	Enabled      *bool
	NodeType     string
	IsHidden     *bool
	TargetNodeID *int64
	Sort         uint32
	NodeGroupID  int64
	NodeGroupIDs []int64
	CreatedAt    int64
	UpdatedAt    int64
}

type SortItem struct {
	ID   int64
	Sort int
}

type UserSubscribeInfo struct {
	UserID      int64
	SubscribeID int64
	Subscribe   string
	Download    int64
	Upload      int64
	ExpireTime  int64
}

type ServerRepo interface {
	CreateServer(ctx context.Context, server *Server) (*Server, error)
	UpdateServer(ctx context.Context, server *Server) (*Server, error)
	DeleteServer(ctx context.Context, id int) error
	GetServerByID(ctx context.Context, id int) (*Server, error)
	FilterServerList(ctx context.Context, page, size int32, search string) (int32, []*Server, error)
	GetServerProtocols(ctx context.Context, id int) ([]*server.Protocol, error)
	ResetServerSort(ctx context.Context, sortItems []*SortItem) error
	GetServerStatus(ctx context.Context, serverID int) (*ServerResourceStatus, error)
	GetOnlineUsers(ctx context.Context, serverID int64, protocol string, port uint16) (map[int64][]string, error)
	GetOnlineUsersByServer(ctx context.Context, serverID int64) (map[string]map[int64][]string, error)
	GetUserSubscribeInfo(ctx context.Context, subscribeID int) (*UserSubscribeInfo, error)
}

type ServerResourceStatus struct {
	Cpu  float64
	Mem  float64
	Disk float64
}

type NodeRepo interface {
	CreateNode(ctx context.Context, node *Node) (*Node, error)
	UpdateNode(ctx context.Context, node *Node) (*Node, error)
	DeleteNode(ctx context.Context, id int) error
	FilterNodeList(ctx context.Context, page, size int32, search string, nodeGroupID *int64) (int32, []*Node, error)
	ToggleNodeStatus(ctx context.Context, id int, enable *bool) (*Node, error)
	QueryNodeTags(ctx context.Context) ([]string, error)
	ResetNodeSort(ctx context.Context, sortItems []*SortItem) error
	ClearNodeCache(ctx context.Context, serverIDs []int) error
	GetServerProtocols(ctx context.Context, id int) ([]*server.Protocol, error)
	GetNodeByID(ctx context.Context, id int64) (*Node, error)
	CountFrontNodesByTarget(ctx context.Context, targetNodeID int64) (int, error)
}

type MigrationRepo interface {
	HasMigrateServerNode(ctx context.Context) (bool, error)
	MigrateServerNode(ctx context.Context) (uint64, uint64, string, error)
}

type ServerUsecase struct {
	repo     ServerRepo
	nodeRepo NodeRepo
	log      *log.Helper
}

func NewServerUsecase(repo ServerRepo, nodeRepo NodeRepo, logger log.Logger) *ServerUsecase {
	return &ServerUsecase{repo: repo, nodeRepo: nodeRepo, log: log.NewHelper(logger)}
}

func normalizeSimnetProtocol(protocol *server.Protocol, address string) {
	if protocol.Type != "simnet" {
		return
	}
	if protocol.Port == 0 {
		protocol.Port = 443
	}
	if protocol.SimnetCarrier == "" || protocol.SimnetCarrier == "grpc" {
		protocol.SimnetCarrier = "h2"
	}
	if protocol.CertMode == "http" || protocol.CertMode == "dns" {
		if strings.TrimSpace(protocol.SNI) == "" && isLikelyDomainAddress(address) {
			protocol.SNI = strings.TrimSpace(address)
		}
		if protocol.Security == "" || protocol.Security == "none" {
			protocol.Security = "tls"
		}
	} else if strings.TrimSpace(protocol.SNI) == "" {
		protocol.Security = ""
	} else if protocol.Security == "" || protocol.Security == "none" {
		protocol.Security = "tls"
	}
	protocol.NormalizeSimnet()
	protocol.NormalizeOmniflow()
}

func normalizeMxProtocol(protocol *server.Protocol, address string) {
	if protocol == nil || protocol.Type != "mx" {
		return
	}
	protocol.Transport = strings.ToLower(strings.TrimSpace(protocol.Transport))
	if protocol.Transport == "" {
		protocol.Transport = "tcp"
	}
	protocol.Security = strings.ToLower(strings.TrimSpace(protocol.Security))
	if protocol.Security == "" {
		protocol.Security = "none"
	}
	if (protocol.Security == "tls" || protocol.Security == "reality") && strings.TrimSpace(protocol.Fingerprint) == "" {
		protocol.Fingerprint = "chrome"
	}
	if protocol.Transport == "mc1" {
		if strings.TrimSpace(protocol.Path) == "" {
			protocol.Path = "/"
		}
		if strings.TrimSpace(protocol.Host) == "" {
			protocol.Host = address
		}
		if strings.TrimSpace(protocol.Mc1Mode) == "" {
			protocol.Mc1Mode = "auto"
		}
	}
}

func normalizeMundoProtocol(protocol *server.Protocol) {
	if protocol == nil || protocol.Type != "mx" || !isMundoNetwork(protocol.Transport) {
		return
	}
	if strings.TrimSpace(protocol.MundoUsername) == "" {
		protocol.MundoUsername = "MundoUser"
	}
}

func isMundoNetwork(transport string) bool {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "mundordp", "mundosql":
		return true
	default:
		return false
	}
}

func isLikelyDomainAddress(address string) bool {
	address = strings.TrimSpace(address)
	if address == "" || strings.Contains(address, ":") {
		return false
	}
	hasLetter := false
	for _, r := range address {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
			break
		}
	}
	return hasLetter && strings.Contains(address, ".")
}

func (uc *ServerUsecase) CreateServer(ctx context.Context, name, country, city, address string, sort int64, protocols []*server.Protocol) (*Server, error) {
	if err := server.ValidateProtocols(protocols); err != nil {
		return nil, err
	}
	processedProtocols := make([]*server.Protocol, len(protocols))
	for i, proto := range protocols {
		copied := &server.Protocol{}
		*copied = *proto
		processedProtocols[i] = copied
	}
	for _, protocol := range processedProtocols {
		if (protocol.Type == "vless" || protocol.Type == "mx") && protocol.Security == "reality" {
			if protocol.RealityPublicKey == "" {
				public, private, err := tool.Curve25519Genkey(false, "")
				if err != nil {
					uc.log.Errorf("Failed to generate Curve25519 key: %v", err)
					return nil, err
				}
				protocol.RealityPublicKey = public
				protocol.RealityPrivateKey = private
				protocol.RealityShortId = tool.GenerateShortID(private)
			}
			if protocol.RealityServerAddr == "" {
				protocol.RealityServerAddr = protocol.SNI
			}
			if protocol.RealityServerPort == 0 {
				protocol.RealityServerPort = 443
			}
		}
		if protocol.Type == "shadowsocks" && strings.Contains(protocol.Cipher, "2022") {
			length := 32
			if protocol.Cipher == "2022-blake3-aes-128-gcm" {
				length = 16
			}
			if len(protocol.ServerKey) != length {
				protocol.ServerKey = tool.GenerateCipher(protocol.ServerKey, length)
			}
		}
		normalizeSimnetProtocol(protocol, address)
		normalizeMxProtocol(protocol, address)
		normalizeMundoProtocol(protocol)
	}
	if country == "" && city == "" && address != "" {
		location, err := ip.GetRegionByIp(address)
		if err == nil && location != nil {
			country = location.Country
			city = location.City
		}
	}
	return uc.repo.CreateServer(ctx, &Server{Name: name, Country: country, City: city, Address: address, Sort: int(sort), Protocols: processedProtocols})
}

func (uc *ServerUsecase) UpdateServer(ctx context.Context, id int, name, country, city, address string, sort int64, protocols []*server.Protocol) (*Server, error) {
	existingServer, err := uc.repo.GetServerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := server.ValidateProtocols(protocols); err != nil {
		return nil, err
	}
	processedProtocols := make([]*server.Protocol, len(protocols))
	for i, proto := range protocols {
		copied := &server.Protocol{}
		*copied = *proto
		processedProtocols[i] = copied
	}
	for _, protocol := range processedProtocols {
		if (protocol.Type == "vless" || protocol.Type == "mx") && protocol.Security == "reality" {
			if protocol.RealityPublicKey == "" {
				public, private, err := tool.Curve25519Genkey(false, "")
				if err != nil {
					uc.log.Errorf("Failed to generate Curve25519 key: %v", err)
					return nil, err
				}
				protocol.RealityPublicKey = public
				protocol.RealityPrivateKey = private
				protocol.RealityShortId = tool.GenerateShortID(private)
			}
			if protocol.RealityServerAddr == "" {
				protocol.RealityServerAddr = protocol.SNI
			}
			if protocol.RealityServerPort == 0 {
				protocol.RealityServerPort = 443
			}
		}
		if protocol.Type == "shadowsocks" && strings.Contains(protocol.Cipher, "2022") {
			length := 32
			if protocol.Cipher == "2022-blake3-aes-128-gcm" {
				length = 16
			}
			if len(protocol.ServerKey) != length {
				protocol.ServerKey = tool.GenerateCipher(protocol.ServerKey, length)
			}
		}
		normalizeSimnetProtocol(protocol, address)
		normalizeMxProtocol(protocol, address)
		normalizeMundoProtocol(protocol)
	}
	if address != existingServer.Address || existingServer.Country == "" || country == "" {
		location, err := ip.GetRegionByIp(address)
		if err == nil && location != nil {
			country = location.Country
			city = location.City
		}
	}
	updatedServer, err := uc.repo.UpdateServer(ctx, &Server{ID: int64(id), Name: name, Country: country, City: city, Address: address, Sort: int(sort), Protocols: processedProtocols})
	if err != nil {
		return nil, err
	}
	if err := uc.nodeRepo.ClearNodeCache(ctx, []int{id}); err != nil {
		uc.log.Warnf("Failed to clear node cache for server %d: %v", id, err)
	}
	return updatedServer, nil
}

func (uc *ServerUsecase) DeleteServer(ctx context.Context, id int) error {
	if err := uc.repo.DeleteServer(ctx, id); err != nil {
		return err
	}
	if err := uc.nodeRepo.ClearNodeCache(ctx, []int{id}); err != nil {
		uc.log.Warnf("Failed to clear node cache for deleted server %d: %v", id, err)
	}
	return nil
}

func (uc *ServerUsecase) FilterServerList(ctx context.Context, page, size int32, search string) (int32, []*Server, error) {
	total, servers, err := uc.repo.FilterServerList(ctx, page, size, search)
	if err != nil {
		return 0, nil, err
	}
	for _, srv := range servers {
		srv.Status = uc.buildServerStatus(ctx, srv)
	}
	return total, servers, nil
}

func (uc *ServerUsecase) GetServerProtocols(ctx context.Context, id int) ([]*server.Protocol, error) {
	return uc.repo.GetServerProtocols(ctx, id)
}

func (uc *ServerUsecase) ResetServerSort(ctx context.Context, sortItems []*SortItem) error {
	return uc.repo.ResetServerSort(ctx, sortItems)
}

func (uc *ServerUsecase) buildServerStatus(ctx context.Context, srv *Server) *ServerStatus {
	status := &ServerStatus{Cpu: 0, Mem: 0, Disk: 0, Protocol: "", Online: make([]*ServerOnlineUser, 0), Status: uc.getServerStatusString(int(srv.LastReportedAt))}
	resourceStatus, err := uc.repo.GetServerStatus(ctx, int(srv.ID))
	if err == nil && resourceStatus != nil {
		status.Cpu = resourceStatus.Cpu
		status.Mem = resourceStatus.Mem
		status.Disk = resourceStatus.Disk
	}
	status.Online = uc.getOnlineUsers(ctx, srv.ID, srv.Protocols)
	return status
}

func (uc *ServerUsecase) getServerStatusString(lastReportedAt int) string {
	if lastReportedAt == 0 {
		return "offline"
	}
	lastReported := time.Unix(int64(lastReportedAt/1000), 0)
	elapsed := time.Since(lastReported)
	if elapsed > 5*time.Minute {
		return "offline"
	}
	if elapsed > 3*time.Minute {
		return "warning"
	}
	return "online"
}

func (uc *ServerUsecase) getOnlineUsers(ctx context.Context, serverID int64, protocols []*server.Protocol) []*ServerOnlineUser {
	result := make([]*ServerOnlineUser, 0)
	legacyFallbackRead := make(map[string]struct{})
	for _, protocol := range protocols {
		if protocol == nil {
			continue
		}
		port := uint16(0)
		if protocol.Port > 0 && protocol.Port <= 65535 {
			port = uint16(protocol.Port)
		}
		onlineData, err := uc.repo.GetOnlineUsers(ctx, serverID, protocol.Type, port)
		if err != nil {
			continue
		}
		if len(onlineData) == 0 {
			legacyKey := strings.ToLower(strings.TrimSpace(protocol.Type))
			if _, read := legacyFallbackRead[legacyKey]; !read {
				legacyFallbackRead[legacyKey] = struct{}{}
				onlineData, err = uc.repo.GetOnlineUsers(ctx, serverID, protocol.Type, 0)
				if err != nil {
					continue
				}
			}
		}
		protocolLabel := protocol.Type
		if port > 0 {
			protocolLabel = fmt.Sprintf("%s:%d", protocol.Type, port)
		}
		for subscribeID, ips := range onlineData {
			ipList := make([]*ServerOnlineIP, 0, len(ips))
			for _, ip := range ips {
				ipList = append(ipList, &ServerOnlineIP{IP: ip, Protocol: protocolLabel})
			}
			result = append(result, &ServerOnlineUser{IP: ipList, SubscribeID: subscribeID})
		}
	}

	return uc.mergeOnlineUsers(ctx, result)
}

func (uc *ServerUsecase) mergeOnlineUsers(ctx context.Context, users []*ServerOnlineUser) []*ServerOnlineUser {
	mergedMap := make(map[int64]*ServerOnlineUser)
	for _, user := range users {
		if existing, exists := mergedMap[user.SubscribeID]; exists {
			existing.Traffic += user.Traffic
			existing.IP = append(existing.IP, user.IP...)
			mergedMap[user.SubscribeID] = existing
			continue
		}
		subscribeInfo, err := uc.repo.GetUserSubscribeInfo(ctx, int(user.SubscribeID))
		if err != nil {
			continue
		}
		mergedMap[user.SubscribeID] = &ServerOnlineUser{IP: user.IP, UserID: subscribeInfo.UserID, Subscribe: subscribeInfo.Subscribe, SubscribeID: user.SubscribeID, Traffic: subscribeInfo.Download + subscribeInfo.Upload, ExpiredAt: subscribeInfo.ExpireTime}
	}
	result := make([]*ServerOnlineUser, 0, len(mergedMap))
	for _, user := range mergedMap {
		result = append(result, user)
	}
	return result
}
