package main

import (
	"io"
	"os"
	"os/exec"
	"time"
)

func executor(workdir string, inputFifoPath string, maxChunkSize int, args []string) error {
	const stdoutMarker = 0x00
	// const stderrMarker = 0x01

	proc := exec.Command(args[0], args[1:]...)
	proc.Dir = workdir

	logger.Printf("Command path: %v\n", proc.Path)

	inputFifo := openFifo(inputFifoPath)

	signal := make(chan bool)
	go startPipeline(proc, os.Stdin, os.Stdout, inputFifo, maxChunkSize, signal)

	// wait pipeline to start
	<-signal

	err := proc.Start()
	fatal_if(err)

	// wait for pipeline exit
	<-signal

	err = safeExit(proc)
	if e, ok := err.(*exec.Error); ok {
		// This shouldn't really happen in practice because we check for
		// program existence in Elixir, before launching odu
		logger.Printf("Run ERROR: %v\n", e)
		os.Exit(3)
	}
	// TODO: return Stderr and exit stauts to beam process
	logger.Printf("Run FINISHED: %#v\n", err)
	return err
}

func startPipeline(proc *exec.Cmd, stdin io.Reader, outstream io.Writer, inputFifo *os.File, maxChunkSize int, signal chan bool) {
	// some commands expect stdin to be connected
	cmdInput, err := proc.StdinPipe()
	fatal_if(err)

	cmdOutput, err := proc.StdoutPipe()
	fatal_if(err)

	logger.Println("Starting pipeline")

	demand, demandConsumerExit := startCommandConsumer(stdin)
	startInputConsumer(cmdInput, inputFifo)
	outputStreamerExit := startOutputStreamer(cmdOutput, outstream, maxChunkSize, demand)

	// signal that pipline is setup
	signal <- true

	// wait for pipline to exit
	select {
	case <-demandConsumerExit:
	case <-outputStreamerExit:
	}

	cmdOutput.Close()
	cmdInput.Close()

	// signal pipeline shutdown
	signal <- true
}

func safeExit(proc *exec.Cmd) error {
	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()
	select {
	case <-time.After(3 * time.Second):
		if err := proc.Process.Kill(); err != nil {
			logger.Fatal("failed to kill process: ", err)
		}
		logger.Println("process killed as timeout reached")
		return nil
	case err := <-done:
		return err
	}
}

func openFifo(fifoPath string) *os.File {
	fifo, err := os.OpenFile(fifoPath, os.O_RDONLY, 0600)
	if err != nil {
		fatal(err)
	}
	return fifo
}
