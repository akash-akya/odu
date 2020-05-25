package main

import (
	"encoding/binary"
	"os"
	"os/exec"
	"time"
)

func executor(workdir string, args []string) error {
	closeStdin := make(chan struct{})
	done := make(chan struct{})
	input := make(chan []byte, 1)

	outputDemand := make(chan bool, 1) // buffered
	inputDemand := make(chan bool)

	defer func() {
		close(outputDemand)
		close(input)
	}()

	proc := exec.Command(args[0], args[1:]...)
	proc.Dir = workdir
	logger.Printf("Command path: %v\n", proc.Path)

	output := execCommand(proc, input, inputDemand, outputDemand, closeStdin)
	go inputCommandDispatcher(input, outputDemand, done, closeStdin)
	go outputCommandDispatcher(output, inputDemand, done)

	// wait for pipline to exit
	<-done

	err := safeExit(proc)
	if e, ok := err.(*exec.Error); ok {
		// This shouldn't really happen in practice because we check for
		// program existence in Elixir, before launching odu
		logger.Printf("Command exited with error: %v\n", e)
		os.Exit(3)
	}
	// TODO: return Stderr and exit stauts to beam process
	logger.Printf("Command exited: %#v\n", err)
	return err
}

func inputCommandDispatcher(input chan<- []byte, outputDemand chan<- bool, done chan struct{}, closeStdin chan struct{}) {
	stdinChan := stdinReader(done)

	var cmd Command
	var ok bool

	cmdStdinClosed := false

	defer func() {
		if !cmdStdinClosed {
			close(closeStdin)
		}
	}()

	for {
		select {
		case cmd, ok = <-stdinChan:
			if !ok {
				close(done)
				return
			}
		case <-done:
			return
		}

		switch cmd.tag {
		case CloseInput:
			if !cmdStdinClosed {
				logger.Printf(" --> CLOSE_INPUT\n")
				close(closeStdin)
				cmdStdinClosed = true
			}
		case Input:
			logger.Printf(" --> INPUT %v\n", len(cmd.data))
			// we use buffered input because this should not block
			input <- cmd.data
		case SendOutput:
			logger.Printf(" --> SEND_OUTPUT\n")
			// we use buffered outputDemand because this should not block
			select {
			case outputDemand <- true:
			default:
				fatal("outputDemand channel is full")
			}
		}
	}

}

func outputCommandDispatcher(output <-chan []byte, inputDemand <-chan bool, done chan struct{}) {
	stdoutChan := stdoutWriter()

	defer func() {
		close(done)
	}()

	for {
		select {
		case <-inputDemand:
			logger.Printf("<--  SEND_INPUT\n")
			stdoutChan <- Command{SendInput, make([]byte, 0)}
		case out, ok := <-output:
			logger.Printf("<--  OUTPUT\n")
			if !ok {
				stdoutChan <- Command{OutputEOF, make([]byte, 0)}
				return
			}
			stdoutChan <- Command{Output, out}
		}
	}
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

func read16Be(data []byte) uint16 {
	return binary.BigEndian.Uint16(data)
}

func read32Be(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

func write16Be(data []byte, num int) {
	data[0] = byte(num >> 8)
	data[1] = byte(num)
}
