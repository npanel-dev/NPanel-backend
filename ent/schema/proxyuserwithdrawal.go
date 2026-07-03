package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ProxyUserWithdrawal holds the schema definition for the ProxyUserWithdrawal entity.
type ProxyUserWithdrawal struct {
	ent.Schema
}

func (ProxyUserWithdrawal) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_withdrawal"},
		entsql.WithComments(true),
	}
}

func (ProxyUserWithdrawal) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("提现ID"),
		field.Int64("user_id").Comment("用户ID"),
		field.Int64("amount").Comment("提现金额"),
		field.String("method").MaxLen(32).Optional().Nillable().Comment("提现方式"),
		field.Text("content").Optional().Nillable().Comment("提现内容"),
		field.Int8("status").Default(0).Comment("提现状态"),
		field.String("reason").MaxLen(500).Optional().Nillable().Comment("拒绝原因"),
		field.Time("processed_at").Optional().Nillable().Comment("处理时间"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxyUserWithdrawal) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", ProxyUser.Type).Ref("withdrawals").Field("user_id").Unique().Required(),
	}
}
