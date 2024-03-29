package fileshare

import (
	"fmt"
	"os"
)

var filename string = "log/peerconnlog"

func OpenPeerConnectionLog() *os.File {
	i := 0
	for {
		_, err := os.Stat(filename + fmt.Sprintf("%d", i) + ".txt")
		if err != nil {
			break
		} else {
			i++
		}
	}
	file, err := os.Create(filename + fmt.Sprintf("%d", i) + ".txt")
	if err != nil {
		fmt.Println("Error while opening the file", filename)
	}
	fmt.Println("Using [", filename, "] for peer connection log")
	return file
}
