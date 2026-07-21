package data

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxynode"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	serverbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/server"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type adminNodeRepo struct {
	data *Data
	log  *log.Helper
}

func NewAdminNodeRepo(data *Data, logger log.Logger) serverbiz.NodeRepo {
	return &adminNodeRepo{data: data, log: log.NewHelper(logger)}
}

func firstNodeGroupID(ids []int64) int64 {
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func adminNodeToBiz(node *ent.ProxyNode) *serverbiz.Node {
	if node == nil {
		return nil
	}
	tags := tool.RemoveDuplicateElements(tool.StringToStringSlice(node.Tags)...)
	enabled := node.Enabled
	return &serverbiz.Node{ID: int64(node.ID), Name: node.Name, Tags: tags, Port: uint16(node.Port), Address: node.Address, ServerID: node.ServerID, Protocol: node.Protocol, Enabled: &enabled, NodeType: node.NodeType, IsHidden: &node.IsHidden, TargetNodeID: node.TargetNodeID, Sort: uint32(node.Sort), NodeGroupID: firstNodeGroupID(node.NodeGroupIds), NodeGroupIDs: node.NodeGroupIds, CreatedAt: node.CreatedAt.UnixMilli(), UpdatedAt: node.UpdatedAt.UnixMilli()}
}

func (r *adminNodeRepo) CreateNode(ctx context.Context, node *serverbiz.Node) (*serverbiz.Node, error) {
	tagsStr := tool.StringSliceToString(node.Tags)
	builder := r.data.db.ProxyNode.Create().SetName(node.Name).SetTags(tagsStr).SetPort(node.Port).SetAddress(node.Address).SetServerID(node.ServerID).SetProtocol(node.Protocol).SetNodeType(node.NodeType).SetSort(int32(node.Sort)).SetNodeGroupIds(node.NodeGroupIDs)
	if node.Enabled != nil {
		builder = builder.SetEnabled(*node.Enabled)
	}
	if node.IsHidden != nil {
		builder = builder.SetIsHidden(*node.IsHidden)
	}
	if node.TargetNodeID != nil && *node.TargetNodeID > 0 {
		builder = builder.SetTargetNodeID(*node.TargetNodeID)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return adminNodeToBiz(created), nil
}

func (r *adminNodeRepo) UpdateNode(ctx context.Context, node *serverbiz.Node) (*serverbiz.Node, error) {
	tagsStr := tool.StringSliceToString(node.Tags)
	builder := r.data.db.ProxyNode.UpdateOneID(node.ID).SetName(node.Name).SetTags(tagsStr).SetPort(node.Port).SetAddress(node.Address).SetServerID(node.ServerID).SetProtocol(node.Protocol).SetNodeType(node.NodeType).SetSort(int32(node.Sort)).SetNodeGroupIds(node.NodeGroupIDs)
	if node.Enabled != nil {
		builder = builder.SetEnabled(*node.Enabled)
	}
	if node.IsHidden != nil {
		builder = builder.SetIsHidden(*node.IsHidden)
	}
	if node.TargetNodeID != nil && *node.TargetNodeID > 0 {
		builder = builder.SetTargetNodeID(*node.TargetNodeID)
	} else {
		builder = builder.ClearTargetNodeID()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return adminNodeToBiz(updated), nil
}

func (r *adminNodeRepo) DeleteNode(ctx context.Context, id int) error {
	node, err := r.data.db.ProxyNode.Query().Where(proxynode.ID(int64(id))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return err
	}
	referenceCount, err := r.data.db.ProxyNode.Query().Where(proxynode.TargetNodeIDEQ(int64(id))).Count(ctx)
	if err != nil {
		return err
	}
	if referenceCount > 0 {
		return fmt.Errorf("node %d is referenced by %d front nodes", id, referenceCount)
	}
	if err := r.data.db.ProxyNode.DeleteOneID(int64(id)).Exec(ctx); err != nil {
		return err
	}
	if err := r.ClearNodeCache(ctx, []int{int(node.ServerID)}); err != nil {
		r.log.Warnf("Failed to clear node cache for server %d after deleting node %d: %v", node.ServerID, id, err)
	}
	return nil
}

func (r *adminNodeRepo) FilterNodeList(ctx context.Context, page, size int32, search string, nodeGroupID *int64) (int32, []*serverbiz.Node, error) {
	query := r.data.db.ProxyNode.Query()
	if nodeGroupID != nil && *nodeGroupID > 0 {
		query = query.Where(func(s *sql.Selector) {
			s.Where(jsonArrayContainsInt64(r.data.databaseDriver, proxynode.FieldNodeGroupIds, *nodeGroupID))
		})
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(func(s *sql.Selector) {
			s.Where(sql.Or(
				sql.ContainsFold(s.C(proxynode.FieldName), search),
				sql.ContainsFold(s.C(proxynode.FieldAddress), search),
				sql.ContainsFold(s.C(proxynode.FieldTags), search),
				numericColumnLike(r.data.databaseDriver, proxynode.FieldPort, searchPattern),
			))
		})
	}
	total, err := query.Count(ctx)
	if err != nil {
		return 0, nil, err
	}
	list, err := query.Order(ent.Asc(proxynode.FieldSort)).Limit(int(size)).Offset(int((page - 1) * size)).All(ctx)
	if err != nil {
		return 0, nil, err
	}
	nodes := make([]*serverbiz.Node, 0, len(list))
	for _, item := range list {
		nodes = append(nodes, adminNodeToBiz(item))
	}
	return int32(total), nodes, nil
}

func (r *adminNodeRepo) ToggleNodeStatus(ctx context.Context, id int, enable *bool) (*serverbiz.Node, error) {
	if enable == nil {
		return nil, fmt.Errorf("enable parameter is required")
	}
	updated, err := r.data.db.ProxyNode.UpdateOneID(int64(id)).SetEnabled(*enable).Save(ctx)
	if err != nil {
		return nil, err
	}
	return adminNodeToBiz(updated), nil
}

func (r *adminNodeRepo) QueryNodeTags(ctx context.Context) ([]string, error) {
	nodes, err := r.data.db.ProxyNode.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	tagSet := make(map[string]bool)
	for _, node := range nodes {
		for _, tag := range tool.StringToStringSlice(node.Tags) {
			if tag != "" {
				tagSet[tag] = true
			}
		}
	}
	uniqueTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		uniqueTags = append(uniqueTags, tag)
	}
	return uniqueTags, nil
}

func (r *adminNodeRepo) GetServerProtocols(ctx context.Context, id int) ([]*servermodel.Protocol, error) {
	server, err := r.data.db.ProxyServer.Query().Where(proxyserver.ID(int64(id))).Only(ctx)
	if err != nil {
		return nil, err
	}
	return servermodel.UnmarshalProtocols(server.Protocol)
}

func (r *adminNodeRepo) GetNodeByID(ctx context.Context, id int64) (*serverbiz.Node, error) {
	node, err := r.data.db.ProxyNode.Query().Where(proxynode.ID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return adminNodeToBiz(node), nil
}

func (r *adminNodeRepo) CountFrontNodesByTarget(ctx context.Context, targetNodeID int64) (int, error) {
	return r.data.db.ProxyNode.Query().Where(proxynode.TargetNodeIDEQ(targetNodeID)).Count(ctx)
}

func (r *adminNodeRepo) ResetNodeSort(ctx context.Context, sortItems []*serverbiz.SortItem) error {
	if len(sortItems) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(sortItems))
	for _, item := range sortItems {
		ids = append(ids, item.ID)
	}
	nodes, err := r.data.db.ProxyNode.Query().Where(proxynode.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	valid := make(map[int64]struct{}, len(nodes))
	for _, item := range nodes {
		valid[item.ID] = struct{}{}
	}
	for _, item := range sortItems {
		if _, ok := valid[item.ID]; !ok {
			continue
		}
		if _, err := r.data.db.ProxyNode.Update().Where(proxynode.ID(item.ID)).SetSort(int32(item.Sort)).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminNodeRepo) ClearNodeCache(ctx context.Context, serverIDs []int) error {
	if len(serverIDs) == 0 {
		return nil
	}
	serverIDsAny := make([]any, len(serverIDs))
	for i, id := range serverIDs {
		serverIDsAny[i] = id
	}
	nodes, err := r.data.db.ProxyNode.Query().Where(func(s *sql.Selector) { s.Where(sql.In(s.C(proxynode.FieldServerID), serverIDsAny...)) }).All(ctx)
	if err != nil {
		r.log.Errorf("Failed to query nodes for cache clearing: %v", err)
		return err
	}
	for _, node := range nodes {
		statusKey := fmt.Sprintf(StatusCacheKey, int64(node.ServerID))
		if err := r.data.rdb.Del(ctx, statusKey).Err(); err != nil {
			r.log.Warnf("Failed to delete status cache for server %d: %v", node.ServerID, err)
		}
		if node.Protocol != "" {
			onlineKey := onlineUserSubscribeCacheKey(int64(node.ServerID), node.Protocol, 0)
			if err := r.data.rdb.Del(ctx, onlineKey).Err(); err != nil {
				r.log.Warnf("Failed to delete online user cache for server %d protocol %s: %v", node.ServerID, node.Protocol, err)
			}
			onlinePortKey := onlineUserSubscribeCacheKey(int64(node.ServerID), node.Protocol, node.Port)
			if err := r.data.rdb.Del(ctx, onlinePortKey).Err(); err != nil {
				r.log.Warnf("Failed to delete online user cache for server %d protocol %s port %d: %v", node.ServerID, node.Protocol, node.Port, err)
			}
		}
	}
	legacyServerIDs := make([]int64, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		if serverID > 0 {
			legacyServerIDs = append(legacyServerIDs, int64(serverID))
		}
	}
	if err := ClearLegacyServerCachesByServerIDs(ctx, r.data.rdb, legacyServerIDs); err != nil {
		r.log.Warnf("Failed to clear legacy server cache for servers %v: %v", serverIDs, err)
	}
	r.log.Infof("Cleared cache for %d nodes across %d servers", len(nodes), len(serverIDs))
	return nil
}
