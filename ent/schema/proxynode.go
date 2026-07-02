package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ProxyNode holds the schema definition for the ProxyNode entity.
type ProxyNode struct {
	ent.Schema
}

func (ProxyNode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "nodes"},
		entsql.WithComments(true),
	}
}

func (ProxyNode) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Comment("节点ID"),
		field.String("name").MaxLen(100).Default("").Comment("节点名称"),
		field.String("tags").MaxLen(255).Default("").Comment("标签"),
		field.Uint16("port").Default(0).Comment("连接端口"),
		field.String("address").MaxLen(255).Default("").Comment("连接地址"),
		field.Int64("server_id").Default(0).Comment("服务器ID"),
		field.String("protocol").MaxLen(100).Default("").Comment("协议"),
		field.Bool("enabled").Default(true).Comment("启用"),
		field.String("node_type").MaxLen(20).Default("landing").Comment("节点类型"),
		field.Bool("is_hidden").Default(false).Comment("是否隐藏"),
		field.Int64("target_node_id").Optional().Nillable().Comment("入口节点绑定的落地节点ID"),
		field.Int32("sort").Default(0).Comment("排序"),
		field.JSON("node_group_ids", []int64{}).Optional().Comment("节点组ID列表"),
		field.Time("created_at").Default(time.Now).Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxyNode) Edges() []ent.Edge {
	return nil
}
