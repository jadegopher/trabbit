// Package version resolves build metadata for the CLI and UI.
package version

import (
	"runtime/debug"
	"strings"
)

const developmentVersion = "dev"

// GoReleaser injects the Git tag's version through -ldflags -X.
var buildVersion = developmentVersion

// Current is resolved once at startup; callers must not change it.
var Current = resolve(buildVersion, moduleVersion())

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

// Go supplies module or VCS versions without linker flags. Builds without
// version metadata report (devel), which we display as dev.
func resolve(injected, module string) string {
	value := injected
	if value == "" || value == developmentVersion {
		value = module
	}
	if value == "" || value == "(devel)" || value == developmentVersion {
		return developmentVersion
	}
	return "v" + strings.TrimPrefix(value, "v")
}
