package httputils

import "cmp"

// IsIn100s returns true if the given status code is in the 100-199 range (Informational).
func IsIn100s(statusCode int) bool {
	return isInRange(100, 199, statusCode)
}

// IsIn200s returns true if the given status code is in the 200-299 range (Success).
func IsIn200s(statusCode int) bool {
	return isInRange(200, 299, statusCode)
}

// IsIn300s returns true if the given status code is in the 300-399 range (Redirection).
func IsIn300s(statusCode int) bool {
	return isInRange(300, 399, statusCode)
}

// IsIn400s returns true if the given status code is in the 400-499 range (Client Error).
func IsIn400s(statusCode int) bool {
	return isInRange(400, 499, statusCode)
}

// IsIn500s returns true if the given status code is in the 500-599 range (Server Error).
func IsIn500s(statusCode int) bool {
	return isInRange(500, 599, statusCode)
}

// isInRange checks if the target value is within the specified range (inclusive).
// It uses generic constraints to work with any ordered type.
func isInRange[T cmp.Ordered](min, max, target T) bool {
	return target >= min && target <= max
}
