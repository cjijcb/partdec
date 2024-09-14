package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

type (
	FileState uint8
	FilePath  struct {
		Base, DstDir, Relative string
	}

	ByteRange struct {
		Start, End, Offset int
	}

	FileIO struct {
		*os.File
		Scope      ByteRange
		State      FileState
		Path       FilePath
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

func buildFileIOs(partCount int, basePath string, dstDirs []string) (FileIOs, error) {

	var err error

	if dstDirs == nil {
		dstDirs = []string{"."}
	}

	fios := make([]*FileIO, partCount)
	dirCount := len(dstDirs)
	freqDistrib := partCount / dirCount
	xtraDistrib := partCount % dirCount

	var idx, xtra int
	for _, dir := range dstDirs {

		xtra = 0
		if xtraDistrib > 0 {
			xtra = 1
			xtraDistrib--
		}

		for range freqDistrib + xtra {

			suffix := fmt.Sprintf("_%d", idx)
			basePathSfx := filepath.Clean(basePath + suffix)

			fio, e := buildFileIO(basePathSfx, dir, os.O_WRONLY)
			err = errors.Join(err, e)

			fios[idx] = fio

			idx++
		}

	}

	return fios, err

}


func buildFileIO(basePath string, dstDir string, oflag int) (*FileIO, error) {

	pathSpr := string(os.PathSeparator)

	basePath = filepath.Clean(basePath)
	dstDir 	= filepath.Clean(dstDir) + pathSpr
	relvPath := filepath.Clean(dstDir + basePath)

	f, err := os.OpenFile(relvPath, os.O_CREATE|oflag, 0640)

	fio := &FileIO{
		File:       f,
		Path:		FilePath{
						Base: basePath,
						DstDir: dstDir,
						Relative: relvPath,
					},
		ClosingSIG: make(chan bool, 1),
	}
	
	return fio, err
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
