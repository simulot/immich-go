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
