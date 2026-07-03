package server

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
)

func TestAdminMxNormalizationDefaultsMc1AndMundoRdp(t *testing.T) {
	mc1 := &servermodel.Protocol{Type: "mx", Transport: "mc1", Security: "tls"}
	normalizeMxProtocol(mc1, "node.example.com")
	if mc1.Path != "/" {
		t.Fatalf("mc1 path = %q, want /", mc1.Path)
	}
	if mc1.Host != "node.example.com" {
		t.Fatalf("mc1 host = %q, want node.example.com", mc1.Host)
	}
	if mc1.Mc1Mode != "auto" {
		t.Fatalf("mc1 mode = %q, want auto", mc1.Mc1Mode)
	}
	if mc1.Fingerprint != "chrome" {
		t.Fatalf("mc1 fingerprint = %q, want chrome", mc1.Fingerprint)
	}

	rdp := &servermodel.Protocol{Type: "mx", Transport: "mundordp"}
	normalizeMundoProtocol(rdp)
	if rdp.MundoUsername != "MundoUser" {
		t.Fatalf("mundordp username = %q, want MundoUser", rdp.MundoUsername)
	}
}

func TestAdminCreateServerAllowsMxMc1RdpAndSql(t *testing.T) {
	repo := &adminServerRepoStub{}
	uc := NewServerUsecase(repo, &adminNodeRepoStub{}, log.DefaultLogger)

	server, err := uc.CreateServer(context.Background(), "Mundo", "US", "LA", "node.example.com", 1, []*servermodel.Protocol{
		{Type: "mx", Port: 443, Enable: true, Transport: "mc1", Security: "tls"},
		{Type: "mx", Port: 3389, Enable: true, Transport: "mundordp"},
		{Type: "mx", Port: 3306, Enable: true, Transport: "mundosql", MundoUsername: "sqluser"},
	})
	if err != nil {
		t.Fatalf("CreateServer returned error: %v", err)
	}
	if server != repo.created {
		t.Fatal("CreateServer did not return repository result")
	}
	if len(server.Protocols) != 3 {
		t.Fatalf("protocol count = %d, want 3", len(server.Protocols))
	}
	if server.Protocols[0].Transport != "mc1" ||
		server.Protocols[0].Host != "node.example.com" ||
		server.Protocols[0].Path != "/" ||
		server.Protocols[0].Mc1Mode != "auto" ||
		server.Protocols[0].Fingerprint != "chrome" {
		t.Fatalf("mc1 defaults not applied: %+v", server.Protocols[0])
	}
	if server.Protocols[1].Transport != "mundordp" || server.Protocols[1].MundoUsername != "MundoUser" {
		t.Fatalf("mundordp defaults not applied: %+v", server.Protocols[1])
	}
	if server.Protocols[2].Transport != "mundosql" || server.Protocols[2].MundoUsername != "sqluser" {
		t.Fatalf("mundosql fields not preserved: %+v", server.Protocols[2])
	}
}

func TestAdminCreateServerNormalizesRealityKeyPairFromPrivate(t *testing.T) {
	repo := &adminServerRepoStub{}
	uc := NewServerUsecase(repo, &adminNodeRepoStub{}, log.DefaultLogger)

	private := "IMGZGlt-9z_W-Lg42MNGngb5DIIkiu_1M8Yu3oLboXI"
	_, err := uc.CreateServer(context.Background(), "Reality", "US", "LA", "node.example.com", 1, []*servermodel.Protocol{
		{
			Type:              "vless",
			Port:              49533,
			Enable:            true,
			Security:          "reality",
			SNI:               "www.python.org",
			RealityPrivateKey: private,
			RealityPublicKey:  "x38_lDn7ZXzfheGbLsFweavt0x4HmrzJ6UX1Mf7HJgs",
		},
	})
	if err != nil {
		t.Fatalf("CreateServer returned error: %v", err)
	}

	protocol := repo.created.Protocols[0]
	if protocol.RealityPrivateKey != private {
		t.Fatalf("private key = %q, want input private", protocol.RealityPrivateKey)
	}
	if protocol.RealityPublicKey != "Ondiyf3k-4Q_N2zkuodXjBgUcBCV0SKSGR9YaCww_Hw" {
		t.Fatalf("public key = %q, want derived key", protocol.RealityPublicKey)
	}
	if protocol.RealityShortId == "" {
		t.Fatal("short id should be generated")
	}
	if protocol.RealityServerAddr != "www.python.org" || protocol.RealityServerPort != 443 {
		t.Fatalf("reality server defaults = %s:%d, want www.python.org:443", protocol.RealityServerAddr, protocol.RealityServerPort)
	}
}

