package main

import (
	"slices"
	"testing"
)

func TestTagList(t *testing.T) {
	cases := []struct {
		in   any
		want []string
	}{
		{"estudos", []string{"estudos"}},
		{"a, #b", []string{"a", "b"}},
		{[]any{"#hub", "estudos", 3, " "}, []string{"hub", "estudos"}},
		{nil, nil},
	}
	for _, c := range cases {
		if got := tagList(c.in); !slices.Equal(got, c.want) {
			t.Errorf("tagList(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
