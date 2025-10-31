package utils //nolint:revive

import (
	"fmt"
	"regexp"

	"github.com/google/go-cmp/cmp"
)

// Applies a function to each element of a slice, returning a new slice with the results
func MapSlice[T any, U any](slice []T, f func(T) U) []U {
	mapped := make([]U, len(slice))
	for i, v := range slice {
		mapped[i] = f(v)
	}
	return mapped
}

// Removes duplicate elements from a slice
func UniqueSliceElements[T comparable](inputSlice []T) []T {
	uniqueSlice := make([]T, 0, len(inputSlice))
	seen := make(map[T]bool, len(inputSlice))
	for _, element := range inputSlice {
		if !seen[element] {
			uniqueSlice = append(uniqueSlice, element)
			seen[element] = true
		}
	}
	return uniqueSlice
}

func CheckSliceContainment[T comparable](needles, haystack []T) []T {
	missingElements := []T{}

	// Check if each element in needles is contained in haystack
	for _, needle := range needles {
		found := false
		for _, hay := range haystack {
			if cmp.Equal(needle, hay) {
				found = true
				break
			}
		}
		if !found {
			missingElements = append(missingElements, needle)
		}
	}

	return missingElements
}

// ValidateNamespaceName checks if the namespace name is valid according to Kubernetes conventions
// and returns an error if it's not valid.
func ValidateNamespaceName(namespace string) error {
	// Kubernetes namespace name validation regex
	validNamespaceName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	// Check if the namespace name is between 1 and 63 characters long
	if len(namespace) < 1 || len(namespace) > 63 {
		return fmt.Errorf("invalid namespace name: '%s' (must be between 1 and 63 characters long)", namespace)
	}

	// Check if the namespace name matches the regex
	if !validNamespaceName.MatchString(namespace) {
		return fmt.Errorf("invalid namespace name: '%s' (must start and end with an alphanumeric character, and contain only lowercase letters, numbers, and dashes)", namespace)
	}

	// If all checks pass, return nil (no error)
	return nil
}
