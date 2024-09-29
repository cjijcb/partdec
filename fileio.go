package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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
		Scope  ByteRange
		State  FileState
		Path   FilePath
		isOpen bool
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
		isOpen: true,
	}

	return fio, err
}

func (fio *FileIO) DataCast(br ByteRange) (io.Reader, error) {

	rangeStart := br.Start + br.Offset
	rangeEnd := br.End

	if rangeStart > rangeEnd {
		rangeStart = rangeEnd
	}

	rangeEnd = rangeEnd - rangeStart + 1

	r := io.NewSectionReader(fio, int64(rangeStart), int64(rangeEnd))

	return r, nil

}

func (fio *FileIO) NewDataCaster(path string) (DataCaster, error) {

	newfio, err := NewFileIO(path, CurrentDir, os.O_RDONLY)
	if err != nil {
		return nil, err
	}

	fio = newfio
	return fio, nil

}

func (fios FileIOs) RenewByState(sm map[FileState]bool) error {

	for _, fio := range fios {
		if sm[fio.State] == false {
			continue
		}

		if err := fio.Truncate(0); err != nil {
			return err
		}

		fio.State = New

	}

	return nil

}

func (fios FileIOs) SetInitState() error {

	for _, fio := range fios {
		size, err := fio.Size()
		if err != nil {
			return err
		}
		sb := fio.Scope.Start
		eb := fio.Scope.End

		if sb == UnknownSize || eb == UnknownSize {
			fio.State = Unknown
		} else if sb > eb {
			fio.State = Broken
		} else if size > eb-sb+1 {
			fio.State = Broken
		} else if size == eb-sb+1 {
			fio.State = Completed
		} else if size > 0 {
			fio.State = Resume
		} else if size == 0 {
			fio.State = New
		}

	}

	return nil

}

func (fios FileIOs) SetByteRange(dataSize int, partSize int) error {

	if dataSize == UnknownSize {
		for _, fio := range fios {
			fio.Scope.Start = UnknownSize
			fio.Scope.End = UnknownSize
		}
		return nil
	}

	if partSize > 0 {
		if err := fios.setByteRangeByPartSize(dataSize, partSize); err != nil {
			return err
		}
	} else {
		if err := fios.setByteRangeByPartCount(dataSize); err != nil {
			return err
		}
	}

	return nil
}

func (fios FileIOs) setByteRangeByPartCount(dataSize int) error {

	var rangeStart, rangeEnd int

	partCount := len(fios)
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

		fio := fios[i]

		fio.Scope.Start = rangeStart
		fio.Scope.End = rangeEnd
		size, err := fio.Size()

		if err != nil {
			return err
		}

		fio.Scope.Offset = size

	}

	return nil
}

func (fios FileIOs) setByteRangeByPartSize(dataSize int, partSize int) error {

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

		fio := fios[i]

		fio.Scope.Start = rangeStart
		fio.Scope.End = rangeEnd
		size, err := fio.Size()

		if err != nil {
			return err
		}

		fio.Scope.Offset = size
	}

	return nil
}

func newFileNameFromPath(path string) string {

	return filepath.Base(path)

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

func (fio *FileIO) Size() (int, error) {

	info, err := os.Stat(fio.Path.Relative)
	if err != nil {
		return UnknownSize, nil
	}
	return int(info.Size()), nil

}

func (fio *FileIO) IsOpen() bool {
	return fio.isOpen
}

func (fio *FileIO) Close() error {

	if err := fio.File.Close(); err != nil {
		return err
	}

	fio.isOpen = false
	return nil

}

func (fios FileIOs) Close() error {

	var err error
	for _, fio := range fios {
		if fio != nil && fio.isOpen {
			err = errors.Join(err, fio.Close())
		}
	}
	return err

}
