package eager

// Valid: constant used eagerly in multiple package-level vars
const maxItems = 3

var doubled = maxItems * 2

var tripled = maxItems * 3
