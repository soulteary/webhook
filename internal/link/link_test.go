package link_test

import (
	"testing"

	"github.com/soulteary/webhook/internal/link"
)

func TestMakeBaseURL(t *testing.T) {
	tests := []struct {
		name   string
		prefix *string
		want   string
	}{
		{
			name:   "nil prefix",
			prefix: nil,
			want:   "",
		},
		{
			name:   "empty prefix",
			prefix: new(string),
			want:   "",
		},
		{
			name:   "non-empty prefix",
			prefix: newString("api"),
			want:   "/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := link.MakeBaseURL(tt.prefix); got != tt.want {
				t.Errorf("MakeBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeRoutePattern(t *testing.T) {
	tests := []struct {
		name   string
		prefix *string
		want   string
	}{
		{
			name:   "nil prefix route pattern",
			prefix: nil,
			want:   "/{id:.*}",
		},
		{
			name:   "empty prefix route pattern",
			prefix: new(string),
			want:   "/{id:.*}",
		},
		{
			name:   "non-empty prefix route pattern",
			prefix: newString("api"),
			want:   "/api/{id:.*}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := link.MakeRoutePattern(tt.prefix); got != tt.want {
				t.Errorf("MakeRoutePattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeHumanPattern(t *testing.T) {
	tests := []struct {
		name   string
		prefix *string
		want   string
	}{
		{
			name:   "nil prefix human pattern",
			prefix: nil,
			want:   "/{id}",
		},
		{
			name:   "empty prefix human pattern",
			prefix: new(string),
			want:   "/{id}",
		},
		{
			name:   "non-empty prefix human pattern",
			prefix: newString("api"),
			want:   "/api/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := link.MakeHumanPattern(tt.prefix); got != tt.want {
				t.Errorf("MakeHumanPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

// newString is a helper function to create a pointer to a string.
func newString(s string) *string {
	return &s
}
