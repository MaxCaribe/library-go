package utils

// ToStrings widens a slice of string-kinded values, such as a set of enum
// constants, to plain strings. Needed wherever an API takes []string rather
// than the named type: pgx binds text[] from []string only.
func ToStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = string(value)
	}
	return out
}
