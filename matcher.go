package main

import (
	"bytes"
	"errors"
	"regexp"
)

type IMatcher interface {
	Match(s []byte) bool
}

type StringContainsMatcher struct {
	Pattern []byte
}

type RegexpMatcher struct {
	Re *regexp.Regexp
}

/* StringContainsMatcher

See if str contains a sub-string
*/
func NewStringContainsMatcher(pattern string) (*StringContainsMatcher, error) {
	if len(pattern) == 0 {
		return nil, errors.New("empty pattern")
	}

	return &StringContainsMatcher{
		Pattern: []byte(pattern),
	}, nil
}

func (ma *StringContainsMatcher) Match(s []byte) bool {
	return bytes.Contains(s, ma.Pattern)
}

/* RegexpMatcher

See if str match a pattern
*/
func NewRegexpMatcher(expr string) (*RegexpMatcher, error) {
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	return &RegexpMatcher{
		Re: re,
	}, nil
}

func (ma *RegexpMatcher) Match(s []byte) bool {
	return ma.Re.Match(s)
}
