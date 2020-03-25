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
