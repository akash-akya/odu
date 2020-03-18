package main

import (
	"encoding/binary"
	"io"
	"os"
)

var outBuf [1 << 16]byte

func startOutputStreamer(pipe io.ReadCloser, outstream io.Writer, maxChunkSize int, demand <-chan int) <-chan struct{} {
	exit := make(chan struct{})
	const stdoutMarker = 0x00

	go func() {
		defer close(exit)

		buf := outBuf
		buf[2] = stdoutMarker
		pending := 0

		for {
			if pending == 0 {
				d, ok := <-demand
				if !ok {
					return
				}
				pending = d
			} else if pending < maxChunkSize {
				select {
				case d, ok := <-demand:
					if !ok {
						return
					}
					pending += d
				default:
				}
			}

			chunkSize := 0
			if pending > maxChunkSize {
				chunkSize = maxChunkSize
			} else {
				chunkSize = pending
			}

			readBytes, readErr := pipe.Read(buf[3 : 3+chunkSize])

			if readBytes > 0 {
				write16Be(buf[:2], readBytes+1)
				bytesWritten, writeErr := outstream.Write(buf[:2+readBytes+1])
				logger.Printf("out: written bytes: %v\n", bytesWritten)
				fatal_if(writeErr)
				pending -= readBytes
			} else if readErr == io.EOF || readBytes == 0 {
				// From io.Reader docs:
				//
				//   Implementations of Read are discouraged from returning a zero
				//   byte count with a nil error, and callers should treat that
				//   situation as a no-op.
				//
				// In this case it appears that 0 bytes may sometimes be returned
				// indefinitely. Therefore we close the pipe.
				if readErr == io.EOF {
					logger.Println("Encountered EOF when reading from stdout")
				} else {
					logger.Println("Read 0 bytes with no error")
				}
				return
			} else {
				switch readErr.(type) {
				case *os.PathError:
					return
				default:
					fatal(readErr)
				}
			}
		}
		logger.Println("Exiting output streamer")
	}()
	return exit
}

func startCommandConsumer(stdin io.Reader) (<-chan int, <-chan struct{}) {
	demand := make(chan int)
	exitSignal := make(chan struct{})
	buf := make([]byte, 4)

	go func() {
		defer close(exitSignal)
		defer close(demand)

		for {
			bytesRead, readErr := io.ReadFull(stdin, buf[:2])
			logger.Printf("READ stdin %v bytes", bytesRead)
			if readErr == io.EOF && bytesRead == 0 {
				logger.Printf("[STDIN] EOF")
				return
			}
			fatal_if(readErr)

			length := read16Be(buf[:2])

			if length == 0 {
				return
			}

			// TODO: must be 2 since no commands are supported now
			if length != 4 {
				fatal("Invalid data length")
			}

			bytesRead, readErr = io.ReadFull(stdin, buf[:4])
			fatal_if(readErr)

			demand <- int(read32Be(buf[:4]))
		}
		logger.Println("Exiting command consumer")
	}()

	return aggregate(demand), exitSignal
}

func aggregate(demand <-chan int) <-chan int {
	out := make(chan int, 1)
	go func() {
		defer close(out)
		for d := range demand {
			select {
			case existing := <-out:
				out <- (existing + d)
			default:
				out <- d
			}
		}
		logger.Println("Exiting aggregator")
	}()
	return out
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
