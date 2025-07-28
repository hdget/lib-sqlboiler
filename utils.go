package sqlboiler

import (
	"fmt"
	"github.com/elliotchance/pie/v2"
	"strings"
)

func escape(s, quote string) string {
	escaped := pie.Map(strings.Split(s, "."), func(v string) string {
		return fmt.Sprintf("%s%s%s", quote, s, quote)
	})
	return strings.Join(escaped, ".")
}
