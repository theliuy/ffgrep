package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
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
	pprofFile  string
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
	flag.IntVar(&opt.numMatcher, "m", 4, "Run m jobs per reader to consume lines of the file.")
	flag.IntVar(&opt.numCpu, "c", runtime.NumCPU(), "Set the maxiumn number of CPU when executing.")
	flag.StringVar(&opt.pattern, "e", "", "Match each line by the given regular expression pattern.")
	flag.StringVar(&opt.pprofFile, "p", "", "Enable pprof by setting output file.")

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
	for w := 0; w < stream.QNum(); w++ {
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

			searcher := NewSearcher(matcher, stream, w, out)

			wg.Add(1)
			searcher.Run(wg)
		}
	}

	wg.Wait()
	return nil
}

func printStatus(stream *Stream, startTime int64) {
	total := stream.FileSize()
	read := stream.ReadSize()

	var qLenState []string
	for i := 0; i < stream.QNum(); i++ {
		qLenState = append(qLenState, fmt.Sprintf("%d", stream.QLen(i)))
	}

	timeLast := time.Now().UnixNano() - startTime
	var timeRemaining int64 = -1
	if total > 0 && read > 0 {
		timeRemaining = int64(float64(timeLast) * float64(total-read) / float64(read))
	}

	fmt.Printf(
		"progress=%.2f (%d / %d)  Q=%s last=%v remain=%v\n",
		float64(read)/float64(total),
		read, total,
		strings.Join(qLenState, ","),
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
	// defer cancel()

	out := NewStdOutput(opt.numMatcher * BUFFER_MUTIFIER)

	if opt.pprofFile != "" {
		pprofFile, err := os.Create(opt.pprofFile)
		if err != nil {
			os.Exit(1)
		}

		pprof.StartCPUProfile(pprofFile)
		defer pprof.StopCPUProfile()
	}

	stream, err := NewStream(ctx, opt.file, opt.numReader, opt.numReader*BUFFER_MUTIFIER)
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(2)
	}

	startTime := time.Now().UnixNano()
	go search(ctx, opt, stream, out)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGUSR1 {
				printStatus(stream, startTime)
			} else {
				cancel()
				return
			}
		case <-out.Done():
			return
		}
	}

}
