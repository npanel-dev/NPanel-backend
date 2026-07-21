package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxyPayment holds the schema definition for the ProxyPayment entity.
type ProxyPayment struct {
	ent.Schema
}

func (ProxyPayment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment"},
		entsql.WithComments(true),
	}
}

func (ProxyPayment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Comment("支付ID"),
		field.String("name").MaxLen(100).NotEmpty().Default("").Comment("支付名称"),
		field.String("platform").MaxLen(100).NotEmpty().Comment("支付平台"),
		field.String("icon").MaxLen(255).Default("").Comment("支付图标"),
		field.String("domain").MaxLen(255).Default("").Comment("通知域名"),
		// Balance is an internal payment method and intentionally stores an empty
		// config/token in the legacy and portable baselines.
		field.Text("config").Comment("支付配置"),
		field.Text("description").Optional().Comment("支付描述"),
		field.Uint("fee_mode").Default(0).Comment("费用模式"),
		field.Int64("fee_percent").Default(0).Comment("费用百分比"),
		field.Int64("fee_amount").Default(0).Comment("固定费用金额"),
		field.Int32("sort").Default(0).Comment("排序"),
		field.Bool("enable").Default(false).Comment("是否启用"),
		field.String("token").MaxLen(255).Unique().Comment("支付令牌"),
	}
}

func (ProxyPayment) Edges() []ent.Edge {
	return nil
}

func (ProxyPayment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform"),
		index.Fields("enable"),
	}
}
