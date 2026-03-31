package toplevel

const maxCount = 3 // want `Place constant "maxCount" below the top-level symbol "Limit" that uses it.`

func Limit() int {
	return maxCount
}
