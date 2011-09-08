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
	bfin  *bufio.Reader
	cfin  *os.File
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

func (sbz *SegmentedBzReader) ReadString() (string, os.Error) {
	if sbz.bfin == nil {
		return "", os.EOF
	}
	str, err := sbz.bfin.ReadString('\n')
	if err == nil {
		return str, nil
	}
	if err != os.EOF {
		fmt.Printf("Index %d: Non-EOF error!\n", sbz.Index)
		fmt.Printf("str: '%v' err: '%v'\n", str, err)
		panic("Unrecoverable error")
	}

	sbz.Index += 1
	sbz.OpenNext()

	// Last file?
	if err != nil || sbz.cfin == nil {
		return str, nil
	}

	nstr, nerr := sbz.bfin.ReadString('\n')

	str = fmt.Sprintf("%v%v", str, nstr)

	return str, nerr
}

func (sbz *SegmentedBzReader) Close() {
	sbz.cfin.Close()
	sbz.cfin = nil
	sbz.bfin = nil
}
