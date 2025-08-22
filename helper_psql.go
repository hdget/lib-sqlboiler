package sqlboiler

import (
	"fmt"
	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

type psqlHelper struct {
	*baseHelper
}

const (
	psqlIdentifierQuote = "\""
)

func Psql() SQLHelper {
	return &psqlHelper{
		&baseHelper{
			identifierQuote: psqlIdentifierQuote,
			functionIfNull:  "COALESCE",
		},
	}
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
