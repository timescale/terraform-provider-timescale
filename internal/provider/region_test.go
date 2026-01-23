package provider

import (
	"fmt"
	"testing"
)

func TestIsNotSupportedRegion(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected bool
	}{
		// AWS regions - should be supported (isNotSupportedRegion returns false)
		{"AWS us-east-1", "us-east-1", false},
		{"AWS us-west-2", "us-west-2", false},
		{"AWS eu-central-1", "eu-central-1", false},
		{"AWS ap-southeast-1", "ap-southeast-1", false},

		// Azure regions - should NOT be supported (isNotSupportedRegion returns true)
		{"Azure az-eastus", "az-eastus", true},
		{"Azure az-westeurope", "az-westeurope", true},
		{"Azure az-australiaeast", "az-australiaeast", true},
		{"Azure az-southeastasia", "az-southeastasia", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotSupportedRegion(tt.region)
			if result != tt.expected {
				t.Errorf("isNotSupportedRegion(%q) = %v, want %v", tt.region, result, tt.expected)
			}
		})
	}
}

func TestErrUnsupportedRegionFormat(t *testing.T) {
	region := "az-eastus"
	expected := "region az-eastus not supported"
	result := fmt.Sprintf(ErrUnsupportedRegion, region)
	if result != expected {
		t.Errorf("ErrUnsupportedRegion format = %q, want %q", result, expected)
	}
}