func TestAdminUpdateServerNormalizesRealityKeyPairFromPrivate(t *testing.T) {
	repo := &adminServerRepoStub{}
	uc := NewServerUsecase(repo, &adminNodeRepoStub{}, log.DefaultLogger)

	private := "gLHRlZrb8Car1bl1OWXongDlmRf49CevfU8PeESwYG8"
	_, err := uc.UpdateServer(context.Background(), 1, "Reality", "US", "LA", "node.example.com", 1, []*servermodel.Protocol{
		{
			Type:              "vless",
			Port:              49533,
			Enable:            true,
			Security:          "reality",
			SNI:               "www.python.org",
			RealityPrivateKey: private,
			RealityPublicKey:  "Ogl56Lr_DJQoRMnxVvdg2dGJtHdy-GZoJCcfDgL1wF0",
		},
	})
	if err != nil {
		t.Fatalf("UpdateServer returned error: %v", err)
	}

	protocol := repo.updated.Protocols[0]
	if protocol.RealityPrivateKey != private {
		t.Fatalf("private key = %q, want input private", protocol.RealityPrivateKey)
	}
	if protocol.RealityPublicKey != "x38_lDn7ZXzfheGbLsFweavt0x4HmrzJ6UX1Mf7HJgs" {
		t.Fatalf("public key = %q, want derived key", protocol.RealityPublicKey)
	}
}

type adminServerRepoStub struct {
	created *Server
	updated *Server
}

func (r *adminServerRepoStub) CreateServer(ctx context.Context, server *Server) (*Server, error) {
	server.ID = 1
	r.created = server
	return server, nil
}

func (r *adminServerRepoStub) UpdateServer(ctx context.Context, server *Server) (*Server, error) {
	r.updated = server
	return server, nil
}

func (r *adminServerRepoStub) DeleteServer(ctx context.Context, id int) error {
	return nil
}

func (r *adminServerRepoStub) GetServerByID(ctx context.Context, id int) (*Server, error) {
	return &Server{ID: int64(id), Address: "node.example.com"}, nil
}

func (r *adminServerRepoStub) FilterServerList(ctx context.Context, page, size int32, search string) (int32, []*Server, error) {
	return 0, nil, nil
}

func (r *adminServerRepoStub) GetServerProtocols(ctx context.Context, id int) ([]*servermodel.Protocol, error) {
	return nil, nil
}

func (r *adminServerRepoStub) ResetServerSort(ctx context.Context, sortItems []*SortItem) error {
	return nil
}

func (r *adminServerRepoStub) GetServerStatus(ctx context.Context, serverID int) (*ServerResourceStatus, error) {
	return nil, nil
}

func (r *adminServerRepoStub) GetOnlineUsers(ctx context.Context, serverID int64, protocol string, port uint16) (map[int64][]string, error) {
	return nil, nil
}

func (r *adminServerRepoStub) GetOnlineUsersByServer(ctx context.Context, serverID int64) (map[string]map[int64][]string, error) {
	return nil, nil
}

func (r *adminServerRepoStub) GetUserSubscribeInfo(ctx context.Context, subscribeID int) (*UserSubscribeInfo, error) {
	return nil, nil
}

