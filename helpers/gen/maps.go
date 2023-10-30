package gen

func MapKeys[K comparable, T any](m map[K]T) []K {
	r := make([]K, len(m))
	i := 0
	for k := range m {
		r[i] = k
		i++
	}
	return r
}

func MapFilterKeys[K comparable, T any](m map[K]T, f func(i T) bool) []K {
	r := make([]K, 0, len(m))
	for k, v := range m {
		if f(v) {
			r = append(r, k)
		}
	}
	return r
}
