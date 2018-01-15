/* Synchronized outputs

 */

package main

import (
	"bufio"
	"os"
)

type IOutput interface {
	Writeln(b []byte)
	Close()
	Done() <-chan struct{}
}

type StdOutput struct {
	msgQ   chan []byte
	done   chan struct{}
	writer *bufio.Writer
}

func NewStdOutput(bufSize int) *StdOutput {
	output := &StdOutput{
		msgQ:   make(chan []byte, bufSize),
		done:   make(chan struct{}),
		writer: bufio.NewWriter(os.Stdout),
	}
	go output.Start()
	return output
}

func (o *StdOutput) Start() {
	defer close(o.done)
	for msg := range o.msgQ {
		if msg == nil {
			continue
		}
		o.writer.WriteString(string(msg) + "\n")
		o.writer.Flush()
	}
}

func (o *StdOutput) Writeln(msg []byte) {
	o.msgQ <- msg
}

func (o *StdOutput) Close() {
	close(o.msgQ)

}

func (o *StdOutput) Done() <-chan struct{} {
	return o.done
}
