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

func Difference(a []string, b []string) []string {
	diff := make([]string, 0, len(a))
	for _, e := range a {
		if !Contains(b, e) {
			diff = append(diff, e)
		}
	}

	return diff
}

func Index(a []string, e string) int {
	for i, elem := range a {
		if e == elem {
			return i
		}
	}
	return -1
}

func Contains(a []string, e string) bool {
	return Index(a, e) != -1
}
