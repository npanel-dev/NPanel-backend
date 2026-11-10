package server

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
)

func relayTestRepo() *adminNodeRepoStub {
	enabled := true
	sourceID := int64(10)
	return &adminNodeRepoStub{
		protocols: []*servermodel.Protocol{
			{Type: "shadowsocks", Port: 443, Enable: true, Ratio: 1},
		},
		nodes: map[int64]*Node{
			sourceID: {
				ID:       sourceID,
				ServerID: 1,
				Protocol: "shadowsocks",
				Port:     443,
				NodeType: NodeTypeLanding,
				Enabled:  &enabled,
			},
		},
	}
}

func TestCreateRelayEntryCreatesLinkWithSourceInstance(t *testing.T) {
	repo := relayTestRepo()
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	node, err := uc.CreateRelayEntry(context.Background(), &RelayEntryParams{
		Name:         "HK 中转",
		Address:      "relay.example.com",
		Port:         8443,
		SourceNodeID: 10,
		ServicePort:  32001,
		Ratio:        0.5,
	})
	if err != nil {
		t.Fatalf("CreateRelayEntry returned error: %v", err)
	}
	if node.NodeType != NodeTypeFront {
		t.Fatalf("node type = %q, want front", node.NodeType)
	}
	if repo.createdRelay == nil {
		t.Fatal("relay entry was not passed to repository")
	}
	if repo.createdRelay.NewInstance == nil || repo.createdRelay.NewInstance.Port != 32001 || repo.createdRelay.NewInstance.Ratio != 0.5 {
		t.Fatalf("new instance not prepared: %+v", repo.createdRelay.NewInstance)
	}
	if repo.createdRelay.Params.Ratio != 0.5 {
		t.Fatalf("ratio = %v, want 0.5", repo.createdRelay.Params.Ratio)
	}
}

func TestCreateRelayEntryRejectsFrontSource(t *testing.T) {
	enabled := true
	repo := relayTestRepo()
	repo.nodes[10] = &Node{ID: 10, ServerID: 1, Protocol: "shadowsocks", Port: 443, NodeType: NodeTypeFront, Enabled: &enabled}
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	_, err := uc.CreateRelayEntry(context.Background(), &RelayEntryParams{
		Name: "x", Address: "relay.example.com", Port: 8443, SourceNodeID: 10, ServicePort: 32001, Ratio: 1,
	})
	if !errors.Is(err, servermodel.ErrInvalidProtocolConfig) {
		t.Fatalf("err = %v, want ErrInvalidProtocolConfig", err)
	}
}

func TestCreateRelayEntryRejectsOccupiedServicePort(t *testing.T) {
	repo := relayTestRepo()
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	_, err := uc.CreateRelayEntry(context.Background(), &RelayEntryParams{
		Name: "x", Address: "relay.example.com", Port: 8443, SourceNodeID: 10,
		ServicePort: 443, // 与源实例端口冲突
		Ratio: 1,
	})
	if !errors.Is(err, servermodel.ErrDuplicateProtocolPort) {
		t.Fatalf("err = %v, want ErrDuplicateProtocolPort", err)
	}
}

func TestCreateRelayEntryRejectsInvalidParams(t *testing.T) {
	uc := NewNodeUsecase(relayTestRepo(), log.DefaultLogger)

	cases := []RelayEntryParams{
		{Name: "", Address: "a.example.com", Port: 1, SourceNodeID: 10, ServicePort: 2, Ratio: 1},
		{Name: "x", Address: " ", Port: 1, SourceNodeID: 10, ServicePort: 2, Ratio: 1},
		{Name: "x", Address: "a.example.com", Port: 0, SourceNodeID: 10, ServicePort: 2, Ratio: 1},
		{Name: "x", Address: "a.example.com", Port: 1, SourceNodeID: 10, ServicePort: 0, Ratio: 1},
		{Name: "x", Address: "a.example.com", Port: 1, SourceNodeID: 0, ServicePort: 2, Ratio: 1},
		{Name: "x", Address: "a.example.com", Port: 1, SourceNodeID: 10, ServicePort: 2, Ratio: 0},
	}
	for i, params := range cases {
		if _, err := uc.CreateRelayEntry(context.Background(), &params); !errors.Is(err, servermodel.ErrInvalidProtocolConfig) {
			t.Fatalf("case %d err = %v, want ErrInvalidProtocolConfig", i, err)
		}
	}
}

func TestUpdateNodeRatioResolvesFrontTarget(t *testing.T) {
	enabled := true
	repo := relayTestRepo()
	repo.nodes[20] = &Node{ID: 20, ServerID: 1, Protocol: "shadowsocks", Port: 443, NodeType: NodeTypeFront, Enabled: &enabled, TargetNodeID: ptrInt64(10)}
	repo.landingCount = 1
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	if err := uc.UpdateNodeRatio(context.Background(), 20, 0.3); err != nil {
		t.Fatalf("UpdateNodeRatio returned error: %v", err)
	}
	if repo.updatedRatio.serverID != 1 || repo.updatedRatio.protocol != "shadowsocks" || repo.updatedRatio.port != 443 || repo.updatedRatio.ratio != 0.3 {
		t.Fatalf("unexpected ratio update: %+v", repo.updatedRatio)
	}
}

func TestUpdateNodeRatioRejectsSharedInstance(t *testing.T) {
	repo := relayTestRepo()
	repo.landingCount = 2
	uc := NewNodeUsecase(repo, log.DefaultLogger)

	err := uc.UpdateNodeRatio(context.Background(), 10, 0.3)
	if !errors.Is(err, servermodel.ErrSharedProtocolInstance) {
		t.Fatalf("err = %v, want ErrSharedProtocolInstance", err)
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
