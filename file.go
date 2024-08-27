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

type FileXtd struct {
	*os.File
	ActiveWriter *int
	WriteSIG     chan struct{}
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

func buildFile(name string) *FileXtd {

	f, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)

	doHandle(err)

	file := &FileXtd{
		File:         f,
		ActiveWriter: new(int),
		WriteSIG:     make(chan struct{}),
	}
	//defer file.Close()
	return file
}

func doWriteFile(f *FileXtd, chR chan io.ReadCloser, wg *sync.WaitGroup) {
	defer wg.Done()
	f.addWriter(1)
	f.WriteSIG <- struct{}{}
	io.Copy(f, <-chR)
	f.addWriter(-1)
	f.Sync()
}

func (f *FileXtd) addWriter(n int) {
	*f.ActiveWriter += n
}

func getFileSize(f *FileXtd) int64 {
	fi, err := f.Stat()
	doHandle(err)
	return fi.Size()
}

func getTotalWriter(fs []*FileXtd) int {
	totalWriter := 0
	for _, v := range fs {
		totalWriter += *v.ActiveWriter
	}
	return totalWriter

}
