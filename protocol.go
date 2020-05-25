package main

import (
	"io"
	"os"
)

const SendInput = 1
const SendOutput = 2
const Output = 3
const Input = 4
const CloseInput = 5
const OutputEOF = 6

type Command struct {
	tag  uint16
	data []byte
}

func stdinReader(done <-chan struct{}) <-chan Command {
	stdinChan := make(chan Command)

	go func() {
		defer func() {
			close(stdinChan)
		}()

		var readErr error
		var length uint16
		var tag uint16

		for {
			select {
			case <-done:
				return
			default:
			}

			inputBuf := make([]byte, 16384-1)

			length, readErr = readUint16(os.Stdin)
			if readErr == io.EOF {
				return
			} else if readErr != nil {
				fatal(readErr)
			}

			if length < 2 || length > 16384 { // payload must be atleast tag size
				fatal("input payload size is invalid")
			}

			tag, readErr = readUint16(os.Stdin)
			if readErr != nil {
				fatal(readErr)
			}

			_, readErr := io.ReadFull(os.Stdin, inputBuf[:length-2])
			if readErr != nil {
				fatal(readErr)
			}

			stdinChan <- Command{tag, inputBuf[:length-2]}
		}
	}()

	return stdinChan
}

func stdoutWriter() chan<- Command {
	stdoutChan := make(chan Command)

	go func() {
		var cmd Command
		var ok bool
		buf := make([]byte, 16384)

		defer func() {
			close(stdoutChan)
		}()

		for {
			select {
			case cmd, ok = <-stdoutChan:
				if !ok {
					return
				}
			}

			payloadLen := len(cmd.data) + 2
			total := payloadLen + 2

			if total > 16384 {
				fatal("Invalid payloadLen")
			}

			write16Be(buf[:2], payloadLen)
			write16Be(buf[2:4], int(cmd.tag))
			copy(buf[4:], cmd.data)

			_, writeErr := os.Stdout.Write(buf[:total])
			if writeErr != nil {
				switch writeErr.(type) {
				// ignore broken pipe or closed pipe errors
				case *os.PathError:
					return
				default:
					fatal(writeErr)
				}
			}
			// logger.Printf("stdout written bytes: %v\n", bytesWritten)
		}
	}()

	return stdoutChan
}

func readUint16(stdin io.Reader) (uint16, error) {
	buf := make([]byte, 2)

	bytesRead, readErr := io.ReadFull(stdin, buf)
	if readErr != nil {
		return 0, io.EOF
	} else if bytesRead == 0 {
		return 0, readErr
	}
	return read16Be(buf), nil
}
