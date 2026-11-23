package post

import (
	"strings"
	"testing"
)

func TestSlugifyAndReadingTime(t *testing.T) {
	cases := map[string]string{
		"Artikel Uji 123!":             "artikel-uji-123",
		"  --  ":                       "artikel",
		"日本語":                          "artikel",
		strings.Repeat("a", 300) + "-": strings.Repeat("a", 200),
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Fatalf("Slugify(%q)=%q want %q", in, got, want)
		}
	}
	if ReadingTime("") != 1 || ReadingTime("a b c") != 1 || ReadingTime(strings.Repeat("k ", 200)) != 1 ||
		ReadingTime(strings.Repeat("k ", 201)) != 2 || ReadingTime(strings.Repeat("k ", 450)) != 3 {
		t.Fatal("reading time salah")
	}
	if s := randomSuffix(); len(s) != 6 {
		t.Fatalf("suffix %q", s)
	}
}
