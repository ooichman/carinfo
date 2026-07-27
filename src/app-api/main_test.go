package main

import (
	"os"
	"testing"
)

func TestDbapiBase(t *testing.T) {
	cases := []struct {
		env  string
		want string
	}{
		{"http://dbapi:8080", "http://dbapi:8080"},
		{"http://dbapi:8080/", "http://dbapi:8080"},
		{"http://dbapi:8080/query", "http://dbapi:8080"},
		{"http://dbapi:8080/query/", "http://dbapi:8080"},
	}
	for _, c := range cases {
		os.Setenv("DBAPI_URL", c.env)
		if got := dbapiBase(); got != c.want {
			t.Errorf("DBAPI_URL=%q → %q, want %q", c.env, got, c.want)
		}
	}
	os.Unsetenv("DBAPI_URL")
}
