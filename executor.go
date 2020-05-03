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

	signal := make(chan struct{})

	stdinFifo, stdoutFifo, stderrFifo := openIOFiles(inputFifoPath, outputFifoPath, errorFifoPath, signal)
	outputStreamerExit, commandExit := startPipeline(proc, stdinFifo, stdoutFifo, stderrFifo)

	err := proc.Start()
	fatalIf(err)

	// wait for pipline to exit
	select {
	case <-outputStreamerExit:
	case <-commandExit:
	}

	// signal that command is completed
	close(signal)

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

func openIOFiles(inputFifoPath string, outputFifoPath string, errorFifoPath string, signal chan struct{})(io.ReadCloser, io.WriteCloser,io.WriteCloser){
	ioReady := make(chan struct{})
	var stdinFifo io.ReadCloser
	var stdoutFifo io.WriteCloser
	var stderrFifo io.WriteCloser

	go func() {
		defer close(ioReady)
		stdinFifo = openReadCloser(inputFifoPath, signal)
		stdoutFifo = openWriteCloser(outputFifoPath, signal)
		stderrFifo = openWriteCloser(errorFifoPath, signal)
	}()

	// do not wait for FIFO files to open indefinitely
	select {
	case <-time.After(5 * time.Second):
		close(signal)
		fatal("FIFO files open timeout. Make sure fifo files are opened at other end")
	case <-ioReady:
	}
	return stdinFifo, stdoutFifo, stderrFifo
}

func startPipeline(proc *exec.Cmd, stdinFifo io.ReadCloser, stdoutFifo io.WriteCloser, stderrFifo io.WriteCloser) (<-chan struct{}, <-chan struct{}) {
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

	return outputStreamerExit, commandExit
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

func openReadCloser(fifoPath string, signal <-chan struct{}) io.ReadCloser {
	switch fifoPath {
	case "":
		writer := NullReadWriteCloser{
			Signal: make(chan struct{}),
		}
		linkChannel(writer, signal)
		return writer
	default:
		return openFile(fifoPath, os.O_RDONLY)
	}
}

func openWriteCloser(fifoPath string, signal <-chan struct{}) io.WriteCloser {
	switch fifoPath {
	case "":
		reader := NullReadWriteCloser{
			Signal: make(chan struct{}),
		}
		linkChannel(reader, signal)
		return reader
	default:
		return openFile(fifoPath, os.O_WRONLY)
	}
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

func linkChannel(closer io.Closer, b <-chan struct{}) {
	go func(){
		<-b
		closer.Close()
	}()
}
