package main

import (
	"fmt"
	"os"
)

func die(reason string) {
	if logger != nil {
		logger.Printf("dying: %v\n", reason)
	}
	fmt.Fprintln(os.Stderr, reason)
	os.Exit(-1)
}

func dieUsage(reason string) {
	if logger != nil {
		logger.Printf("dying: %v\n", reason)
	}
	fmt.Fprintf(os.Stderr, "%v\n%v\n", reason, usage)
	os.Exit(-1)
}

func fatal(any interface{}) {
	if logger == nil {
		fmt.Fprintf(os.Stderr, "%v\n", any)
		os.Exit(-1)
	}
	logger.Panicf("%v\n", any)
}

func fatalIf(any interface{}) {
	if logger == nil {
		fmt.Fprintf(os.Stderr, "%v\n", any)
		os.Exit(-1)
	}
	if any != nil {
		logger.Panicf("%v\n", any)
	}
}

// Helper writeCloser implementations

type nullWriteCloser bool

func (w nullWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (w nullWriteCloser) Close() (err error) {
	return nil
}

type stdoutWriteCloser bool

func (w stdoutWriteCloser) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (w stdoutWriteCloser) Close() (err error) {
	return nil
}

type stderrWriteCloser bool

func (w stderrWriteCloser) Write(p []byte) (n int, err error) {
	return os.Stderr.Write(p)
}

func (w stderrWriteCloser) Close() (err error) {
	return nil
}
