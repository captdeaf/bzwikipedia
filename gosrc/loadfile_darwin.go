// loadfile_darwin.go

package loadfile

import (
	"fmt"
	"os"
	"syscall"
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

	// How should we approach this? We have a few options:
	//  mmap: Use disk. Less memory, but slower access.
	//  ram: Read into RAM. A lot more memory, but faster access.
	if dommap {
		// Try to mmap.
		addr, errno := syscall.Mmap(
			fin.Fd(),
			0,
			int(file_size),
			syscall.PROT_READ,
			syscall.MAP_PRIVATE)
		if errno == 0 {
			file_blob = addr
			fmt.Printf("Successfully mmaped!\n")
		} else {
			fmt.Printf("Unable to mmap! error: '%v'\n", os.Errno(errno))
			dommap = false
		}
	}
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
