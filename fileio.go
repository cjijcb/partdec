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
		Scope ByteRange
		State FileState
		Path  FilePath
		Err   error
	}

	FileIOs []*FileIO
)

const (
	New FileState = iota
	Resume
	Completed
	Broken
	Unknown

	UnknownSize   = -1
	CurrentDir    = "."
	PathSeparator = string(os.PathSeparator)
)

func NewFileName(uri string, hdr http.Header) string {

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

func BuildFileIOs(partCount int, basePath string, dstDirs []string) (FileIOs, error) {

	if dstDirs == nil {
		dstDirs = []string{CurrentDir}
	}

	fios := make([]*FileIO, partCount)
	dirCount := len(dstDirs)
	fioPerDirCount := partCount / dirCount
	fioExtraCount := partCount % dirCount
	addIndex := FileNameIndexer(partCount)

	var idx uint
	for _, dir := range dstDirs {

		fioExtra := 0
		if fioExtraCount > 0 {
			fioExtra = 1
			fioExtraCount--
		}

		for range fioPerDirCount + fioExtra {

			fio, err := NewFileIO(addIndex(basePath), dir, os.O_WRONLY)
			if err != nil {
				return nil, err
			}

			fios[idx] = fio
			idx++
		}

	}

	return fios, nil

}

func NewFileIO(basePath string, dstDir string, oflag int) (*FileIO, error) {

	basePath = filepath.Clean(basePath)
	dstDir = filepath.Clean(dstDir) + PathSeparator
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
		Err: nil,
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

func (fs FileIOs) RenewByState(sm map[FileState]bool) error {

	for _, f := range fs {
		if sm[f.State] == false {
			continue
		}

		if err := f.Truncate(0); err != nil {
			return err
		}

		f.State = New

	}

	return nil

}

func (fs FileIOs) SetInitState() error {

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
			f.State = Broken
		} else if size > eb-sb+1 {
			f.State = Broken
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

func (fs FileIOs) SetByteRange(dataSize int, partSize int) error {

	if dataSize == UnknownSize {
		for _, f := range fs {
			f.Scope.Start = UnknownSize
			f.Scope.End = UnknownSize
		}
		return nil
	}

	if partSize > 0 {
		if err := fs.setByteRangeByPartSize(dataSize, partSize); err != nil {
			return err
		}
	} else {
		if err := fs.setByteRangeByPartCount(dataSize); err != nil {
			return err
		}
	}

	return nil
}

func (fs FileIOs) setByteRangeByPartCount(dataSize int) error {

	var rangeStart, rangeEnd int

	partCount := len(fs)
	basePartSize := dataSize / partCount
	remainder := dataSize % partCount

	for i, offset := 0, 0; i < partCount; i, offset = i+1, offset+basePartSize {

		extraByte := 0
		if remainder > 0 {
			extraByte = 1
			remainder--
		}

		rangeStart = offset
		rangeEnd = (rangeStart - 1) + basePartSize + extraByte
		offset = offset + extraByte

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

func (fs FileIOs) setByteRangeByPartSize(dataSize int, partSize int) error {

	var rangeStart, rangeEnd int

	partCount := dataSize / partSize
	remainder := dataSize % partSize

	if remainder > 0 {
		partCount++
	}

	for i, offset := 0, 0; i < partCount; i, offset = i+1, offset+partSize {

		if i+1 == partCount {
			rangeStart = offset
			rangeEnd = dataSize - 1
		} else {
			rangeStart = offset
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

func FileNameIndexer(maxIndex int) func(string) string {
	if maxIndex <= 1 {
		return func(name string) string {
			return name
		}
	}

	currentIndex := 0

	return func(name string) string {
		if currentIndex < maxIndex {
			currentIndex++
			return fmt.Sprintf("%s_%d", name, currentIndex)
		}
		return name
	}
}

func (f *FileIO) Size() (int, error) {

	info, err := os.Stat(f.Path.Relative)
	if err != nil {
		return UnknownSize, nil
	}
	return int(info.Size()), nil

}

func (fs FileIOs) Error() error {
	var err error
	for _, f := range fs {
		err = errors.Join(err, f.Err)
	}
	return err
}

func (fs FileIOs) Close() {
	for _, f := range fs {
		f.Close()
	}
}
