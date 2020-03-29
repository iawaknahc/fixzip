package main

// This program takes a zipfile and uses an external program uchardet to detect filename encoding.
// Once the filename encoding is determined, a copy of the zipfile is produced with filename
// encoded in UTF-8.

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

func DetectEncoding(b []byte) (name string, encoding encoding.Encoding, err error) {
	cmd := exec.Command("uchardet")
	cmd.Stdin = bytes.NewReader(b)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	s := string(out)
	s = strings.TrimSpace(s)
	encoding, err = ianaindex.IANA.Encoding(s)
	if err != nil {
		return
	}
	name, err = ianaindex.IANA.Name(encoding)
	if err != nil {
		return
	}
	return
}

func FixZip(in string, out string) (err error) {
	reader, err := zip.OpenReader(in)
	if err != nil {
		return
	}
	defer reader.Close()

	outFile, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return
	}
	defer outFile.Close()

	writer := zip.NewWriter(outFile)
	defer writer.Close()

	var encodingName string
	var encoding encoding.Encoding
	for _, readerFile := range reader.File {
		b := []byte(readerFile.FileHeader.Name)

		// Assume all filenames have the same encoding.
		if encoding == nil {
			encodingName, encoding, err = DetectEncoding(b)
			if err != nil {
				return
			}
			fmt.Printf("%s\n", encodingName)
		}

		utf8Bytes, err := encoding.NewDecoder().Bytes(b)
		if err != nil {
			return err
		}
		fileHeader := zip.FileHeader{
			Name:     string(utf8Bytes),
			Comment:  readerFile.FileHeader.Comment,
			Method:   readerFile.FileHeader.Method,
			Modified: readerFile.FileHeader.Modified,
		}
		w, err := writer.CreateHeader(&fileHeader)
		if err != nil {
			return err
		}

		readCloser, err := readerFile.Open()
		if err != nil {
			return err
		}
		defer readCloser.Close()

		io.Copy(w, readCloser)
	}

	return
}

func main() {
	var in string
	var out string
	flag.StringVar(&in, "in", "", "input zip file")
	flag.StringVar(&out, "out", "", "output zip file")
	flag.Parse()
	err := FixZip(in, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
