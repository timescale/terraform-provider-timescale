package provider

import "strings"

// isNotSupportedRegion checks if the given region is supported.
// Returns false for Azure regions (prefixed with "az-").
func isNotSupportedRegion(region string) bool {
	return strings.HasPrefix(region, "az-")
}

// ErrUnsupportedRegion is the error message format for unsupported regions.
const ErrUnsupportedRegion = "region %s not supported"
