package sqlboiler

import (
	"github.com/hdget/common/protobuf"
	"github.com/hdget/utils/paginator"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"time"
)

type SQLHelper interface {
	IfNull(column string, defaultValue any, args ...string) string
	IfNullWithColumn(column string, anotherColumn string, args ...string) string
	JsonValue(jsonColumn string, jsonKey string, defaultValue any) qm.QueryMod
	JsonValueCompare(jsonColumn string, jsonKey string, operator string, compareValue any) qm.QueryMod
	SUM(col string, args ...string) string
	AsAliasColumn(alias, colName string) string
	InnerJoin(joinTable string, args ...string) *JoinClauseBuilder
	LeftJoin(joinTable string, args ...string) *JoinClauseBuilder
	OrderBy() *OrderByHelper
}

type baseHelper struct {
	quote string //  identifier quote
}

func (b baseHelper) InnerJoin(joinTable string, asTable ...string) *JoinClauseBuilder {
	return innerJoin(b.quote, joinTable, asTable...)
}

func (b baseHelper) LeftJoin(joinTable string, asTable ...string) *JoinClauseBuilder {
	return leftJoin(b.quote, joinTable, asTable...)
}

// OrderBy OrderBy字段加入desc
func (b baseHelper) OrderBy() *OrderByHelper {
	return &OrderByHelper{tokens: make([]string, 0), quote: b.quote}
}

// GetLimitQueryMods 获取Limit相关QueryMods
func GetLimitQueryMods(list *protobuf.ListParam) []qm.QueryMod {
	p := getPaginator(list)
	return []qm.QueryMod{qm.Offset(int(p.Offset)), qm.Limit(int(p.PageSize))}
}

// WithUpdateTime 除了cols中的会更新以外还会更新更新时间字段
func WithUpdateTime(cols map[string]any, args ...string) map[string]any {
	updateColName := "updated_at"
	if len(args) > 0 {
		updateColName = args[0]
	}

	cols[updateColName] = time.Now().In(boil.GetLocation())
	return cols
}

func GetDB(args ...boil.Executor) boil.Executor {
	if len(args) > 0 && args[0] != nil {
		return args[0]
	}
	return boil.GetDB()
}

func getPaginator(list *protobuf.ListParam) paginator.Paginator {
	if list == nil {
		return paginator.DefaultPaginator
	}
	return paginator.New(list.Page, list.PageSize)
}
