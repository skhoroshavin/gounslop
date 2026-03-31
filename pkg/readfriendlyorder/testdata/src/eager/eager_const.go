package eager

// Valid: MAX is used eagerly in package-level const/var expressions
const maxVal = 3

var total = maxVal + 1

func useMax() int {
	return maxVal
}
