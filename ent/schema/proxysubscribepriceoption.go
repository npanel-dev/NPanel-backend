package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// ProxySubscribePriceOption holds the schema definition for the ProxySubscribePriceOption entity.
type ProxySubscribePriceOption struct {
	ent.Schema
}

func (ProxySubscribePriceOption) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "subscribe_price_option"},
		entsql.WithComments(true),
	}
}

func (ProxySubscribePriceOption) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Positive().Comment("价格档位ID"),
		field.Int64("subscribe_id").Comment("订阅套餐ID"),
		field.String("name").MaxLen(255).Default("").Comment("价格档位名称"),
		field.String("duration_unit").MaxLen(32).Default("Month").Comment("时长单位"),
		field.Int64("duration_value").Default(1).Comment("时长数值"),
		field.Int64("price").Default(0).Comment("售价"),
		field.Int64("original_price").Default(0).Comment("原价"),
		field.Int32("inventory").Default(-1).Comment("档位库存"),
		field.Bool("show").Default(true).Comment("是否显示"),
		field.Bool("sell").Default(true).Comment("是否售卖"),
		field.Bool("is_default").Default(false).Comment("默认档位"),
		field.Int32("sort").Default(0).Comment("排序"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("创建时间"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("更新时间"),
	}
}

func (ProxySubscribePriceOption) Edges() []ent.Edge {
	return nil
}
