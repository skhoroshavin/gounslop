package a

var good = "—"
var goodAscii = "plain ascii"
var goodRaw = `\u2014`
var goodEmpty = ""

var bad = "\u2014"          // want `Use the actual character instead of a \\uXXXX escape sequence.`
var badUpper = "\U00002014" // want `Use the actual character instead of a \\uXXXX escape sequence.`
var badChar = '\u2014'      // want `Use the actual character instead of a \\uXXXX escape sequence.`
