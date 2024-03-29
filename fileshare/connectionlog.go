package fileshare

import (
	"fmt"
	"os"
)

var filename_const string = "log/connectionlog"

func OpenConnectionStatusLog() *os.File {
	i := 0
	for {
		_, err := os.Stat(filename_const + fmt.Sprintf("%d", i) + ".txt")
		if err != nil {
			break
		} else {
			i++
		}
	}
	file, err := os.Create(filename_const + fmt.Sprintf("%d", i) + ".txt")
	if err != nil {
		fmt.Println("Error while opening the file", filename_const)
	}
	fmt.Println("Using [", filename_const, "] for connection status log")
	return file
}
