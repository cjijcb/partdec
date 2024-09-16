package main

import (
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
		url, _ := url.Parse(uri)
		fileName := path.Base(url.Path)
		return fileName
	}

}

func buildFileIOs(partCount int, basePath string, dstDirs []string) (FileIOs, error) {

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

			fio, err := buildFileIO(basePathSfx, dir, os.O_WRONLY)
			if err != nil {
				return nil, err
			}

			fios[idx] = fio

			idx++
		}

	}

	return fios, nil

}

func buildFileIO(basePath string, dstDir string, oflag int) (*FileIO, error) {

	pathSpr := string(os.PathSeparator)

	basePath = filepath.Clean(basePath)
	dstDir = filepath.Clean(dstDir) + pathSpr
	relvPath := filepath.Clean(dstDir + basePath)

	f, err := os.OpenFile(relvPath, os.O_CREATE|oflag, 0640)

	if err != nil {
		return nil, err
	}

	fio := &FileIO{
		File: f,
		Path: FilePath{
			Base:     basePath,
			DstDir:   dstDir,
			Relative: relvPath,
		},
		ClosingSIG: make(chan bool, 1),
	}

	return fio, err
}

func (f FileIO) DataCast(br ByteRange) (io.ReadCloser, error) {

	rangeStart := br.Start + br.Offset
	rangeEnd := br.End

	if rangeStart > rangeEnd {
		rangeStart = rangeEnd
	}

	rangeEnd = rangeEnd - rangeStart + 1

	sr := io.NewSectionReader(f, int64(rangeStart), int64(rangeEnd))

	return io.NopCloser(sr), nil

}

func (fs FileIOs) setInitState() error {

	for _, f := range fs {
		size, err := f.Size()
		if err != nil {
			return err
		}
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

	return nil

}

func (fs FileIOs) setByteRange(byteCount int) error {

	if byteCount == UnknownSize {
		for _, f := range fs {
			f.Scope.Start = UnknownSize
			f.Scope.End = UnknownSize
		}
		return nil
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
		size, err := f.Size()

		if err != nil {
			return err
		}

		f.Scope.Offset = size
	}

	return nil
}

func (f *FileIO) Size() (int, error) {
	fi, err := f.Stat()
	if err != nil {
		return UnknownSize, nil
	}
	return int(fi.Size()), nil
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
