package main

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
)

type byteOffsetStart = int
type byteOffsetEnd = int

type FileIOs []*FileIO

type FileIO struct {
	*os.File
	ActiveWriter *int
	WriteSIG     chan struct{}
	bOffS        byteOffsetStart
	bOffE        byteOffsetEnd
}

func buildFileName(rawURL string, hdr *http.Header) string {
	_, params, _ := mime.ParseMediaType(hdr.Get("Content-Disposition"))
	fileName := params["filename"]

	if fileName != "" {
		return fileName
	}

	url, err := url.Parse(rawURL)
	doHandle(err)

	fileName = path.Base(url.Path)
	return fileName

}

func buildFile(name string) *FileIO {

	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0666)

	doHandle(err)

	file := &FileIO{
		File:         f,
		ActiveWriter: new(int),
		WriteSIG:     make(chan struct{}),
	}
	//defer file.Close()
	return file
}

func WriteToFile(f *FileIO, r *io.PipeReader, wg *sync.WaitGroup) {
	defer wg.Done()
	f.addWriter(1)
	f.WriteSIG <- struct{}{}
	//io.Copy(f, r)
	f.Seek(f.getSize(), io.SeekStart)
	f.ReadFrom(r)
	f.addWriter(-1)
	f.Sync()
}

func (f *FileIO) addWriter(n int) {
	*f.ActiveWriter += n
}

func (f *FileIO) getSize() int64 {
	fi, err := f.Stat()
	doHandle(err)
	return fi.Size()
}

func (fs FileIOs) getTotalWriter() int {
	totalWriter := 0
	for _, f := range fs {
		totalWriter += *f.ActiveWriter
	}
	return totalWriter
}

func (fs FileIOs) setByteRange(byteCount int) {

	partCount := len(fs)
	partSize := byteCount / partCount
	var lowerLimit, upperLimit int

	for i, ii := 0, 0; i < partCount; i, ii = i+1, ii+partSize {

		if i+1 == partCount {
			lowerLimit = ii
			upperLimit = byteCount - 1
		} else {
			lowerLimit = ii
			upperLimit = (lowerLimit - 1) + partSize
		}

		fs[i].bOffS = lowerLimit + int(fs[i].getSize())
		fs[i].bOffE = upperLimit

	}
}
