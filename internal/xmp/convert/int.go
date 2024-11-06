package convert

import "strconv"

func IntToString(i int) string {
	return strconv.Itoa(i)
}

func StringToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
