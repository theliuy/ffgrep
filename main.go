package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Options struct {
	numReader  int
	numMatcher int
	numCpu     int
	pattern    string
	isRegexp   bool
	file       string
}

const (
	INTRODUCTION = `
ffgrep - parallel file pattern searcher
`
	EXAMPLE = `
Example:
  # search text pattern
  ffgrep "hello" access.log

  # search regular expression pattern
  ffgrep -e 'hello[ab]+world' access.log
`

	BUFFER_MUTIFIER = 100
)

func parseOption() (*Options, error) {
	opt := new(Options)

	flag.IntVar(&opt.numReader, "r", 1, "Run up to r readers to read the file.")
	flag.IntVar(&opt.numMatcher, "m", runtime.NumCPU(), "Run m jobs to consume lines of the file.")
	flag.IntVar(&opt.numCpu, "c", runtime.NumCPU(), "Set the maxiumn number of CPU when executing.")
	flag.StringVar(&opt.pattern, "e", "", "Match each line by the given regular expression pattern.")

	flag.Parse()

	// text mode
	if opt.pattern == "" {
		opt.pattern = flag.Arg(0)
		opt.file = flag.Arg(1)
	} else {
		opt.isRegexp = true
		opt.file = flag.Arg(0)
	}

	errMesage := ""
	if opt.pattern == "" {
		errMesage = "Pattern must not be empty"
	} else if opt.file == "" {
		errMesage = "Filename must not be empty"
	} else if opt.numReader <= 0 {
		errMesage = "The number of readers (-r) must be positive"
	} else if opt.numCpu <= 0 {
		errMesage = "The number of CPU (-c) must be positive"
	}

	if errMesage != "" {
		fmt.Println(errMesage)
		fmt.Println(INTRODUCTION)
		flag.PrintDefaults()
		fmt.Println(EXAMPLE)
		return nil, errors.New("pasring option failed")
	}

	return opt, nil
}

func search(ctx context.Context, opt *Options, stream *Stream, out IOutput) error {
	defer out.Close()

	var err error
	wg := new(sync.WaitGroup)
	// init output
	for i := 0; i < opt.numMatcher; i++ {
		var matcher IMatcher
		if opt.isRegexp {
			matcher, err = NewRegexpMatcher(opt.pattern)
		} else {
			matcher, err = NewStringContainsMatcher(opt.pattern)
		}

		if err != nil {
			return err
		}

		searcher := NewSearcher(matcher, stream, out)

		wg.Add(1)
		searcher.Run(wg)
	}

	wg.Wait()
	return nil
}

func printStatus(stream *Stream, startTime int64) {
	total := stream.FileSize()
	read := stream.ReadSize()
	qLen := stream.QLen()

	timeLast := time.Now().UnixNano() - startTime
	var timeRemaining int64 = -1
	if total > 0 && read > 0 {
		timeRemaining = int64(float64(timeLast) * float64(total-read) / float64(read))
	}

	fmt.Printf(
		"progress=%.2f (%d / %d)  QLength=%d last=%v remain=%v\n",
		float64(read)/float64(total),
		read, total, qLen,
		formatDurationSecond(timeLast),
		formatDurationSecond(timeRemaining),
	)
}

func formatDurationSecond(ds int64) string {
	if ds < 0 {
		return "N/A"
	}

	d := time.Duration(ds)
	return d.String()
}

func main() {
	var err error
	opt, err := parseOption()
	if err != nil {
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	out := NewStdOutput(opt.numMatcher * BUFFER_MUTIFIER)

	stream, err := NewStream(ctx, opt.file, opt.numReader, opt.numReader*BUFFER_MUTIFIER)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(2)
	}

	startTime := time.Now().UnixNano()
	go search(ctx, opt, stream, out)
	//	if err != nil {
	//		os.Exit(2)
	//	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, infoSig)
	for {
		select {
		case sig := <-sigs:
			if sig == infoSig {
				printStatus(stream, startTime)
			} else {
				cancel()
			}
		case <-out.Done():
			return
		}
	}

}
