package utils

func Intersection(a []string, b []string) []string {
	max := len(a)
	big, small := a, b
	if len(b) > len(a) {
		max = len(b)
		big, small = b, a
	}

	intersection := make([]string, 0, max)
	// loop over smaller set
	for _, elem := range big {
		if Contains(small, elem) {
			intersection = append(intersection, elem)
		}
	}
	return intersection
}

func Contains(a []string, e string) bool {
	for _, elem := range a {
		if e == elem {
			return true
		}
	}
	return false
}
