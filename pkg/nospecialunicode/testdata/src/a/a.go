package a

var good = "plain ascii text"
var goodEmpty = ""
var goodSpecialAscii = "hello - world ... 'quoted' \"double\""

var badEmDash = "a — b"          // want `String contains em dash \(U\+2014\). Use the ASCII equivalent.`
var badEnDash = "a – b"          // want `String contains en dash \(U\+2013\). Use the ASCII equivalent.`
var badEllipsis = "wait…"        // want `String contains horizontal ellipsis \(U\+2026\). Use the ASCII equivalent.`
var badLeftDQ = "\u201Chello"    // want `String contains left double quotation mark \(U\+201C\). Use the ASCII equivalent.`
var badRightDQ = "hello\u201D"   // want `String contains right double quotation mark \(U\+201D\). Use the ASCII equivalent.`
var badLeftSQ = "\u2018hello"    // want `String contains left single quotation mark \(U\+2018\). Use the ASCII equivalent.`
var badRightSQ = "hello\u2019"   // want `String contains right single quotation mark \(U\+2019\). Use the ASCII equivalent.`
var badNBSP = "hello\u00A0world" // want `String contains non-breaking space \(U\+00A0\). Use the ASCII equivalent.`
var badZWS = "hello\u200Bworld"  // want `String contains zero-width space \(U\+200B\). Use the ASCII equivalent.`

var badRawString = `hello — world` // want `String contains em dash \(U\+2014\). Use the ASCII equivalent.`

var badMultiple = "a\u2014b\u2013c" // want `String contains en dash \(U\+2013\). Use the ASCII equivalent.` `String contains em dash \(U\+2014\). Use the ASCII equivalent.`

var badRune = '\u2014' // want `String contains em dash \(U\+2014\). Use the ASCII equivalent.`
