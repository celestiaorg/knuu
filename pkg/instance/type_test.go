package instance

import (
	"testing"
)

func TestInstanceType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   InstanceType
		want string
	}{
		{
			name: "BasicInstance", // Test case name
			in:   BasicInstance,   // Input
			want: "BasicInstance", // Expected output
		},
		{
			name: "TimeoutHandlerInstance",
			in:   TimeoutHandlerInstance,
			want: "TimeoutHandlerInstance",
		},
		{
			name: "UnknownInstance",
			in:   UnknownInstance,
			want: "Unknown",
		},
		{
			name: "4",
			in:   4,
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.String()
			if got != tt.want {
				t.Errorf("got %q; want %q", got, tt.want)
			}
		})
	}
}
