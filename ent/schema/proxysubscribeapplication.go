package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ProxySubscribeApplication holds the schema definition for the ProxySubscribeApplication entity.
type ProxySubscribeApplication struct {
	ent.Schema
}

func (ProxySubscribeApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscribe_application"},
		entsql.WithComments(true),
	}
}

func (ProxySubscribeApplication) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("应用配置ID"),
		field.String("name").MaxLen(255).NotEmpty().Default("").Comment("应用名称"),
		field.Text("icon").SchemaType(map[string]string{dialect.MySQL: "mediumtext"}).Optional().Nillable().Comment("应用图标"),
		field.String("description").MaxLen(255).Optional().Nillable().Comment("应用描述"),
		field.String("scheme").MaxLen(255).Default("").Comment("应用Scheme"),
		field.String("user_agent").MaxLen(255).NotEmpty().Default("").Comment("User Agent"),
		field.Bool("enabled").Default(true).Comment("是否启用（参与客户端 UA 匹配分发；禁用后不参与分发且 UA 不进白名单）"),
		field.Bool("is_default").Default(false).Comment("是否默认应用"),
		field.Text("subscribe_template").SchemaType(map[string]string{dialect.MySQL: "mediumtext"}).Optional().Nillable().Comment("订阅模板"),
		field.String("output_format").MaxLen(50).Default("yaml").Comment("输出格式"),
		field.Text("download_link").Comment("下载链接"),
		field.Time("created_at").Optional().Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Optional().Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxySubscribeApplication) Edges() []ent.Edge {
	return nil
}
