package main

import (
	"encoding/binary"
	"io"
	"os"
)

var buf [1 << 16]byte

func startOutputStreamer(pipe io.ReadCloser, fifo *os.File) <-chan struct{} {
	exit := make(chan struct{})
	go func() {
		defer func() {
			pipe.Close()
			fifo.Close()
			close(exit)
		}()

		for {
			bytesRead, ReadErr := pipe.Read(buf[2:])
			if bytesRead > 0 {
				write16Be(buf[:2], bytesRead)
				bytesWritten, writeErr := fifo.Write(buf[:bytesRead+2])
				if writeErr != nil {
					switch writeErr.(type) {
					// ignore broken pipe or closed pipe errors
					case *os.PathError:
						return
					default:
						fatal(writeErr)
					}
				}
				logger.Printf("[cmd_out] written bytes: %v\n", bytesWritten)

			} else if ReadErr == io.EOF && bytesRead == 0 {
				return
			} else {
				fatal(ReadErr)
			}
		}
	}()
	return exit
}

func startInputConsumer(pipe io.WriteCloser, fifo *os.File) {
	buf := make([]byte, 2)

	go func() {
		defer func() {
			fifo.Close()
			pipe.Close()
		}()

		for {
			bytesRead, readErr := io.ReadFull(fifo, buf)
			if readErr == io.EOF && bytesRead == 0 {
				return
			}
			fatal_if(readErr)

			length := read16Be(buf)
			logger.Printf("[cmd_in] read packet length = %v\n", length)
			if length == 0 {
				return
			}

			_, writeErr := io.CopyN(pipe, fifo, int64(length))
			if writeErr != nil {
				switch writeErr.(type) {
				// ignore broken pipe or closed pipe errors
				case *os.PathError:
					return
				default:
					fatal(writeErr)
				}
			}
		}
	}()
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
