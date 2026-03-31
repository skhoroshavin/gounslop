package cyclic

// Valid: 3-way cycle between unexported helpers
func aFunc() int { return bFunc() }
func bFunc() int { return cFunc() }
func cFunc() int { return aFunc() }

func Main() int { return aFunc() }
