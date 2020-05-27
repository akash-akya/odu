package main

import (
	"io"
	"os"
	"os/exec"
)

func execCommand(proc *exec.Cmd, input <-chan []byte, inputDemand chan<- bool, outputDemand <-chan bool, close_stdin chan struct{}) <-chan []byte {
	cmdInput, err := proc.StdinPipe()
	fatalIf(err)

	cmdOutput, err := proc.StdoutPipe()
	fatalIf(err)

	// cmdError, err := proc.StderrPipe()
	// fatalIf(err)

	execErr := proc.Start()
	fatalIf(execErr)

	output := make(chan []byte)

	go func() {
		var buf [BufferSize - 1]byte
		var pipe io.ReadCloser

		pipe = cmdOutput

		defer func() {
			cmdOutput.Close()
			close(output)
		}()

		for {
			<-outputDemand

			// blocking
			bytesRead, readErr := pipe.Read(buf[:])
			if bytesRead > 0 {
				output <- buf[:bytesRead]
			} else if readErr == io.EOF || bytesRead == 0 {
				return
			} else {
				fatal(readErr)
			}
		}
	}()

	go func() {
		var in []byte
		var ok bool

		defer func() {
			cmdInput.Close()
		}()

		inputDemand <- true

		for {
			select {
			case <-close_stdin:
				return
			case in, ok = <-input:
				if !ok {
					return
				}
			}

			// blocking
			_, writeErr := cmdInput.Write(in)
			if writeErr != nil {
				switch writeErr.(type) {
				// ignore broken pipe or closed pipe errors
				case *os.PathError:
					return
				default:
					fatal(writeErr)
				}
			}
			inputDemand <- true
		}
	}()

	return output
}
