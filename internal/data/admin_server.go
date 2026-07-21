package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxyserver"
	"github.com/npanel-dev/NPanel-backend/ent/proxyusersubscribe"
	serverbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/server"
	servermodel "github.com/npanel-dev/NPanel-backend/internal/model/server"
	"github.com/redis/go-redis/v9"
)

type adminServerRepo struct {
	data *Data
	log  *log.Helper
}

func NewAdminServerRepo(data *Data, logger log.Logger) serverbiz.ServerRepo {
	return &adminServerRepo{data: data, log: log.NewHelper(logger)}
}

func (r *adminServerRepo) CreateServer(ctx context.Context, server *serverbiz.Server) (*serverbiz.Server, error) {
	protocolsJSON, err := servermodel.MarshalProtocols(server.Protocols)
	if err != nil {
		return nil, err
	}
	created, err := r.data.db.ProxyServer.Create().SetName(server.Name).SetCountry(server.Country).SetCity(server.City).SetServerAddr(server.Address).SetSort(int32(server.Sort)).SetProtocol(protocolsJSON).Save(ctx)
	if err != nil {
		return nil, err
	}
	protocols, _ := servermodel.UnmarshalProtocols(created.Protocol)
	lastReportedAt := int64(0)
	if created.LastReportedAt != nil {
		lastReportedAt = created.LastReportedAt.UnixMilli()
	}
	return &serverbiz.Server{ID: int64(created.ID), Name: created.Name, Country: created.Country, City: created.City, Address: created.ServerAddr, Sort: int(created.Sort), Protocols: protocols, LastReportedAt: lastReportedAt, CreatedAt: created.CreatedAt.UnixMilli(), UpdatedAt: created.UpdatedAt.UnixMilli()}, nil
}

func (r *adminServerRepo) UpdateServer(ctx context.Context, server *serverbiz.Server) (*serverbiz.Server, error) {
	protocolsJSON, err := servermodel.MarshalProtocols(server.Protocols)
	if err != nil {
		return nil, err
	}
	updated, err := r.data.db.ProxyServer.UpdateOneID(server.ID).SetName(server.Name).SetCountry(server.Country).SetCity(server.City).SetServerAddr(server.Address).SetSort(int32(server.Sort)).SetProtocol(protocolsJSON).Save(ctx)
	if err != nil {
		return nil, err
	}
	protocols, _ := servermodel.UnmarshalProtocols(updated.Protocol)
	lastReportedAt := int64(0)
	if updated.LastReportedAt != nil {
		lastReportedAt = updated.LastReportedAt.UnixMilli()
	}
	return &serverbiz.Server{ID: int64(updated.ID), Name: updated.Name, Country: updated.Country, City: updated.City, Address: updated.ServerAddr, Sort: int(updated.Sort), Protocols: protocols, LastReportedAt: lastReportedAt, CreatedAt: updated.CreatedAt.UnixMilli(), UpdatedAt: updated.UpdatedAt.UnixMilli()}, nil
}

func (r *adminServerRepo) DeleteServer(ctx context.Context, id int) error {
	_, err := r.data.db.ProxyServer.Delete().Where(proxyserver.ID(int64(id))).Exec(ctx)
	return err
}

func (r *adminServerRepo) GetServerByID(ctx context.Context, id int) (*serverbiz.Server, error) {
	server, err := r.data.db.ProxyServer.Query().Where(proxyserver.ID(int64(id))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("server not found or access denied")
		}
		return nil, err
	}
	protocols, _ := servermodel.UnmarshalProtocols(server.Protocol)
	lastReportedAt := int64(0)
	if server.LastReportedAt != nil {
		lastReportedAt = server.LastReportedAt.UnixMilli()
	}
	return &serverbiz.Server{ID: int64(server.ID), Name: server.Name, Country: server.Country, City: server.City, Address: server.ServerAddr, Sort: int(server.Sort), Protocols: protocols, LastReportedAt: lastReportedAt, CreatedAt: server.CreatedAt.UnixMilli(), UpdatedAt: server.UpdatedAt.UnixMilli()}, nil
}

func (r *adminServerRepo) FilterServerList(ctx context.Context, page, size int32, search string) (int32, []*serverbiz.Server, error) {
	query := r.data.db.ProxyServer.Query()
	if search != "" {
		query = query.Where(func(s *sql.Selector) {
			s.Where(sql.Or(sql.ContainsFold(s.C(proxyserver.FieldName), search), sql.ContainsFold(s.C(proxyserver.FieldServerAddr), search)))
		})
	}
	total, err := query.Count(ctx)
	if err != nil {
		return 0, nil, err
	}
	list, err := query.Order(ent.Asc(proxyserver.FieldSort)).Limit(int(size)).Offset(int((page - 1) * size)).All(ctx)
	if err != nil {
		return 0, nil, err
	}
	servers := make([]*serverbiz.Server, 0, len(list))
	for _, item := range list {
		protocols, _ := servermodel.UnmarshalProtocols(item.Protocol)
		lastReportedAt := int64(0)
		if item.LastReportedAt != nil {
			lastReportedAt = item.LastReportedAt.UnixMilli()
		}
		servers = append(servers, &serverbiz.Server{ID: int64(item.ID), Name: item.Name, Country: item.Country, City: item.City, Address: item.ServerAddr, Sort: int(item.Sort), Protocols: protocols, LastReportedAt: lastReportedAt, CreatedAt: item.CreatedAt.UnixMilli(), UpdatedAt: item.UpdatedAt.UnixMilli()})
	}
	return int32(total), servers, nil
}

