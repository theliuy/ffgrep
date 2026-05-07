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

type StreamQueue chan []byte

type Stream struct {
	qList     []StreamQueue
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

	qList := make([]StreamQueue, numWorker)
	for i := 0; i < numWorker; i++ {
		qList[i] = make(StreamQueue, bufferSize)
	}

	stream := &Stream{
		qList:     qList,
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
		if offset >= s.fileSize {
			close(s.qList[readerId])
			continue
		}

		fh, err := os.Open(s.filename)
		if err != nil {
			return err
		}

		var startPos int64 = offset
		if offset > 0 {
			_, err = fh.Seek(offset-1, 0)
			if err != nil {
				return err
			}
			adjustOffset := adjustStart(fh)
			startPos = offset + adjustOffset
		}

		var endOffset int64
		if readerId == s.numWorker-1 {
			endOffset = s.fileSize
		} else {
			endOffset = offset + step
		}

		// 重新 seek 到正确的起始位置
		_, err = fh.Seek(startPos, 0)
		if err != nil {
			return err
		}

		wg.Add(1)
		go s.startReader(ctx, wg, readerId, fh, startPos, endOffset)

		offset += step
	}
	wg.Wait()

	return nil
}

func (s *Stream) FileSize() int64 {
	return s.fileSize
}

func (s *Stream) ReadSize() int64 {
	return atomic.LoadInt64(&s.readSize)
}

func (s *Stream) QNum() int {
	return s.numWorker
}

func (s *Stream) QLen(id int) int {
	return len(s.qList[id])
}

func (s *Stream) Next(id int) <-chan []byte {
	return s.qList[id]
}

func (s *Stream) startReader(ctx context.Context, wg *sync.WaitGroup, id int, fh *os.File, startOffset, endOffset int64) {
	// fmt.Printf("[reader %d] starts\n", id)
	// defer fmt.Printf("[reader %d] ends\n", id)

	defer wg.Done()
	defer close(s.qList[id])
	defer fh.Close()

	reader := bufio.NewReader(fh)
	var currentOffset int64 = startOffset

	for {
		select {
		case <-ctx.Done():
			return
		default:
			bs, err := reader.ReadBytes(LINE_ENDING)
			bytesRead := int64(len(bs))

			// including io.EOF
			if err != nil && err != io.EOF {
				return
			}

			line := bs
			hasNewline := len(line) > 0 && line[len(line)-1] == LINE_ENDING

			// remove line ending
			// fix it, when goes to windows
			if hasNewline {
				line = line[:len(line)-1]
			}

			// 即使是空行，也要更新偏移量
			currentOffset += bytesRead

			if len(line) == 0 {
				if err == io.EOF {
					return
				}
				continue
			}

			atomic.AddInt64(&s.readSize, int64(len(line)))
			s.qList[id] <- line

			// 检查是否超过 endOffset，注意是比较 currentOffset
			// 只有非最后一个reader才检查
			if endOffset < s.fileSize && currentOffset > endOffset {
				return
			}

			if err == io.EOF {
				return
			}
		}
	}
}

// adjust filehandler's start to the right position, and returns offset from original position
func adjustStart(fh *os.File) int64 {
	buf := make([]byte, 1)
	_, err := fh.Read(buf)

	if err == io.EOF {
		return 0
	}

	// starts from new line - next character is start of new line
	if string(buf) == "\n" {
		return 1
	}

	// Read until next line
	var offset int64 = 1 // already read 1 byte
	for {
		_, err = fh.Read(buf)
		if err == io.EOF {
			break
		}
		offset += 1

		if string(buf) == "\n" {
			break
		}
	}
	return offset
}
