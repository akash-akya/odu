package main

import (
	"encoding/binary"
	"io"
	"os"
)

var buf [1 << 16]byte

func startOutputStreamer(pipe io.ReadCloser, writer io.WriteCloser) <-chan struct{} {
	exit := make(chan struct{})
	go func() {
		defer func() {
			pipe.Close()
			writer.Close()
			close(exit)
		}()

		for {
			bytesRead, readErr := pipe.Read(buf[2:])
			if bytesRead > 0 {
				write16Be(buf[:2], bytesRead)
				bytesWritten, writeErr := writer.Write(buf[:bytesRead+2])
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

			} else if readErr == io.EOF && bytesRead == 0 {
				return
			} else {
				fatal(readErr)
			}
		}
	}()
	return exit
}

func startInputConsumer(pipe io.WriteCloser, reader io.ReadCloser) {
	buf := make([]byte, 2)

	go func() {
		defer func() {
			reader.Close()
			pipe.Close()
		}()

		for {
			bytesRead, readErr := io.ReadFull(reader, buf)
			if readErr == io.EOF && bytesRead == 0 {
				return
			}
			fatalIf(readErr)

			length := read16Be(buf)
			logger.Printf("[cmd_in] read packet length = %v\n", length)
			if length == 0 {
				return
			}

			_, writeErr := io.CopyN(pipe, reader, int64(length))
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

func startErrorStreamer(pipe io.ReadCloser, writer io.WriteCloser) {
	go func() {
		defer func() {
			pipe.Close()
			writer.Close()
		}()

		for {
			bytesRead, readErr := pipe.Read(buf[2:])
			if bytesRead > 0 {
				write16Be(buf[:2], bytesRead)
				bytesWritten, writeErr := writer.Write(buf[:bytesRead+2])
				if writeErr != nil {
					switch writeErr.(type) {
					// ignore broken pipe or closed pipe errors
					case *os.PathError:
						return
					default:
						fatal(writeErr)
					}
				}
				logger.Printf("[cmd_err] written bytes: %v\n", bytesWritten)

			} else if readErr == io.EOF && bytesRead == 0 {
				return
			} else if readErr != nil {
				switch readErr.(type) {
				// ignore broken pipe or closed pipe errors
				case *os.PathError:
					return
				default:
					fatal(readErr)
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
