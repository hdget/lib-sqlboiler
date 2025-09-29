package sqlboiler

import (
	"fmt"
	"reflect"
	"time"

	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/hdget/common/protobuf"
	"github.com/hdget/utils/convert"
	jsonUtils "github.com/hdget/utils/json"
	"github.com/hdget/utils/paginator"
	reflectUtils "github.com/hdget/utils/reflect"
)

type SQLHelper interface {
	IfNull(column string, defaultValue any, args ...string) string
	JsonValue(jsonColumn string, jsonKey string, defaultValue any) qm.QueryMod
	JsonValueCompare(jsonColumn string, jsonKey string, operator string, compareValue any) qm.QueryMod
	SUM(col string, args ...string) string
	InnerJoin(joinTable string, args ...string) *JoinClauseBuilder
	LeftJoin(joinTable string, args ...string) *JoinClauseBuilder
	OrderBy() *OrderByHelper
	Quote(s string, splitWord ...bool) string // 默认quote整个字符串，true否则将分割字符串中的单词，每个单词进行quote
	SelectAll(tableColumns any) qm.QueryMod
}

type baseHelper struct {
	identifierQuote string //  identifier quote
	functionIfNull  string // 是否为空的函数
}

func (b baseHelper) SelectAll(tableColumns any) qm.QueryMod {
	v, _ := indirect(reflect.ValueOf(tableColumns))

	selectColumns := make([]string, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i).String()
		selectColumns[i] = fmt.Sprintf("%s as %s", b.Quote(field, true), b.Quote(field))
	}

	return qm.Select(
		selectColumns...,
	)
}

func (b baseHelper) IfNull(column string, defaultValue any, args ...string) string {
	alias := column
	if len(args) > 0 {
		alias = args[0]
	}

	var asValue string
	if defaultValue == nil {
		asValue = "''"
	} else {
		v := reflectUtils.Indirect(defaultValue)
		switch vv := reflect.ValueOf(v); vv.Kind() {
		case reflect.String:
			asValue = vv.String()
			if asValue == "" {
				asValue = "''"
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			asValue = fmt.Sprintf("%d", v)
		case reflect.Float32, reflect.Float64:
			asValue = fmt.Sprintf("%.4f", v)
		case reflect.Slice:
			if vv.Type().Elem().Kind() == reflect.Uint8 {
				if jsonUtils.IsEmptyJsonObject(vv.Bytes()) {
					asValue = "'{}'"
				} else if jsonUtils.IsEmptyJsonArray(vv.Bytes()) {
					asValue = "'[]'"
				} else {
					asValue = fmt.Sprintf("'%s'", convert.BytesToString(vv.Bytes()))
				}
			}
		}
	}

	return fmt.Sprintf("%s(%s, %s) AS %s", b.functionIfNull, b.Quote(column, true), asValue, b.Quote(alias, false))
}

func (b baseHelper) SUM(col string, args ...string) string {
	return b.IfNull(fmt.Sprintf("SUM(%s)", b.Quote(col, true)), 0, args...)
}

func (b baseHelper) Quote(s string, splitWord ...bool) string {
	return escape(s, b.identifierQuote, splitWord...)
}

func (b baseHelper) InnerJoin(joinTable string, asTable ...string) *JoinClauseBuilder {
	return innerJoin(b.identifierQuote, joinTable, asTable...)
}

func (b baseHelper) LeftJoin(joinTable string, asTable ...string) *JoinClauseBuilder {
	return leftJoin(b.identifierQuote, joinTable, asTable...)
}

// OrderBy OrderBy字段加入desc
func (b baseHelper) OrderBy() *OrderByHelper {
	return &OrderByHelper{tokens: make([]string, 0), quote: b.identifierQuote}
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
