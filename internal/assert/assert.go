package assert

import (
	"net/http"
	"slices"
	"testing"
)

func Equal[T comparable](t *testing.T, expected, got T) {
	t.Helper()

	if expected != got {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func EqualHeaders(t *testing.T, expected, got http.Header) {
	for key := range expected {
		if !slices.Equal(expected.Values(key), got.Values(key)) {
			t.Errorf("expected: %v, got %v for %s key in header", expected.Values(key), got.Values(key), key)
		}
	}
}
