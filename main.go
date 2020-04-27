package main

import (
	"flag"
	"fmt"
	"os"
)

// VERSION of the odu
const VERSION = "0.1.0"

const usage = "Usage: odu [options] -- <program> [<arg>...]"

var dirFlag = flag.String("dir", ".", "working directory for the spawned process")
var logFlag = flag.String("log", "", "enable logging")
var stdinFlag = flag.String("stdin", "", "path to fifo file from which input is read. ignored if not set")
var stdoutFlag = flag.String("stdout", "", "path to fifo file to which output written. ignored if not set")
var stderrFlag = flag.String("stderr", "", "path to fifo file to which stderr output is written. stderr is ignored if its not set (default)")
var versionFlag = flag.Bool("v", false, "print version and exit")

func main() {
	flag.Parse()

	initLogger(*logFlag)

	if *versionFlag {
		fmt.Printf("%s\n", VERSION)
		os.Exit(0)
	}

	if *stdoutFlag != "" && notFifo(*stdoutFlag) {
		dieUsage("stdin param is not a fifo file")
	}

	if *stdinFlag != "" && notFifo(*stdinFlag) {
		dieUsage("stdout param is not a fifo file")
	}

	if *stderrFlag != "" && notFifo(*stderrFlag) {
		dieUsage("stderr param invalid")
	}

	args := flag.Args()
	validateArgs(args)

	err := executor(*dirFlag, *stdinFlag, *stdoutFlag, *stderrFlag, args)
	if err != nil {
		os.Exit(getExitStatus(err))
	}
}

func validateArgs(args []string) {
	if len(args) < 1 {
		dieUsage("Not enough arguments.")
	}

	logger.Printf("Flag values:\n  dir: %v\nArgs: %v\n", *dirFlag, args)
}

func notFifo(path string) bool {
	info, err := os.Stat(path)
	return os.IsNotExist(err) || info.Mode()&os.ModeNamedPipe == 0
}
