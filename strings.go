package bunquery

import "strings"

func PascalToDelimited(s string, delim string) string {
	var b strings.Builder
	b.Grow(len(s) + 3)
	low := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			if low > 0 {
				b.WriteString(delim)
				low = 0
			}
			b.WriteByte(c + ('a' - 'A'))
		} else {
			low = 1
			b.WriteByte(c)
		}
	}
	return b.String()
}
