package main

import (
	"sync"
)

type Searcher struct {
	matcher     IMatcher
	out         IOutput
	stream      *Stream
	streamIndex int
}

func NewSearcher(matcher IMatcher, stream *Stream, streamIndex int, out IOutput) *Searcher {
	return &Searcher{
		matcher:     matcher,
		stream:      stream,
		streamIndex: streamIndex,
		out:         out,
	}
}

func (s *Searcher) Run(wg *sync.WaitGroup) {
	go func() {
		// fmt.Printf("searcher starts\n")
		// defer fmt.Printf("searcher ends\n")
		defer wg.Done()

		for msg := range s.stream.Next(s.streamIndex) {
			if !s.matcher.Match(msg) {
				continue
			}

			s.out.Writeln(msg)
		}
	}()
}
