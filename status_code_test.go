package httputils

import (
	"fmt"
	"testing"
)

func TestIsIn100s(t *testing.T) {
	tests := []struct {
		Code     int
		Expected bool
	}{
		{Code: 99, Expected: false},
		{Code: 100, Expected: true},
		{Code: 150, Expected: true},
		{Code: 199, Expected: true},
		{Code: 200, Expected: false},
		{Code: -100, Expected: false},
		{Code: -150, Expected: false},
		{Code: 0, Expected: false},
		{Code: 300, Expected: false},
	}

	for indx, test := range tests {
		t.Run(fmt.Sprintf("Test %v", indx), func(t *testing.T) {
			result := IsIn100s(test.Code)
			if test.Expected != result {
				t.Errorf("Expected: %v || Received: %v", test.Expected, result)
			}
		})
	}
}

func TestIsIn200s(t *testing.T) {
	tests := []struct {
		Code     int
		Expected bool
	}{
		{Code: 100, Expected: false},
		{Code: 199, Expected: false},
		{Code: 200, Expected: true},
		{Code: 250, Expected: true},
		{Code: 299, Expected: true},
		{Code: 300, Expected: false},
		{Code: -200, Expected: false},
		{Code: -250, Expected: false},
		{Code: 0, Expected: false},
		{Code: 400, Expected: false},
	}

	for indx, test := range tests {
		t.Run(fmt.Sprintf("Test %v", indx), func(t *testing.T) {
			result := IsIn200s(test.Code)
			if test.Expected != result {
				t.Errorf("Expected: %v || Received: %v", test.Expected, result)
			}
		})
	}
}

func TestIsIn300s(t *testing.T) {
	tests := []struct {
		Code     int
		Expected bool
	}{
		{Code: 200, Expected: false},
		{Code: 299, Expected: false},
		{Code: 300, Expected: true},
		{Code: 350, Expected: true},
		{Code: 399, Expected: true},
		{Code: 400, Expected: false},
		{Code: -300, Expected: false},
		{Code: -350, Expected: false},
		{Code: 0, Expected: false},
		{Code: 500, Expected: false},
	}

	for indx, test := range tests {
		t.Run(fmt.Sprintf("Test %v", indx), func(t *testing.T) {
			result := IsIn300s(test.Code)
			if test.Expected != result {
				t.Errorf("Expected: %v || Received: %v", test.Expected, result)
			}
		})
	}
}

func TestIsIn400s(t *testing.T) {
	tests := []struct {
		Code     int
		Expected bool
	}{
		{Code: 100, Expected: false},
		{Code: 200, Expected: false},
		{Code: 300, Expected: false},
		{Code: 399, Expected: false},
		{Code: 400, Expected: true},
		{Code: 450, Expected: true},
		{Code: 499, Expected: true},
		{Code: -400, Expected: false},
		{Code: -450, Expected: false},
		{Code: -499, Expected: false},
		{Code: 500, Expected: false},
		{Code: 600, Expected: false},
	}

	for indx, test := range tests {
		t.Run(fmt.Sprintf("Test %v", indx), func(t *testing.T) {
			result := IsIn400s(test.Code)
			if test.Expected != result {
				t.Errorf("Expected: %v || Received: %v", test.Expected, result)
			}
		})
	}
}

func TestIsIn500s(t *testing.T) {
	tests := []struct {
		Code     int
		Expected bool
	}{
		{Code: 100, Expected: false},
		{Code: 200, Expected: false},
		{Code: 300, Expected: false},
		{Code: 399, Expected: false},
		{Code: 499, Expected: false},
		{Code: 500, Expected: true},
		{Code: 550, Expected: true},
		{Code: 599, Expected: true},
		{Code: -500, Expected: false},
		{Code: -550, Expected: false},
		{Code: -599, Expected: false},
		{Code: 600, Expected: false},
	}

	for indx, test := range tests {
		t.Run(fmt.Sprintf("Test %v", indx), func(t *testing.T) {
			result := IsIn500s(test.Code)
			if test.Expected != result {
				t.Errorf("Expected: %v || Received: %v", test.Expected, result)
			}
		})
	}
}

// Additional comprehensive test to verify edge cases
func TestStatusCodeRanges(t *testing.T) {
	// Test boundary conditions for all ranges
	testCases := []struct {
		function string
		code     int
		expected bool
	}{
		// 100s range boundaries
		{"IsIn100s", 99, false},
		{"IsIn100s", 100, true},
		{"IsIn100s", 199, true},
		{"IsIn100s", 200, false},

		// 200s range boundaries
		{"IsIn200s", 199, false},
		{"IsIn200s", 200, true},
		{"IsIn200s", 299, true},
		{"IsIn200s", 300, false},

		// 300s range boundaries
		{"IsIn300s", 299, false},
		{"IsIn300s", 300, true},
		{"IsIn300s", 399, true},
		{"IsIn300s", 400, false},

		// 400s range boundaries
		{"IsIn400s", 399, false},
		{"IsIn400s", 400, true},
		{"IsIn400s", 499, true},
		{"IsIn400s", 500, false},

		// 500s range boundaries
		{"IsIn500s", 499, false},
		{"IsIn500s", 500, true},
		{"IsIn500s", 599, true},
		{"IsIn500s", 600, false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%d", tc.function, tc.code), func(t *testing.T) {
			var result bool
			switch tc.function {
			case "IsIn100s":
				result = IsIn100s(tc.code)
			case "IsIn200s":
				result = IsIn200s(tc.code)
			case "IsIn300s":
				result = IsIn300s(tc.code)
			case "IsIn400s":
				result = IsIn400s(tc.code)
			case "IsIn500s":
				result = IsIn500s(tc.code)
			}

			if result != tc.expected {
				t.Errorf("%s(%d): expected %v, got %v", tc.function, tc.code, tc.expected, result)
			}
		})
	}
}
