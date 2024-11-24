package xmpsidecar

import "strings"

func BoolToString(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func StringToBool(s string) bool {
	return strings.ToLower(s) == "true"
}
