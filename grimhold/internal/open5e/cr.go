package open5e

import "strconv"

// CRValue parses a challenge rating such as "1/4" or "2".
func CRValue(s string) float64 {
	s = strconvTrim(s)
	if s == "" {
		return 0
	}
	if n, d, ok := cut(s, "/"); ok {
		nf, err1 := strconv.ParseFloat(n, 64)
		df, err2 := strconv.ParseFloat(d, 64)
		if err1 != nil || err2 != nil || df == 0 {
			return 0
		}
		return nf / df
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func strconvTrim(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func cut(s, sep string) (string, string, bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return "", "", false
}
