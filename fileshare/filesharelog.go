package fileshare

import (
	"fmt"
	"os"
)

var filename_fileshare string = "log/filesharelog"

func OpenFileStatusLog() *os.File {
	i := 0
	for {
		_, err := os.Stat(filename_fileshare + fmt.Sprintf("%d", i) + ".txt")
		if err != nil {
			break
		} else {
			i++
		}
	}
	file, err := os.Create(filename_fileshare + fmt.Sprintf("%d", i) + ".txt")
	if err != nil {
		fmt.Println("Error while opening the file", filename_const)
	}
	fmt.Println("Using [", filename_fileshare, "] for connection status log")
	return file
}
