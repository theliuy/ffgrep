package main_test

import (
	"testing"

	ffgrep "github.com/theliuy/ffgrep"
)

func TestNewStringContainer(t *testing.T) {
	_, err := ffgrep.NewStringContainsMatcher("")
	if err == nil {
		t.Fatal("new string contains matcher with empty pattern")
	}
}

func TestStringContainsMatcher(t *testing.T) {
	ma, err := ffgrep.NewStringContainsMatcher("hello")
	if err != nil {
		t.Fatal(err.Error())
	}

	if !ma.Match([]byte("hello world")) {
		t.Fatal("should match")
	}

	if ma.Match([]byte("hola world")) {
		t.Fatal("should not match")
	}
}

func TestNewRegexMatcher(t *testing.T) {
	_, err := ffgrep.NewRegexpMatcher("+x+*")
	if err == nil {
		t.Fatal("bad regex pattern")
	}
}

func TestRegexpMatcher(t *testing.T) {
	ma, err := ffgrep.NewRegexpMatcher("hello")
	if err != nil {
		t.Fatal(err.Error())
	}
	if !ma.Match([]byte("hello world")) {
		t.Fatal("should match")
	}
	if ma.Match([]byte("hola world")) {
		t.Fatal("should not match")
	}

	ma, err = ffgrep.NewRegexpMatcher("a*b+.")
	if err != nil {
		t.Fatal(err.Error())
	}
	if !ma.Match([]byte("bbbx")) {
		t.Fatal("should match")
	}
	if !ma.Match([]byte("caabbbx")) {
		t.Fatal("should match")
	}
	if ma.Match([]byte("hola world")) {
		t.Fatal("should not match")
	}

}
