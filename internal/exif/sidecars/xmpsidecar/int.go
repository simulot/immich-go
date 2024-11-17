package xmpsidecar

import "strconv"

func IntToString(i int) string {
	return strconv.Itoa(i)
}

func StringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func StringToByte(s string) byte {
	i, _ := strconv.Atoi(s)
	return byte(i)
}
