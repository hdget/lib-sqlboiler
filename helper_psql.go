package sqlboiler

import (
	"fmt"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/hdget/utils/convert"
	jsonUtils "github.com/hdget/utils/json"
	reflectUtils "github.com/hdget/utils/reflect"
	"reflect"
)

type psqlHelper struct {
	*baseHelper
}

const (
	psqlIdentifierQuote = "\""
)

func Psql() SQLHelper {
	return &psqlHelper{
		&baseHelper{quote: psqlIdentifierQuote},
	}
}

func (psqlHelper) IfNull(column string, defaultValue any, args ...string) string {
	alias := column
	if len(args) > 0 {
		alias = args[0]
	}

	if defaultValue == nil {
		return fmt.Sprintf("COALESCE((%s), '') AS \"%s\"", column, alias)
	}

	v := reflectUtils.Indirect(defaultValue)

	switch vv := reflect.ValueOf(v); vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("COALESCE((%s), %d) AS \"%s\"", column, v, alias)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("COALESCE((%s), %.4f) AS \"%s\"", column, v, alias)
	case reflect.Slice:
		if vv.Type().Elem().Kind() == reflect.Uint8 {
			if jsonUtils.IsEmptyJsonObject(vv.Bytes()) {
				return fmt.Sprintf("COALESCE((%s), '{}') AS \"%s\"", column, alias)
			} else if jsonUtils.IsEmptyJsonArray(vv.Bytes()) {
				return fmt.Sprintf("COALESCE((%s), '[]') AS \"%s\"", column, alias)
			} else {
				return fmt.Sprintf("COALESCE((%s), '%s') AS \"%s\"", column, convert.BytesToString(vv.Bytes()), alias)
			}
		}
	}

	return fmt.Sprintf("COALESCE((%s), '%v') AS \"%s\"", column, defaultValue, alias)
}

func (psqlHelper) IfNullWithColumn(column string, anotherColumn string, args ...string) string {
	alias := column
	if len(args) > 0 {
		alias = args[0]
	}
	return fmt.Sprintf("COALESCE(%s, %s) AS \"%s\"", column, anotherColumn, alias)
}

func (psqlHelper) JsonValue(jsonColumn string, jsonKey string, defaultValue any) qm.QueryMod {
	var template string
	switch v := defaultValue.(type) {
	case string:
		template = fmt.Sprintf("COALESCE(%s->>'%s', '%s') AS %s", jsonColumn, jsonKey, v, jsonKey)
	case int8, int, int32, int64:
		template = fmt.Sprintf("COALESCE((%s->>'%s')::numeric, %d) AS %s", jsonColumn, jsonKey, v, jsonKey)
	case float32, float64:
		template = fmt.Sprintf("COALESCE((%s->>'%s')::numeric, %d) AS %s", jsonColumn, jsonKey, v, jsonKey)
	default:
		return nil
	}
	return qm.Select(template)
}

func (psqlHelper) JsonValueCompare(jsonColumn string, jsonKey string, operator string, compareValue any) qm.QueryMod {
	var template string
	switch v := compareValue.(type) {
	case string:
		template = fmt.Sprintf("(%s->>'%s') %s '%s'", jsonColumn, jsonKey, operator, v)
	case int8, int, int32, int64:
		template = fmt.Sprintf("(%s->>'%s') %s %d", jsonColumn, jsonKey, operator, v)
	case float32, float64:
		template = fmt.Sprintf("(%s->>'%s') %s %f", jsonColumn, jsonKey, operator, v)
	default:
		return nil
	}
	return qm.Where(template)
}

func (h psqlHelper) SUM(col string, args ...string) string {
	return h.IfNull(fmt.Sprintf("SUM(%s)", col), 0, args...)
}

func (psqlHelper) AsAliasColumn(alias, colName string) string {
	return fmt.Sprintf("\"%s\".\"%s\" AS \"%s\".\"%s\"", alias, colName, alias, colName)
}
