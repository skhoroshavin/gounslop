package cyclic

// Valid: cyclic helpers are exempt from reordering
func parseExpression() int {
	return parseAtom()
}

func parseAtom() int {
	return parseExpression()
}

func Parse() int {
	return parseExpression()
}
