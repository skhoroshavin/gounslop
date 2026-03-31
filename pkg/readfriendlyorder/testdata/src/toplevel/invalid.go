package toplevel

func helperBad() int { return 1 } // want `Place helper "helperBad" below the top-level symbol "ExportedBad" that depends on it.`

func ExportedBad() int {
	return helperBad()
}
