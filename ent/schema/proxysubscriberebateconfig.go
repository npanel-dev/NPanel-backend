package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProxySubscribeRebateConfig holds the schema definition for the ProxySubscribeRebateConfig entity.
// 存储按"套餐 + 价格档位（月卡/季卡/半年卡/年卡...）"维度细化的返佣比例配置。
// price_option_id = 0 表示该套餐的整体默认档（用于兜底未单独配置的档位）。
type ProxySubscribeRebateConfig struct {
	ent.Schema
}

func (ProxySubscribeRebateConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscribe_rebate_config"},
		entsql.WithComments(true),
	}
}

func (ProxySubscribeRebateConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("返佣配置ID"),
		field.Int64("subscribe_id").Comment("订阅套餐ID"),
		field.Int64("price_option_id").Default(0).Comment("价格档位ID，0表示套餐整体默认"),
		field.Int8("referral_percentage").Default(0).Comment("返佣百分比 0-100"),
		field.Bool("enabled").Default(true).Comment("是否启用，false表示显式关闭该档返佣（不回退兜底）"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxySubscribeRebateConfig) Indexes() []ent.Index {
	return []ent.Index{
		// 一个套餐的同一个价格档位最多一条配置
		index.Fields("subscribe_id", "price_option_id").Unique(),
		index.Fields("subscribe_id"),
	}
}

func (ProxySubscribeRebateConfig) Edges() []ent.Edge {
	return nil
}
