package task

// GroupBy groups items by key(item), preserving order within each group.
func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T {
	out := map[K][]T{}
	for _, it := range items {
		k := key(it)
		out[k] = append(out[k], it)
	}
	return out
}