func TestAdminCreateNodeRequiresEnabledProtocolPortInstance(t *testing.T) {
	repo := &adminNodeRepoStub{
		protocols: []*servermodel.Protocol{
			{Type: "vless", Port: 443, Enable: true},
			{Type: "vless", Port: 8443, Enable: true},
			{Type: "shadowsocks", Port: 9443, Enable: false},
		},
	}
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	node, err := uc.CreateNode(context.Background(), "VLESS 8443", nil, 8443, "node.example.com", 1, "vless", nil, "backend", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateNode returned error: %v", err)
	}
	if node.Port != 8443 || node.Protocol != "vless" {
		t.Fatalf("created node = %+v, want vless:8443", node)
	}

	_, err = uc.CreateNode(context.Background(), "VLESS missing", nil, 9443, "node.example.com", 1, "vless", nil, "backend", nil, nil, nil)
	if !errors.Is(err, servermodel.ErrInvalidProtocolConfig) {
		t.Fatalf("CreateNode error = %v, want ErrInvalidProtocolConfig", err)
	}

	_, err = uc.CreateNode(context.Background(), "SS disabled", nil, 9443, "node.example.com", 1, "shadowsocks", nil, "backend", nil, nil, nil)
	if !errors.Is(err, servermodel.ErrInvalidProtocolConfig) {
		t.Fatalf("CreateNode error = %v, want ErrInvalidProtocolConfig", err)
	}
}

type adminNodeRepoStub struct {
	protocols []*servermodel.Protocol
	nodes     map[int64]*Node
}

func (r *adminNodeRepoStub) CreateNode(ctx context.Context, node *Node) (*Node, error) {
	return node, nil
}

func (r *adminNodeRepoStub) UpdateNode(ctx context.Context, node *Node) (*Node, error) {
	return node, nil
}

func (r *adminNodeRepoStub) DeleteNode(ctx context.Context, id int) error {
	return nil
}

func (r *adminNodeRepoStub) FilterNodeList(ctx context.Context, page, size int32, search string, nodeGroupID *int64) (int32, []*Node, error) {
	return 0, nil, nil
}

func (r *adminNodeRepoStub) ToggleNodeStatus(ctx context.Context, id int, enable *bool) (*Node, error) {
	return &Node{ID: int64(id), Enabled: enable}, nil
}

func (r *adminNodeRepoStub) QueryNodeTags(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (r *adminNodeRepoStub) ResetNodeSort(ctx context.Context, sortItems []*SortItem) error {
	return nil
}

func (r *adminNodeRepoStub) ClearNodeCache(ctx context.Context, serverIDs []int) error {
	return nil
}

func (r *adminNodeRepoStub) GetServerProtocols(ctx context.Context, id int) ([]*servermodel.Protocol, error) {
	return r.protocols, nil
}

func (r *adminNodeRepoStub) GetNodeByID(ctx context.Context, id int64) (*Node, error) {
	if r.nodes == nil {
		return nil, nil
	}
	return r.nodes[id], nil
}

func (r *adminNodeRepoStub) CountFrontNodesByTarget(ctx context.Context, targetNodeID int64) (int, error) {
	return 0, nil
}

func TestAdminCreateFrontNodeUsesTargetLandingProtocolValidation(t *testing.T) {
	enabled := true
	targetID := int64(10)
	repo := &adminNodeRepoStub{
		protocols: []*servermodel.Protocol{
			{Type: "shadowsocks", Port: 443, Enable: true},
		},
		nodes: map[int64]*Node{
			targetID: {
				ID:       targetID,
				ServerID: 1,
				Protocol: "shadowsocks",
				Port:     443,
				NodeType: NodeTypeLanding,
				Enabled:  &enabled,
			},
		},
	}
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	node, err := uc.CreateNode(context.Background(), "SG relay", nil, 1443, "10.0.0.1", 0, "", nil, NodeTypeFront, nil, nil, &targetID)
	if err != nil {
		t.Fatalf("CreateNode front returned error: %v", err)
	}
	if node.TargetNodeID == nil || *node.TargetNodeID != targetID {
		t.Fatalf("front target = %v, want %d", node.TargetNodeID, targetID)
	}
	if node.Port != 1443 || node.Protocol != "" || node.ServerID != 0 {
		t.Fatalf("front node should keep custom entry and no own protocol/server: %+v", node)
	}

	_, err = uc.CreateNode(context.Background(), "missing target", nil, 1443, "10.0.0.1", 0, "", nil, NodeTypeFront, nil, nil, nil)
	if !errors.Is(err, servermodel.ErrInvalidProtocolConfig) {
		t.Fatalf("CreateNode missing target error = %v, want ErrInvalidProtocolConfig", err)
	}
}
