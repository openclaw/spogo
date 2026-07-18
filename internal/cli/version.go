package cli

import (
	"strings"
)

var version = "dev"

func currentVersion() string {
	if linkedVersion := strings.TrimPrefix(strings.TrimSpace(version), "v"); linkedVersion != "" {
		return linkedVersion
	}
	return "dev"
}
