package stream

import (
	"bufio"
	"fmt"
	"github.com/libp2p/go-libp2p/core/network"
	"io"
	"os"
	"time"
)

var filename_send string = ""

func sendToStream(str network.Stream) {
	fmt.Println("Sending file to ", str.Conn().RemotePeer())
	filesize, err := getByteSize(filename_send)
	if err != nil {
		fmt.Println("Error while walking file")
	}
	fmt.Println("Sending file :", filename_send, "Of size-", filesize)
	buffersize := filesize / 10000
	sendBytes := make([]byte, 10000)
	file, err := os.Open(filename_send)
	if err != nil {
		fmt.Println("Error while opening the sending file")
	} else {
		bufstr := bufio.NewWriter(str)
		for i := 0; i < buffersize; i++ {
			_, err = file.Read(sendBytes)
			if err == io.EOF {
				fmt.Println("Send the file completely")
				break
			}
			if err != nil {
				fmt.Println("Error while reading from the file")
			}
			_, err = bufstr.Write(sendBytes)
			if err != nil {
				fmt.Println("Error while sending to the stream")
			}
		}
		leftByte := filesize % 10000
		leftbytebuffer := make([]byte, leftByte)
		_, err = file.Read(leftbytebuffer)
		if err == io.EOF {
			fmt.Println("Send the file completely")
		}
		if err != nil {
			fmt.Println("Error while reading from the file")
		}
		_, err = bufstr.Write(leftbytebuffer)
		if err != nil {
			fmt.Println("Error while sending to the stream")
		}
		fmt.Println("Closing the stream")
		str.Close()
	}

}

func getByteSize(filename string) (int, error) {
	file, err := os.Stat(filename)
	if err != nil {
		fmt.Println("Error while walking the file - file doesn't exist")
		return 0, err
	}
	return int(file.Size()), nil
}

func ReceivedFromStream(str network.Stream, filename string, filetype string, logfile *os.File, filesize int) {
	fullfilename := filename + "." + filetype
	file, err := os.Create(fullfilename)
	fmt.Fprintln(logfile, "Recieving file from stream to :", str.Conn().RemotePeer())
	buffersize := filesize / 10000
	readBytes := make([]byte, 10000)
	if err != nil {
		fmt.Fprintln(logfile, "Error while creating the recieving file")
	} else {
		bufstr := bufio.NewReader(str)
		for i := 0; i < buffersize; i++ {
			_, err := bufstr.Read(readBytes)
			if err == io.EOF {
				fmt.Fprintln(logfile, "End of file reached")
				break
			}
			if err != nil {
				fmt.Fprintln(logfile, "Error while reading from stream")
				break
			}
			fmt.Print(readBytes)
			_, err = file.Write(readBytes)
			if err != nil {
				fmt.Fprintln(logfile, "Error while writing to the stream")
			}

		}
		leftByte := filesize % 10000
		leftbytebuffer := make([]byte, leftByte)
		_, err := bufstr.Read(leftbytebuffer)
		if err == io.EOF {
			fmt.Fprintln(logfile, "End of file reached after leftbyte read")
		}
		if err != nil {
			fmt.Fprintln(logfile, "Error while reading from stream after loop")
		}
		fmt.Print(leftbytebuffer)
		_, err = file.Write(leftbytebuffer)
		if err != nil {
			fmt.Fprintln(logfile, "Error while writing to the file after loop")
		}
		fmt.Fprintln(logfile, "Completed reading from the stream")
		fmt.Println("Completed reading from the stream")
	}
	file.Close()
	time.Sleep(1 * time.Minute)

}
