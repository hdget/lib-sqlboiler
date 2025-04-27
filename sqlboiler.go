package sqlboiler

import (
	"fmt"
	"github.com/hdget/common/protobuf"
	"github.com/hdget/utils/paginator"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"time"
)

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

func AsAliasColumn(alias, colName string) string {
	return fmt.Sprintf("`%s`.`%s` AS \"%s.%s\"", alias, colName, alias, colName)
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