func (r *adminServerRepo) GetServerProtocols(ctx context.Context, id int) ([]*servermodel.Protocol, error) {
	server, err := r.data.db.ProxyServer.Query().Where(proxyserver.ID(int64(id))).Only(ctx)
	if err != nil {
		return nil, err
	}
	return servermodel.UnmarshalProtocols(server.Protocol)
}

func (r *adminServerRepo) ResetServerSort(ctx context.Context, sortItems []*serverbiz.SortItem) error {
	if len(sortItems) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(sortItems))
	for _, item := range sortItems {
		ids = append(ids, item.ID)
	}
	servers, err := r.data.db.ProxyServer.Query().Where(proxyserver.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	valid := make(map[int64]struct{}, len(servers))
	for _, item := range servers {
		valid[item.ID] = struct{}{}
	}
	for _, item := range sortItems {
		if _, ok := valid[item.ID]; !ok {
			continue
		}
		if _, err := r.data.db.ProxyServer.Update().Where(proxyserver.ID(item.ID)).SetSort(int32(item.Sort)).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminServerRepo) GetServerStatus(ctx context.Context, serverID int) (*serverbiz.ServerResourceStatus, error) {
	key := fmt.Sprintf(StatusCacheKey, int(serverID))
	result, err := r.data.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if result == "" {
		return nil, nil
	}
	var status serverbiz.ServerResourceStatus
	if err := json.Unmarshal([]byte(result), &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (r *adminServerRepo) GetOnlineUsers(ctx context.Context, serverID int64, protocol string, port uint16) (map[int64][]string, error) {
	key := onlineUserSubscribeCacheKey(serverID, protocol, port)
	result, err := r.data.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return map[int64][]string{}, nil
		}
		return nil, err
	}
	if result == "" {
		return map[int64][]string{}, nil
	}
	var onlineUsers map[int64][]string
	if err := json.Unmarshal([]byte(result), &onlineUsers); err != nil {
		return nil, err
	}
	for subscribeID, ips := range onlineUsers {
		onlineUsers[subscribeID] = dedupeStringSlicePreserveOrder(ips)
	}
	return onlineUsers, nil
}

func (r *adminServerRepo) GetOnlineUsersByServer(ctx context.Context, serverID int64) (map[string]map[int64][]string, error) {
	if r.data == nil || r.data.rdb == nil {
		return map[string]map[int64][]string{}, nil
	}

	pattern := fmt.Sprintf("node:online:subscribe:%d:*", serverID)
	result := make(map[string]map[int64][]string)
	var cursor uint64

	for {
		keys, nextCursor, err := r.data.rdb.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			if err == redis.Nil {
				return result, nil
			}
			return nil, err
		}
		for _, key := range keys {
			protocol := strings.TrimPrefix(key, fmt.Sprintf("node:online:subscribe:%d:", serverID))
			if protocol == key {
				continue
			}
			raw, err := r.data.rdb.Get(ctx, key).Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				return nil, err
			}
			if raw == "" {
				continue
			}
			var onlineUsers map[int64][]string
			if err := json.Unmarshal([]byte(raw), &onlineUsers); err != nil {
				continue
			}
			for subscribeID, ips := range onlineUsers {
				onlineUsers[subscribeID] = dedupeStringSlicePreserveOrder(ips)
			}
			result[protocol] = onlineUsers
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return result, nil
}

func dedupeStringSlicePreserveOrder(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (r *adminServerRepo) GetUserSubscribeInfo(ctx context.Context, subscribeID int) (*serverbiz.UserSubscribeInfo, error) {
	userSubscribe, err := r.data.db.ProxyUserSubscribe.Query().Where(proxyusersubscribe.ID(int64(subscribeID))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("subscribe %d not found", subscribeID)
		}
		return nil, err
	}
	subscribeName := ""
	if userSubscribe.SubscribeID != 0 {
		subscribeEntity, err := r.data.db.ProxySubscribe.Get(ctx, userSubscribe.SubscribeID)
		if err == nil && subscribeEntity != nil {
			subscribeName = subscribeEntity.Name
		}
	}
	var download, upload int64
	if userSubscribe.Download != nil {
		download = int64(*userSubscribe.Download)
	}
	if userSubscribe.Upload != nil {
		upload = int64(*userSubscribe.Upload)
	}
	var expireTime int64
	if userSubscribe.ExpireTime != nil {
		expireTime = userSubscribe.ExpireTime.UnixMilli()
	}
	return &serverbiz.UserSubscribeInfo{UserID: int64(userSubscribe.UserID), SubscribeID: int64(userSubscribe.ID), Subscribe: subscribeName, Download: download, Upload: upload, ExpireTime: expireTime}, nil
}
