package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

func executor(workdir string, inputFifoPath string, outputFifoPath string, errorFifoPath string, args []string) error {
	proc := exec.Command(args[0], args[1:]...)
	proc.Dir = workdir

	logger.Printf("Command path: %v\n", proc.Path)

	stdinFifo := openReadCloser(inputFifoPath)
	stdoutFifo := openWriteCloser(outputFifoPath)
	stderrFifo := openWriteCloser(errorFifoPath)

	signal := make(chan bool)
	go startPipeline(proc, stdinFifo, stdoutFifo, stderrFifo, signal)

	// wait pipeline to start
	<-signal

	err := proc.Start()
	fatalIf(err)

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

func startPipeline(proc *exec.Cmd, stdinFifo io.ReadCloser, stdoutFifo io.WriteCloser, stderrFifo io.WriteCloser, signal chan bool) {
	// some commands expect stdin to be connected
	cmdInput, err := proc.StdinPipe()
	fatalIf(err)

	cmdOutput, err := proc.StdoutPipe()
	fatalIf(err)

	cmdError, err := proc.StderrPipe()
	fatalIf(err)

	logger.Println("Starting pipeline")

	startInputConsumer(cmdInput, stdinFifo)
	outputStreamerExit := startOutputStreamer(cmdOutput, stdoutFifo)
	startErrorStreamer(cmdError, stderrFifo)
	commandExit := createCommandExitChan(os.Stdin)

	// signal that pipline is setup
	signal <- true

	// wait for pipline to exit
	select {
	case <-outputStreamerExit:
	case <-commandExit:
	}

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

func openReadCloser(fifoPath string) io.ReadCloser {
	return openFile(fifoPath, os.O_RDONLY)
}

func openWriteCloser(fifoPath string) io.WriteCloser {
	var file io.WriteCloser
	switch fifoPath {
	case "":
		file = new(nullWriteCloser)
	default:
		file = openFile(fifoPath, os.O_WRONLY)
	}
	return file
}

func openFile(path string, mode int) *os.File {
	file, err := os.OpenFile(path, mode, 0600)
	if err != nil {
		fatal(err)
	}
	return file
}

func createCommandExitChan(stdin io.ReadCloser) <-chan struct{} {
	exitSignal := make(chan struct{})
	go func() {
		defer close(exitSignal)

		_, err := io.Copy(ioutil.Discard, stdin)
		fatalIf(err)
	}()

	return exitSignal
}
