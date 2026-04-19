package plugin

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
)

func TestPluginsRegistered(t *testing.T) {
	for _, name := range []string{
		"boundarycontrol",
		"nodeepimports",
		"nofalsesharing",
		"nospecialunicode",
		"nounicodeescape",
		"readfriendlyorder",
	} {
		if _, err := register.GetPlugin(name); err != nil {
			t.Fatalf("plugin %q not registered: %v", name, err)
		}
	}
}
