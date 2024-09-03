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

const (
	New       FileState = 0
	Resume    FileState = 1
	Completed FileState = 3
	Corrupted FileState = 4
)

type FileState uint8

type FileIOs []*FileIO

type FileIO struct {
	*os.File
	ActiveWriter       *int
	WriteSIG           chan struct{}
	StartByte, EndByte int
	State              FileState
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

func WriteToFile(f *FileIO, r *DataStream, wg *sync.WaitGroup) {
	defer wg.Done()
	f.addWriter(1)
	f.WriteSIG <- struct{}{}
	f.Seek(0, io.SeekEnd)
	f.ReadFrom(r.PipeReader)
	f.addWriter(-1)
}

func (f *FileIO) addWriter(n int) {
	*f.ActiveWriter += n
}

func (f *FileIO) getSize() int {
	fi, err := f.Stat()
	doHandle(err)
	return int(fi.Size())
}

func (fs FileIOs) getTotalWriter() int {
	totalWriter := 0
	for _, f := range fs {
		totalWriter += *f.ActiveWriter
	}
	return totalWriter
}

func (fs FileIOs) setInitState() {

	for _, f := range fs {
		size := f.getSize()
		sb := f.StartByte
		eb := f.EndByte

		if sb > eb {
			f.State = Corrupted
		} else if size > eb-sb+1 {
			f.State = Corrupted
		} else if size == eb-sb+1 {
			f.State = Completed
		} else if size > 0 {
			f.State = Resume
		} else if size == 0 {
			f.State = New
		}

	}

}

func (fs FileIOs) setByteRange(byteCount int) {

	partCount := len(fs)
	partSize := byteCount / partCount
	var rangeStart, rangeEnd int

	for i, ii := 0, 0; i < partCount; i, ii = i+1, ii+partSize {

		if i+1 == partCount {
			rangeStart = ii
			rangeEnd = byteCount - 1
		} else {
			rangeStart = ii
			rangeEnd = (rangeStart - 1) + partSize
		}

		fs[i].StartByte = rangeStart
		fs[i].EndByte = rangeEnd

	}
}
