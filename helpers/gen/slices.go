package gen

func DeleteItem[T comparable](s []T, delete T) []T {
	r := make([]T, 0, len(s))
	for i := range s {
		if s[i] != delete {
			r = append(r, s[i])
		}
	}
	return r
}
