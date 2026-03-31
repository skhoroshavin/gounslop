package toplevel

// Valid: exported A uses exported B, B uses unexported C below
func A() int {
	return 1 + B()
}

func B() int {
	return 1 - c()
}

func c() int {
	return 42
}
