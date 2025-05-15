package sqlboiler

import (
	"fmt"
	"github.com/elliotchance/pie/v2"
	"strings"
)

const (
	MysqlIdentifierQuote = "`"
	PsqlIdentifierQuote  = "\""
)

func MysqlEscape(s string) string {
	return escape(s, MysqlIdentifierQuote)
}

func PsqlEscape(s string) string {
	return escape(s, PsqlIdentifierQuote)
}

func escape(s, quote string) string {
	escaped := pie.Map(strings.Split(s, "."), func(v string) string {
		return quoteString(v, quote)
	})
	return strings.Join(escaped, ".")
}

func quoteString(s, quote string) string {
	return fmt.Sprintf("%s%s%s", quote, s, quote)
}
