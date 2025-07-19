package httputils

import "cmp"

// IsIn100s returns a bool indicating if the given status code is between 100-199
func IsIn100s(statusCode int) bool {
	return isInRange(100, 199, statusCode)
}

// IsIn200s returns a bool indicating if the given status code is between 200-299
func IsIn200s(statusCode int) bool {
	return isInRange(200, 299, statusCode)
}

// IsIn300s returns a bool indicating if the given status code is between 300-399
func IsIn300s(statusCode int) bool {
	return isInRange(300, 399, statusCode)
}

// IsIn400s returns a bool indicating if the given status code is between 400-499
func IsIn400s(statusCode int) bool {
	return isInRange(400, 499, statusCode)
}

// IsIn500s returns a bool indicating if the given status code is between 500-599
func IsIn500s(statusCode int) bool {
	return isInRange(500, 599, statusCode)
}

// isInRange checks if the target is in a given range, inclusive
func isInRange[T cmp.Ordered](min, max, target T) bool {
	return target >= min && target <= max
}
