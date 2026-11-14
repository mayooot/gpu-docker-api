package services

import "testing"

func TestTrimVersionSuffix(t *testing.T) {
	cases := map[string]string{
		"foo-3":       "foo",
		"my-model-12": "my-model",
		"foo":         "foo",
		"foo-":        "foo-",
		"-1":          "-1",
		"foo-bar":     "foo-bar",
	}
	for in, want := range cases {
		if got := trimVersionSuffix(in); got != want {
			t.Errorf("trimVersionSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}
