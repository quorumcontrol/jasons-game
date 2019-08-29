package stringslice

// Index returns the first index of the target string t, or -1 if no match is found.
func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

// Include returns true if the target string t is in the slice.
func Include(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

// Equal returns true if two string slices are equal
func Equal(vs []string, vs2 []string) bool {
	if len(vs) != len(vs2) {
		return false
	}

	return All(vs, func(s string) bool {
		return Include(vs2, s)
	})
}

// Any returns true if one of the strings in the slice satisfies the predicate f.
func Any(vs []string, f func(string) bool) bool {
	for _, v := range vs {
		if f(v) {
			return true
		}
	}
	return false
}

// All returns true if all of the strings in the slice satisfy the predicate f.
func All(vs []string, f func(string) bool) bool {
	for _, v := range vs {
		if !f(v) {
			return false
		}
	}
	return true
}

// Reverse modifies the slice to be in reverse order
func Reverse(vs []string) {
	for i := len(vs)/2 - 1; i >= 0; i-- {
		opp := len(vs) - 1 - i
		vs[i], vs[opp] = vs[opp], vs[i]
	}
}
