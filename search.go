package hades

import (
	"fmt"
	"strings"

	"xorm.io/builder"
)

type joinType string

const (
	joinTypeInner joinType = "inner"
	joinTypeLeft  joinType = "left"
)

type join struct {
	joinType  joinType
	joinTable string
	joinCond  string
}

type Search struct {
	groups    []string
	orders    []string
	orderArgs []any
	joins     []join
	offset    *int64
	limit     *int64
}

func (s Search) GroupBy(group string) Search {
	s.groups = append(s.groups, group)
	return s
}

// OrderBy appends an ORDER BY expression, which may contain `?`
// placeholders bound to args. Since Apply emits ORDER BY after the WHERE
// clause, callers executing the query must bind OrderArgs after the
// WHERE clause's arguments (Select and ExecWithSearch do this).
func (s Search) OrderBy(order string, args ...any) Search {
	s.orders = append(s.orders, order)
	s.orderArgs = append(s.orderArgs, args...)
	return s
}

// OrderArgs returns the placeholder arguments accumulated from OrderBy
// calls, in the order their expressions appear in the query.
func (s Search) OrderArgs() []any {
	return s.orderArgs
}

func (s Search) Limit(limit int64) Search {
	s.limit = &limit
	return s
}

func (s Search) Offset(offset int64) Search {
	s.offset = &offset
	return s
}

func (s Search) InnerJoin(joinTable string, joinCond string) Search {
	s.joins = append(s.joins, join{
		joinType:  joinTypeInner,
		joinTable: joinTable,
		joinCond:  joinCond,
	})
	return s
}

func (s Search) LeftJoin(joinTable string, joinCond string) Search {
	s.joins = append(s.joins, join{
		joinType:  joinTypeLeft,
		joinTable: joinTable,
		joinCond:  joinCond,
	})
	return s
}

func (s Search) Apply(sql string) string {
	if len(s.groups) > 0 {
		sql = fmt.Sprintf("%s GROUP BY %s", sql, strings.Join(s.groups, ", "))
	}

	if len(s.orders) > 0 {
		sql = fmt.Sprintf("%s ORDER BY %s", sql, strings.Join(s.orders, ", "))
	}

	if s.limit != nil || s.offset != nil {
		var limit int64 = -1
		if s.limit != nil {
			limit = *s.limit
		}

		// offset must appear without limit,
		// and a negative limit means no limit.
		// see https://www.sqlite.org/lang_select.html#limitoffset
		sql = fmt.Sprintf("%s LIMIT %d", sql, limit)

		if s.offset != nil {
			sql = fmt.Sprintf("%s OFFSET %d", sql, *s.offset)
		}
	}

	return sql
}

func (s Search) ApplyJoins(b *builder.Builder) {
	for _, j := range s.joins {
		switch j.joinType {
		case joinTypeInner:
			b.InnerJoin(j.joinTable, j.joinCond)
		case joinTypeLeft:
			b.LeftJoin(j.joinTable, j.joinCond)
		}
	}
}

func (s Search) String() string {
	return s.Apply("")
}
