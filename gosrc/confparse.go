// confparse.go
//
// Copyright 2011 Martin Schoenmakers <aiviru@gmail.com>
//
// Parse a .conf file with the following format:
//
// # Comment
// key : value
//
// Empty strings and lines beginning with # are ignored.
//
// confparse.ParseFile(filename) returns a map[string] string and err.

package confparse

import (
	"os"
	"io"
	"bufio"
	"strings"
)

func ParseFile(fileName string) (data map[string]string, err os.Error) {
	data = make(map[string]string)

	file, err := os.Open(fileName)
	if err != nil {
		return data, err
	}
	defer file.Close()

	err = ParseIO(file, data)

	return data, err
}

func ParseIO(in io.Reader, data map[string]string) (err os.Error) {
	inBuf := bufio.NewReader(in)

	for {
		line, err := inBuf.ReadString('\n')
		k, v := keyValue(line)
		if v != "" {
			data[k] = v
		}

		if err == os.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func keyValue(in string) (key, value string) {
	colon := strings.Index(in, ":")

	if colon == -1 {
		return
	}

	key = strings.TrimSpace(in[:colon])
	value = strings.TrimSpace(in[colon+1:])

	if key[0] == '#' {
		return "", ""
	}
	return
}
