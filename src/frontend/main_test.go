package main

import "testing"

func TestIsAPIPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/api", true},
		{"/api/", true},
		{"/api/v1", true},
		{"/apiv2", false},
		{"/", false},
		{"/cars", false},
	}
	for _, c := range cases {
		if got := isAPIPath(c.path); got != c.want {
			t.Errorf("isAPIPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestStripAPIPrefix(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/api", "/"},
		{"/api/", "/"},
		{"/api/v1", "/v1"},
		{"/api/v1/query", "/v1/query"},
	}
	for _, c := range cases {
		if got := stripAPIPrefix(c.path); got != c.want {
			t.Errorf("stripAPIPrefix(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}
