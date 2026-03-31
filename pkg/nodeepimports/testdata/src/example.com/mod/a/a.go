package a

import (
	_ "example.com/mod/a/child"      // OK: 1 level deep
	_ "example.com/mod/a/child/deep" // want `example.com/mod/a/child/deep is too deep`
	_ "example.com/mod/b/other/deep" // OK: different scope
)
