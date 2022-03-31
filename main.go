package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const BUFFER_SIZE = 16
const NUM_OF_LINES = 10
const WHENCE = 2

func main() {
	if len(os.Args) < 2 {
		log.Fatal("file name missing in the command")
	}
	fileName := os.Args[1]
	checkFileExists(fileName)
	readFile(fileName)
	watchFile(fileName)
}

//Watch file for changes
func watchFile(fileName string) {
	fd, err := unix.InotifyInit()
	if err != nil {
		log.Fatal(err)
	}
	defer unix.Close(fd)
	wd, err := unix.InotifyAddWatch(fd, fileName, unix.IN_ALL_EVENTS)
	if err != nil {
		log.Fatal(err)
	}
	defer unix.InotifyRmWatch(fd, uint32(wd))
	// fmt.Printf("WD is %d\n", wd)

	for {
		// Room for at least 128 events
		buffer := make([]byte, unix.SizeofInotifyEvent*128)
		bytesRead, err := unix.Read(fd, buffer)
		if err != nil {
			log.Fatal(err)
		}

		if bytesRead < unix.SizeofInotifyEvent {
			// No point trying if we don't have at least one event
			continue
		}

		// fmt.Printf("Size of InotifyEvent is %s\n", unix.SizeofInotifyEvent)
		// fmt.Printf("Bytes read: %d\n", bytesRead)

		offset := 0
		for offset < bytesRead-unix.SizeofInotifyEvent {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buffer[offset]))
			// fmt.Printf("%+v\n", event)

			if (event.Mask & unix.IN_MODIFY) > 0 {
				fmt.Printf("\nFile has been modified\n")
				time.Sleep(time.Second * 2)
				readFile(fileName)
			}

			// We need to account for the length of the name
			offset += unix.SizeofInotifyEvent + int(event.Len)
		}
	}
}

func checkFileExists(fileName string) {
	_, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("%s file does not exist.", fileName)
		}
	}
}

func getFileSize(fileName string) int64 {
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		log.Fatal(err)
	}
	return fileInfo.Size()
}
func readData(file *os.File, pos int64, filesize int64) ([]byte, bool) {
	// if seek position extends the filesize
	var done bool = false
	if int64(math.Abs(float64(pos))) > filesize {
		_, err := file.Seek(-filesize, 2)
		if err != nil {
			log.Fatal(err)
		}
		done = true
	} else {
		_, err := file.Seek(pos, 2)
		if err != nil {
			log.Fatal(err)
		}
	}

	buffer := make([]byte, BUFFER_SIZE)
	//Read file data in buffer
	_, err := file.Read(buffer)
	if err != nil {
		if err == io.EOF {
			return buffer, true
		}
		log.Fatal(err)
	}
	return buffer, done
}

func readLine(buffer []byte, start, end int) (string, int) {
	pos := bytes.LastIndexByte(buffer[start:end], '\n')
	if pos >= 0 {
		return string(buffer[pos+1 : end]), pos
	}
	return "", 0
}

func printLines(output []string) {
	for i := len(output) - 1; i >= 0; i-- {
		fmt.Println(output[i])
	}
}

func readFile(fileName string) {
	//Open file
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var position int64 = 0
	var lineNum int = 1

	fileSize := getFileSize(fileName)

	position -= BUFFER_SIZE
	buffer, done := readData(file, position, fileSize)
	if done {
		fmt.Printf("%s", string(buffer))
		return
	}
	end := len(buffer)
	output := make([]string, 0)
	for lineNum <= NUM_OF_LINES {
		line, pos := readLine(buffer, 0, end)
		// if buffer is has no more lines to read then update buffer with new next data
		if line == "" && pos == 0 {
			if done {
				break
			}
			position = position - BUFFER_SIZE + (int64(end))
			buffer, done = readData(file, position, fileSize)
			end = len(buffer)
		} else {
			end = pos
			output = append(output, line)
			lineNum++
		}

	}
	printLines(output)
}
