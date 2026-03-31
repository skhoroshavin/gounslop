package eager

// Valid: helper is used eagerly in a package-level var
var buildValue = func() int { return 1 }

var cached = buildValue()
