package codec

import "strings"

func filter[T any](elements []T, keepIfTrue func(element T) bool) (out []T) {
	for _, element := range elements {
		if keepIfTrue(element) {
			out = append(out, element)
		}
	}

	return
}

func Not[E any, T func(E) bool](original T) T {
	return func(element E) bool {
		return !original(element)
	}
}

func StringHasPrefix(prefix string) func(string) bool {
	return func(candidate string) bool {
		return strings.HasPrefix(candidate, prefix)
	}
}
