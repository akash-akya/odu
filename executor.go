package main

import (
	"io"
	"os"
	"os/exec"
)

func executor(workdir string, maxChunkSize int, args []string) error {
	const stdoutMarker = 0x00
	// const stderrMarker = 0x01

	proc := exec.Command(args[0], args[1:]...)
	proc.Dir = workdir

	logger.Printf("Command path: %v\n", proc.Path)

	signal := make(chan bool)
	go startPipeline(proc, os.Stdin, os.Stdout, maxChunkSize, signal)

	// wait pipeline to start
	<-signal

	err := proc.Start()
	fatal_if(err)

	// wait for pipeline exit
	<-signal

	err = proc.Wait()
	if e, ok := err.(*exec.Error); ok {
		// This shouldn't really happen in practice because we check for
		// program existence in Elixir, before launching odu
		logger.Printf("Run ERROR: %v\n", e)
		os.Exit(3)
	}
	logger.Printf("Run FINISHED: %#v\n", err)
	return err
}

func startPipeline(proc *exec.Cmd, stdin io.Reader, outstream io.Writer, maxChunkSize int, signal chan bool) {
	done := make(chan struct{})
	defer close(done)

	// some commands expect stdin to be connected
	cmdInput, err := proc.StdinPipe()
	fatal_if(err)

	cmdOutput, err := proc.StdoutPipe()
	fatal_if(err)

	logger.Println("Starting pipeline")

	demand := startCommandConsumer(done, stdin)
	exit := startOutputStreamer(done, cmdOutput, outstream, maxChunkSize, demand)

	// signal that pipline is setup
	signal <- true

	// wait for pipline to exit
	<-exit

	cmdOutput.Close()
	cmdInput.Close()

	// signal pipeline shutdown
	signal <- true
}
