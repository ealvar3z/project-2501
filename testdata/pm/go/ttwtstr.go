package twtstr

import "strings"

func find(s, substr string, start int) int {
	if start < 0 || start > len(s) {
		return -1
	}
	i := strings.Index(s[start:], substr)
	if i < 0 {
		return -1
	}
	return start + i
}

func rfind(s string, ch byte, start, last int) int {
	if start < 0 {
		start = 0
	}
	if last >= len(s) {
		last = len(s) - 1
	}
	for i := last; i >= start; i-- {
		if s[i] == ch {
			return i
		}
	}
	return -1
}

func containsToken(s, token string) bool {
	for _, field := range strings.Fields(s) {
		if field == token {
			return true
		}
	}
	return token == "" && len(s) > 0 && strings.TrimLeft(s, " \t\r\n\f") != s
}

func strip(s string, leading, trailing bool, cutset string) string {
	if leading {
		s = strings.TrimLeft(s, cutset)
	}
	if trailing {
		s = strings.TrimRight(s, cutset)
	}
	return s
}

func getContentTypeAttr(s, name string) string {
	for _, part := range strings.Split(s, ";")[1:] {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) != name {
			continue
		}
		v = strings.TrimSpace(v)
		return strings.Trim(v, `"`)
	}
	return ""
}

func setContentTypeAttr(s, name, value string) string {
	parts := strings.Split(s, ";")
	for i := 1; i < len(parts); i++ {
		k, _, ok := strings.Cut(parts[i], "=")
		if ok && strings.TrimSpace(k) == name {
			parts[i] = k + "=" + value
			return strings.Join(parts, ";")
		}
	}
	return s + "; " + name + "=" + value
}
