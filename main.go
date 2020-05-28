package main

import (
	"flag"
	"fmt"
	"os"
)

// VERSION of the odu
const VERSION = "0.1.0"

const usage = "Usage: odu [options] -- <program> [<arg>...]"

var cdFlag = flag.String("cd", ".", "working directory for the spawned process")
var logFlag = flag.String("log", "", "enable logging")
var versionFlag = flag.Bool("v", false, "print version and exit")

func main() {
	flag.Parse()

	initLogger(*logFlag)

	if *versionFlag {
		fmt.Printf("%s\n", VERSION)
		os.Exit(0)
	}

	args := flag.Args()
	validateArgs(args)

	err := execute(*cdFlag, args)
	if err != nil {
		os.Exit(getExitStatus(err))
	}
}

func validateArgs(args []string) {
	if len(args) < 1 {
		dieUsage("Not enough arguments.")
	}

	logger.Printf("Flag values:\n  dir: %v\nArgs: %v\n", *cdFlag, args)
}

func notFifo(path string) bool {
	info, err := os.Stat(path)
	return os.IsNotExist(err) || info.Mode()&os.ModeNamedPipe == 0
}
