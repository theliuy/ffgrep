package main

import (
	"sync"
)

type Searcher struct {
	matcher IMatcher
	out     IOutput
	stream  *Stream
}

func NewSearcher(matcher IMatcher, stream *Stream, out IOutput) *Searcher {
	return &Searcher{
		matcher: matcher,
		stream:  stream,
		out:     out,
	}
}

func (s *Searcher) Run(wg *sync.WaitGroup) {
	go func() {
		// fmt.Printf("searcher starts\n")
		// defer fmt.Printf("searcher ends\n")
		defer wg.Done()

		for msg := range s.stream.Next() {
			if !s.matcher.Match(msg) {
				continue
			}

			s.out.Writeln(msg)
		}
	}()
}
