package data

import (
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/npanel-dev/NPanel-backend/internal/database"
)

func jsonArrayContainsInt64(driver database.Driver, column string, value int64) *entsql.Predicate {
	jsonValue := fmt.Sprintf("[%d]", value)
	return entsql.P(func(builder *entsql.Builder) {
		if driver == database.Postgres {
			builder.Ident(column)
			builder.WriteString(" @> ")
			builder.Arg(jsonValue)
			builder.WriteString("::jsonb")
			return
		}
		builder.WriteString("JSON_CONTAINS(")
		builder.Ident(column)
		builder.WriteString(", ")
		builder.Arg(jsonValue)
		builder.WriteString(")")
	})
}

func csvContains(driver database.Driver, column string, value any) *entsql.Predicate {
	return entsql.P(func(builder *entsql.Builder) {
		if driver == database.Postgres {
			builder.Arg(fmt.Sprint(value))
			builder.WriteString(" = ANY(string_to_array(")
			builder.Ident(column)
			builder.WriteString(", ','))")
			return
		}
		builder.WriteString("FIND_IN_SET(")
		builder.Arg(value)
		builder.WriteString(", ")
		builder.Ident(column)
		builder.WriteString(")")
	})
}

func numericColumnLike(driver database.Driver, column, pattern string) *entsql.Predicate {
	return entsql.P(func(builder *entsql.Builder) {
		builder.WriteString("CAST(")
		builder.Ident(column)
		if driver == database.Postgres {
			builder.WriteString(" AS TEXT) LIKE ")
		} else {
			builder.WriteString(" AS CHAR) LIKE ")
		}
		builder.Arg(pattern)
	})
}
