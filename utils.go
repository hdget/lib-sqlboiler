package sqlboiler

import (
	"strings"
)

func escape(s, quote string) string {
	// 1. 分割字符串
	parts := strings.Split(s, ".")
	// 2. 使用Builder高效构建
	var builder strings.Builder
	builder.Grow(len(s) + len(parts)*2) // 关键：预分配内存避免多次扩容
	for i, p := range parts {
		if i > 0 {
			builder.WriteString(".") // 直接写入点号+双引号组合
		}
		builder.WriteString(quote)
		builder.WriteString(p)
		builder.WriteString(quote)
	}
	return builder.String()
}
