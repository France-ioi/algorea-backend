package utils

import (
	"testing"
)

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "should capitalize the first letter",
			str:  "test string",
			want: "Test string",
		},
		{
			name: "should keep as it is if already capitalized",
			str:  "Test string",
			want: "Test string",
		},
		{
			name: "should handle empty strings",
			str:  "",
			want: "",
		},
		{
			name: "should keep non-letter character as they are",
			str:  ".test",
			want: ".test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Capitalize(tt.str); got != tt.want {
				t.Errorf("Capitalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		want  []string
	}{
		{
			name:  "should return the same slice if all elements are unique",
			slice: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "should return unique elements only",
			slice: []string{"a", "b", "a", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "should handle empty slices",
			slice: []string{},
			want:  []string{},
		},
	}

	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !Contains(b, a[i]) {
				return false
			}
		}
		return true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueStrings(tt.slice)

			if !equal(got, tt.want) {
				t.Errorf("UniqueStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}
