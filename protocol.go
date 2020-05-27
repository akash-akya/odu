package main

import (
	"encoding/binary"
	"io"
	"os"
)

const SendInput = 1
const SendOutput = 2
const Output = 3
const Input = 4
const CloseInput = 5
const OutputEOF = 6

// This size is *NOT* related to pipe buffer size
const BufferSize = 1 << 16

type Command struct {
	tag  uint8
	data []byte
}

func stdinReader(done <-chan struct{}) <-chan Command {
	stdinChan := make(chan Command)

	go func() {
		defer func() {
			close(stdinChan)
		}()

		var readErr error
		var length uint32
		var tag uint8

		for {
			select {
			case <-done:
				return
			default:
			}

			inputBuf := make([]byte, BufferSize)

			length, readErr = readUint32(os.Stdin)
			if readErr == io.EOF {
				return
			} else if readErr != nil {
				fatal(readErr)
			}

			if length < 1 || length > (BufferSize-4) { // payload must be atleast tag size
				fatal("input payload size is invalid")
			}

			tag, readErr = readUint8(os.Stdin)
			if readErr != nil {
				fatal(readErr)
			}

			_, readErr := io.ReadFull(os.Stdin, inputBuf[:length-1])
			if readErr != nil {
				fatal(readErr)
			}

			stdinChan <- Command{tag, inputBuf[:length-1]}
		}
	}()

	return stdinChan
}

func stdoutWriter() chan<- Command {
	stdoutChan := make(chan Command)

	go func() {
		var cmd Command
		var ok bool
		buf := make([]byte, BufferSize)

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

			payloadLen := len(cmd.data) + 1
			total := payloadLen + 4

			if total > BufferSize {
				fatal("Invalid payloadLen")
			}

			writeUint32Be(buf[:4], uint32(payloadLen))
			writeUint8Be(buf[4:5], cmd.tag)

			copy(buf[5:], cmd.data)

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

func readUint32(stdin io.Reader) (uint32, error) {
	var buf [4]byte

	bytesRead, readErr := io.ReadFull(stdin, buf[:])
	if readErr != nil {
		return 0, io.EOF
	} else if bytesRead == 0 {
		return 0, readErr
	}
	return binary.BigEndian.Uint32(buf[:]), nil
}

func readUint8(stdin io.Reader) (uint8, error) {
	var buf [1]byte

	bytesRead, readErr := io.ReadFull(stdin, buf[:])
	if readErr != nil {
		return 0, io.EOF
	} else if bytesRead == 0 {
		return 0, readErr
	}
	return uint8(buf[0]), nil
}

func writeUint32Be(data []byte, num uint32) {
	binary.BigEndian.PutUint32(data, num)
}

func writeUint8Be(data []byte, num uint8) {
	data[0] = byte(num)
}
