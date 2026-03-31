package testorder

import "testing"

func TestSomething(t *testing.T) {}

func TestMain(m *testing.M) { m.Run() } // want `Place TestMain first in test file.`
