package k8s

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		panic    bool
	}{
		{
			name:     "simple case",
			input:    "SimpleName",
			expected: "simplename",
			panic:    false,
		},
		{
			name:     "contains invalid characters",
			input:    "Name_With_Invalid_Characters!",
			expected: "name-with-invalid-characters",
			panic:    false,
		},
		{
			name:     "too long name",
			input:    strings.Repeat("a", 64),
			expected: strings.Repeat("a", 63),
			panic:    false,
		},
		{
			name:     "leading and trailing hyphens",
			input:    "---name---",
			expected: "name",
			panic:    false,
		},
		{
			name:     "name with mixed case",
			input:    "MixedCASEname",
			expected: "mixedcasename",
			panic:    false,
		},
		{
			name:     "name with spaces",
			input:    "name with spaces",
			expected: "name-with-spaces",
			panic:    false,
		},
		{
			name:     "name with dots and underscores",
			input:    "name.with.dots_and_underscores",
			expected: "name-with-dots-and-underscores",
			panic:    false,
		},
		{
			name:     "name with special characters",
			input:    "name!@#with$%^special&*()characters",
			expected: "name-with-special-characters",
			panic:    false,
		},
		{
			name:     "name with trailing hyphens after length cut",
			input:    strings.Repeat("a", 62) + "-b",
			expected: strings.Repeat("a", 62),
			panic:    false,
		},
		{
			name:  "empty name",
			input: "",
			panic: true,
		},
		{
			name:  "name with only invalid characters",
			input: "!!@@##$$",
			panic: true,
		},
		{
			name:     "name with leading and trailing spaces",
			input:    "  leading-and-trailing-spaces  ",
			expected: "leading-and-trailing-spaces",
			panic:    false,
		},
		{
			name:     "name with a mix of allowed and invalid characters",
			input:    "Name123_with.Mixed_Characters!",
			expected: "name123-with-mixed-characters",
			panic:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panic {
				assert.Panics(t, func() { SanitizeName(tt.input) })
				return
			}
			assert.Equal(t, tt.expected, SanitizeName(tt.input))
		})
	}
}
