package main_test

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	ffgrep "github.com/theliuy/ffgrep"
)

func TestStream(t *testing.T) {
	testFile := testFile()
	t.Log("test: %s\n", testFile)

	// read test file in a rough way
	lines, err := testReadAll(testFile)
	if err != nil {
		t.Fatalf("read file: %s", testFile)
	}

	ctx := context.Background()

	// 1 reader case
	stream1, err := ffgrep.NewStream(ctx, testFile, 1, 10)
	if err != nil {
		t.Fatalf("build stream: %s", testFile)
	}
	testValidateStream(t, stream1, lines)

	// multi reader
	stream2, err := ffgrep.NewStream(ctx, testFile, 5, 10)
	if err != nil {
		t.Fatalf("build stream: %s", testFile)
	}
	testValidateStream(t, stream2, lines)

	// more than number of lines
	stream3, err := ffgrep.NewStream(ctx, testFile, 400, 10)
	if err != nil {
		t.Fatalf("build stream: %s", testFile)
	}
	testValidateStream(t, stream3, lines)

}

// LICENCE is a perfect
func testFile() string {
	goPath := os.Getenv("GOPATH")
	return filepath.Join(goPath, "src", "github.com", "theliuy", "ffgrep", "LICENSE")
	// return filepath.Join(goPath, "src", "github.com", "theliuy", "ffgrep", "x.file")
}

func testReadAll(filename string) ([]string, error) {
	var lines []string
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	reader := bufio.NewReader(fh)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}

		line = strings.Replace(line, "\n", "", -1)
		if line == "" {
			if err == io.EOF {
				break
			}
			continue
		}

		lines = append(lines, line)
		if err == io.EOF {
			break
		}
	}

	return lines, nil
}

func testValidateStream(t *testing.T, stream *ffgrep.Stream, lines []string) {
	var streamLines []string

	// read all lines via stream
	for i := 0; i < stream.QNum(); i++ {
		for line := range stream.Next(i) {
			streamLines = append(streamLines, string(line))
		}
	}

	if len(lines) != len(streamLines) {
		// t.Fatalf("line number mismatch: exp=%d stream=%d", len(lines), len(streamLines))
		// return
	}

	sort.Strings(lines)
	sort.Strings(streamLines)

	minLen := len(lines)
	if len(streamLines) < minLen {
		minLen = len(streamLines)
	}
	for i, _ := range lines {
		if lines[i] == streamLines[i] {
			continue
		}

		t.Fatalf("line mismatch: exp=\"%s\" stream=\"%s\"", lines[i], streamLines[i])
	}
}
