package sqlboiler

import (
	"fmt"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"strings"
)

type OrderByHelper struct {
	tokens []string
	quote  string
}

func (o *OrderByHelper) Desc(col string) *OrderByHelper {
	o.tokens = append(o.tokens, fmt.Sprintf("%s DESC", escape(col, o.quote)))
	return o
}

func (o *OrderByHelper) Asc(col string) *OrderByHelper {
	o.tokens = append(o.tokens, fmt.Sprintf("%s ASC", escape(col, o.quote)))
	return o
}

func (o OrderByHelper) Output(args ...any) qm.QueryMod {
	return qm.OrderBy(strings.Join(o.tokens, ","), args...)
}
