package toplevel

// Valid: exported function uses unexported helper defined below
func Exported() int {
	return helper() + constant
}

func helper() int { return 1 }

const constant = 42
