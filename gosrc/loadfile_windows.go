// loadfile_darwin.go

package loadfile

import (
	"fmt"
	"os"
)

func ReadFile(title_file string, dommap bool) (bool, int64, []byte) {
	fin, err := os.Open(title_file)
	if err != nil {
		fmt.Println(err)
		return false, 0, nil
	}
	defer fin.Close()

	// Find out how big it is.
	stat, err := fin.Stat()
	if err != nil {
		fmt.Printf("Error while slurping in title cache: '%v'\n", err)
		return false, 0, nil
	}
	file_size := stat.Size
	var file_blob []byte

	// Windows doesn't support mmap.
	dommap = false

	if !dommap {
		// Default: Load into memory.
		fmt.Printf("Loading titlecache.dat into Memory . . .\n")
		file_blob = make([]byte, file_size, file_size)

		nread, err := fin.Read(file_blob)

		if err != nil && err != os.EOF {
			fmt.Printf("Error while slurping in title cache: '%v'\n", err)
			return false, 0, nil
		}
		if int64(nread) != file_size || err != nil {
			fmt.Printf("Unable to read entire file, only read %d/%d\n",
				nread, stat.Size)
			return false, 0, nil
		}
	}
	return true, file_size, file_blob
}
