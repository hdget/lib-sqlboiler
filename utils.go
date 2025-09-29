package sqlboiler

import "strings"

func escape(s, quote string, splitWord ...bool) string {
	var builder strings.Builder
	if len(splitWord) > 0 && splitWord[0] {
		// 1. 分割字符串
		parts := strings.Split(s, ".")
		// 2. 使用Builder高效构建
		builder.Grow(len(s) + len(parts)*2) // 关键：预分配内存避免多次扩容
		for i, p := range parts {
			if i > 0 {
				builder.WriteString(".") // 直接写入点号+双引号组合
			}
			builder.WriteString(quote)
			builder.WriteString(p)
			builder.WriteString(quote)
		}
	} else {
		builder.Grow(len(s) + 2)
		builder.WriteString(quote)
		builder.WriteString(s)
		builder.WriteString(quote)
	}
	return builder.String()
}
