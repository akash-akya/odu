package main

import (
	"flag"
	"fmt"
	"os"
)

const VERSION = "0.1.0"

const usage = "Usage: odu [options] -- <program> [<arg>...]"

var dirFlag = flag.String("dir", ".", "working directory for the spawned process")
var logFlag = flag.String("log", "", "enable logging")
var chunkSizeFlag = flag.Int("chunk-size", 65035, "maximum chunk size (depends on operating system)")
var versionFlag = flag.Bool("v", false, "print version and exit")

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s\n", VERSION)
		os.Exit(0)
	}

	if *chunkSizeFlag <= 0 {
		die_usage("chunk-size should be a valid positive integer.")
	}

	initLogger(*logFlag)

	args := flag.Args()
	validateArgs(args)

	err := executor(*dirFlag, *chunkSizeFlag, args)
	if err != nil {
		os.Exit(getExitStatus(err))
	}
}

func validateArgs(args []string) {
	if len(args) < 1 {
		die_usage("Not enough arguments.")
	}

	logger.Printf("Flag values:\n  dir: %v\nArgs: %v\n", *dirFlag, args)
}
