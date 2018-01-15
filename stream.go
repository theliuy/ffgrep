package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

type Stream struct {
	q         chan []byte
	fileSize  int64
	readSize  int64
	numWorker int
	filename  string
}

const (
	STREAM_DEFAULT_SEEK_STEP int = 64
	LINE_ENDING                  = '\n'
)

func NewStream(ctx context.Context, filename string, numWorker, bufferSize int) (*Stream, error) {
	if numWorker == 0 {
		return nil, errors.New("zero workers")
	}

	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	stats, err := fh.Stat()
	if err != nil {
		return nil, err
	}

	stream := &Stream{
		q:         make(chan []byte, bufferSize),
		fileSize:  stats.Size(),
		readSize:  0,
		numWorker: numWorker,
		filename:  filename,
	}

	go stream.kickoff(ctx)
	return stream, nil
}

func (s *Stream) kickoff(ctx context.Context) error {
	var offset int64 = 0
	var step int64 = s.fileSize / int64(s.numWorker)
	if step <= 0 {
		step = s.fileSize
	}

	wg := new(sync.WaitGroup)

	for readerId := 0; readerId < s.numWorker; readerId++ {
		if offset > s.fileSize {
			break
		}

		fh, err := os.Open(s.filename)
		if err != nil {
			return err
		}

		if offset > 0 {
			_, err = fh.Seek(offset-1, 0)
			if err != nil {
				return err
			}
			adjustOffset := adjustStart(fh)
			offset += adjustOffset
		}

		wg.Add(1)
		go s.startReader(ctx, wg, readerId, fh, step)

		offset += step
	}
	wg.Wait()
	// fmt.Printf("close q\n")
	close(s.q)

	return nil
}

func (s *Stream) FileSize() int64 {
	return s.fileSize
}

func (s *Stream) ReadSize() int64 {
	return atomic.LoadInt64(&s.readSize)
}

func (s *Stream) QLen() int {
	return len(s.q)
}

func (s *Stream) Next() <-chan []byte {
	return s.q
}

func (s *Stream) startReader(ctx context.Context, wg *sync.WaitGroup, id int, fh *os.File, numToEnd int64) {
	// fmt.Printf("[reader %d] starts\n", id)
	// defer fmt.Printf("[reader %d] ends\n", id)
	defer wg.Done()
	defer fh.Close()

	reader := bufio.NewReader(fh)
	var numRead int64 = 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			bs, err := reader.ReadBytes(LINE_ENDING)
			numRead += int64(len(bs))

			// including io.EOF
			if err != nil && err != io.EOF {
				return
			}

			// remove line ending
			// fix it, when goes to windows
			if len(bs) > 0 && bs[len(bs)-1] == LINE_ENDING {
				bs = bs[:len(bs)-1]
			}

			lenBs := int64(len(bs))
			if lenBs == 0 {
				if err == io.EOF {
					return
				}
				continue
			}
			atomic.AddInt64(&s.readSize, lenBs)

			s.q <- bs

			if numRead > numToEnd || err == io.EOF {
				return
			}
		}
	}
}

// adjust filehandler's start to the rigth position, and returns offset
func adjustStart(fh *os.File) int64 {
	buf := make([]byte, 1)
	_, err := fh.Read(buf)

	if err == io.EOF {
		return 0
	}

	// starts from new line
	if string(buf) == "\n" {
		return 0
	}

	// Read until next line
	var offset int64 = 0
	for {
		_, err = fh.Read(buf)
		offset += 1
		if err == io.EOF {
			break
		}

		if string(buf) == "\n" {
			break
		}
	}
	return offset
}
