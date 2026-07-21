package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxyRedemptionRecord holds the schema definition for the ProxyRedemptionRecord entity.
type ProxyRedemptionRecord struct {
	ent.Schema
}

func (ProxyRedemptionRecord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "redemption_record"},
		entsql.WithComments(true),
	}
}

func (ProxyRedemptionRecord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("兑换记录ID"),
		field.Int64("redemption_code_id").Comment("兑换码ID"),
		field.Int64("user_id").Comment("用户ID"),
		field.Int64("subscribe_id").Comment("订阅ID"),
		field.String("unit_time").Default("month").MaxLen(50).Comment("时间单位"),
		field.Int32("quantity").Default(1).Comment("数量"),
		field.Time("redeemed_at").Default(time.Now).Immutable().Comment("兑换时间"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
	}
}

func (ProxyRedemptionRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", ProxyUser.Type).Ref("redemption_records").Field("user_id").Unique().Required(),
		edge.From("redemption_code", ProxyRedemptionCode.Type).Ref("records").Field("redemption_code_id").Unique().Required(),
	}
}

func (ProxyRedemptionRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("redemption_code_id"),
	}
}
