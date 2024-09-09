package main

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
)

type (
	FileState uint8

	ByteRange struct {
		Start, End, Offset int
	}

	FileIO struct {
		*os.File
		Scope      ByteRange
		State      FileState
		ClosingSIG chan bool
	}

	FileIOs []*FileIO
)

const (
	New FileState = iota
	Resume
	Completed
	Corrupted
	Unknown

	UnknownSize = -1
)

func buildFileName(uri string, hdr http.Header) string {

	if hdr == nil {
		fileName := path.Base(uri)
		return fileName
	}

	_, params, _ := mime.ParseMediaType(hdr.Get("Content-Disposition"))

	if fileName := params["filename"]; fileName != "" {
		return fileName
	} else {
		url, err := url.Parse(uri)
		doHandle(err)
		fileName := path.Base(url.Path)
		return fileName
	}

}

func buildFile(name string, flag int) *FileIO {

	f, err := os.OpenFile(name, os.O_CREATE|flag, 0666)

	doHandle(err)

	fileIO := &FileIO{
		File:       f,
		ClosingSIG: make(chan bool, 1),
	}
	return fileIO
}

func (f *FileIO) getSize() int {
	fi, err := f.Stat()
	doHandle(err)
	return int(fi.Size())
}

func (fs FileIOs) WaitClosingSIG() {
	for _, f := range fs {
		<-f.ClosingSIG
	}
}

func (fs FileIOs) Close() {
	for _, f := range fs {
		f.Close()
		close(f.ClosingSIG)
	}
}

func (f FileIO) DataCast(br ByteRange) io.ReadCloser {

	rangeStart := br.Start + br.Offset
	rangeEnd := br.End

	if rangeStart > rangeEnd {
		rangeStart = rangeEnd
	}

	rangeEnd = rangeEnd - rangeStart + 1

	sr := io.NewSectionReader(f, int64(rangeStart), int64(rangeEnd))

	return io.NopCloser(sr)

}

func (fs FileIOs) setInitState() {

	for _, f := range fs {
		size := f.getSize()
		sb := f.Scope.Start
		eb := f.Scope.End

		if sb == UnknownSize || eb == UnknownSize {
			f.State = Unknown
		} else if sb > eb {
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

	if byteCount == UnknownSize {
		for _, f := range fs {
			f.Scope.Start = UnknownSize
			f.Scope.End = UnknownSize
		}
		return
	}

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
		f := fs[i]

		f.Scope.Start = rangeStart
		f.Scope.End = rangeEnd
		f.Scope.Offset = f.getSize()
	}
}
