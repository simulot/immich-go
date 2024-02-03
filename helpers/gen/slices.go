package gen

func DeleteItem[T comparable](s []T, item T) []T {
	r := make([]T, 0, len(s))
	for i := range s {
		if s[i] != item {
			r = append(r, s[i])
		}
	}
	return r
}

func Filter[T any](s []T, f func(i T) bool) []T {
	var r []T
	for _, i := range s {
		if f(i) {
			r = append(r, i)
		}
	}
	return r
}
