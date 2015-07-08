package main

import (
	_ "fmt"
	"testing"
)

func TestNewByteArray(t *testing.T) {
	var tests = []struct {
		length int
	}{
		{0}, {1}, {10}, {100}, {1000},
	}

	for _, test := range tests {
		b := newByteArray(test.length)
		if test.length != len(b) {
			t.Errorf("\nExpected: %d\nActual: %d", test.length, len(b))
		}
	}
}
