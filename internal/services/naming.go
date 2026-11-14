package services

import (
	"strconv"
	"strings"
)

// trimVersionSuffix turns a versioned resource name into its base name, e.g.
// "foo-3" -> "foo", "my-model-12" -> "my-model".
//
// Earlier versions used strings.Split(name, "-")[0], which returned "my" for "my-model-12".
// Only a trailing "-<number>" is treated as the version suffix.
func trimVersionSuffix(name string) string {
	i := strings.LastIndex(name, "-")
	if i <= 0 || i == len(name)-1 {
		return name
	}
	if _, err := strconv.ParseInt(name[i+1:], 10, 64); err != nil {
		return name
	}
	return name[:i]
}
