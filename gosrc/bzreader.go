// bzreader.go
//
// Author: Greg Millam <captdeaf@gmail.com>

package bzreader

import (
	"bufio"
	"compress/bzip2"
	"fmt"
	"os"
)

type SegmentedBzReader struct {
	Index  int
	bfin   *bufio.Reader
	cfin   *os.File
	path   string
	dbname string
}

//
// This will sequentially read .bz2 files starting from a given index.
//
func NewBzReader(path, dbname string, index int) *SegmentedBzReader {
	sbz := new(SegmentedBzReader)
	sbz.Index = index
	sbz.bfin = nil
	sbz.cfin = nil
	sbz.path = path
	sbz.dbname = dbname

	sbz.OpenNext()
	return sbz
}

//
// Open rec<index>dbname.xml.bz2 for reading
//
func (sbz *SegmentedBzReader) OpenNext() {
	if sbz.cfin != nil {
		sbz.cfin.Close()
		sbz.cfin = nil
		sbz.bfin = nil
	}
	fn := fmt.Sprintf("%v/rec%05d%v", sbz.path, sbz.Index, sbz.dbname)
	cfin, err := os.Open(fn)
	if err != nil {
		sbz.cfin = nil
		sbz.bfin = nil
	} else {
		sbz.cfin = cfin
		sbz.bfin = bufio.NewReader(bzip2.NewReader(cfin))
	}
}

// Return a byte array. This does not create a copy unless necessary,
// so it should be faster and less memory consuming for scanning.
func (sbz *SegmentedBzReader) ReadBytes() ([]byte, os.Error) {
	if sbz.bfin == nil {
		return nil, os.EOF
	}

	line, err := sbz.bfin.ReadSlice('\n')

	// Most common case: Quick line read.

	if err == nil {
		return line, nil
	}

	buff := make([]byte, len(line))
	copy(buff, line)

	// If it was a buffer issue, that's easy enough to fix. 
	if err == bufio.ErrBufferFull {
		var nbytes []byte
		nbytes, err = sbz.bfin.ReadBytes('\n')
		nbuff := make([]byte, len(buff)+len(nbytes))
		copy(nbuff, buff)
		copy(nbuff[len(buff):], nbytes)
		buff = nbuff
	}

	if err == os.EOF {
		sbz.Index += 1
		sbz.OpenNext()
		if sbz.cfin == nil {
			return buff, os.EOF
		}
		var nbytes []byte
		nbytes, err = sbz.bfin.ReadBytes('\n')
		nbuff := make([]byte, len(buff)+len(nbytes))
		copy(nbuff, buff)
		copy(nbuff[len(buff):], nbytes)
		buff = nbuff
	}

	if err == nil {
		return buff, nil
	}

	fmt.Printf("Unknown Error in bzreader.ReadBytes? '%v'\n", err)

	return nil, os.EOF
}

func (sbz *SegmentedBzReader) ReadString() (string, os.Error) {
	s, e := sbz.ReadBytes()
	return string(s), e
}

func (sbz *SegmentedBzReader) Close() {
	sbz.cfin.Close()
	sbz.cfin = nil
	sbz.bfin = nil
}
