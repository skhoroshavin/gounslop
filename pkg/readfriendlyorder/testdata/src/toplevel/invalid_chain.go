package toplevel

// Invalid: unexported C is above exported B that uses it
func cHelper() int { return 42 } // want `Place helper "cHelper" below the top-level symbol "BExported" that depends on it.`

func BExported() int {
	return 1 - cHelper()
}

func AExported() int {
	return 1 + BExported()
}
