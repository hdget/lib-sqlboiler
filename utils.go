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

// capitalize 首字母大写
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}

//func camelToSnake(s string) string {
//	if s == "" {
//		return s
//	}
//
//	// 预分配足够缓冲区（ASCII每个字符1字节）
//	buf := make([]byte, 0, len(s)+5) // +5为额外下划线预留
//
//	var prevLower bool // 记录前一个字符是否是小写
//	for i := 0; i < len(s); i++ {
//		c := s[i]
//		if c >= 'A' && c <= 'Z' { // 仅处理ASCII大写字母
//			if i > 0 && prevLower {
//				buf = append(buf, '_')
//			}
//			buf = append(buf, c+32) // 快速转小写（ASCII码+32）
//			prevLower = false
//		} else {
//			buf = append(buf, c)
//			prevLower = c >= 'a' && c <= 'z'
//		}
//	}
//	return string(buf)
//}
