/*
Copyright 2024 Carlo Jay I. Jacaba

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package partdec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type (
	FileState uint8

	FileResets map[FileState]bool

	FilePath struct {
		Base, DstDir, Relative string
	}

	ByteRange struct {
		Start, End, Offset int64
		isFullRange        bool
	}

	FileIO struct {
		*os.File
		Scope  ByteRange
		State  FileState
		Path   FilePath
		Oflag  int
		Perm   os.FileMode
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

	FilePerm os.FileMode = 0644

	UnknownSize   = -1
	PathSeparator = string(os.PathSeparator)
)

func BuildFileIOs(partCount int, base string, dirs []string) (FileIOs, error) {

	if dirs == nil {
		dirs = []string{""}
	}

	fios := make(FileIOs, partCount)
	dirCount := len(dirs)
	perDir := partCount / dirCount
	remains := partCount % dirCount
	addIndex := FileNameIndexer(partCount)

	var x int
	for _, d := range dirs {

		e := 0
		if remains > 0 {
			e = 1
			remains--
		}

		for range perDir + e {

			fio, err := NewFileIO(addIndex(base), d, os.O_CREATE|os.O_WRONLY)
			if err != nil {
				return nil, err
			}
			fio.Close()
			fios[x] = fio
			x++
		}

	}

	return fios, nil

}

func NewFileIO(base, dir string, oflag int) (*FileIO, error) {

	var relv string

	switch {
	case dir == "":
		relv = filepath.Clean(base)
	default:
		relv = filepath.Clean(dir + PathSeparator + base)
	}

	f, err := os.OpenFile(relv, oflag, FilePerm)

	if err != nil {
		return nil, err
	}

	return &FileIO{
		File: f,
		Path: FilePath{
			Base:     base,
			DstDir:   dir,
			Relative: relv,
		},
		Oflag:  oflag,
		Perm:   FilePerm,
		isOpen: true,
	}, nil

}

func (fio *FileIO) DataCast(br ByteRange) (io.ReadCloser, error) {

	rangeStart := br.Start + br.Offset
	rangeEnd := br.End

	if rangeEnd < 0 {
		rangeEnd = 0
	}

	if rangeStart > rangeEnd || rangeStart < 0 {
		rangeStart = rangeEnd
	}

	r := io.NewSectionReader(fio, rangeStart, rangeEnd-rangeStart+1)

	return io.NopCloser(r), nil

}

func NewFileDataCaster(path string) (DataCaster, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &FileIO{
		File: f,
		Path: FilePath{
			Base: path,
		},
		isOpen: true,
	}, nil
}

func (fios FileIOs) RenewByState(fr FileResets) error {

	for _, fio := range fios {

		if fio.State == Unknown {
			if err := fio.Open(); err != nil {
				return err
			}

			if err := fio.Truncate(0); err != nil {
				return err
			}
			fio.Close()
			continue
		}

		if fr != nil && fr[fio.State] == true {
			if err := fio.Open(); err != nil {
				return err
			}

			if err := fio.Truncate(0); err != nil {
				return err
			}
			fio.Close()
			fio.Scope.Offset = 0
			fio.State = New
		}

	}

	return nil

}

func (fios FileIOs) SetInitialState() error {

	for _, fio := range fios {
		size, err := fio.Size()
		if err != nil {
			return err
		}

		rs := fio.Scope.Start
		re := fio.Scope.End

		switch {
		case rs == UnknownSize || re == UnknownSize:
			fio.State = Unknown
		case rs > re:
			fio.State = Broken
		case size > re-rs+1:
			fio.State = Broken
		case size == re-rs+1:
			fio.State = Completed
		case size > 0:
			fio.State = Resume
		default:
			fio.State = New
		}
	}
	return nil

}

func (fios FileIOs) SetByteRange(dataSize int64, partSize int64) error {

	if len(fios) == 1 {
		fios[0].Scope.isFullRange = true
	}

	if dataSize < 0 {
		for _, fio := range fios {
			fio.Scope.Start = 1 // end - start + 1 = -1
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

func (fios FileIOs) setByteRangeByPartCount(dataSize int64) error {

	var rangeStart, rangeEnd, offset, extraByte int64

	partCount := len(fios)
	basePartSize := dataSize / int64(partCount)
	remainder := dataSize % int64(partCount)

	var i int
	for i, offset = 0, 0; i < partCount; i, offset = i+1, offset+basePartSize {

		extraByte = 0
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

func (fios FileIOs) setByteRangeByPartSize(dataSize, partSize int64) error {

	var rangeStart, rangeEnd, offset int64

	partCount := int(dataSize / partSize)
	remainder := dataSize % partSize

	if remainder > 0 {
		partCount++
	}

	var i int
	for i, offset = 0, 0; i < partCount; i, offset = i+1, offset+partSize {

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

func (fio *FileIO) Open() (err error) {

	fio.File, err = os.OpenFile(fio.Path.Relative, fio.Oflag, fio.Perm)
	if err != nil {
		return err
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
	countDigits := func(n int) int {
		count := 0
		switch {
		case n == 0:
			return 1
		case n < 0:
			n = -n
		}

		for n > 0 {
			n /= 10
			count++
		}
		return count
	}
	pad := countDigits(maxIndex)
	currentIndex := 0

	return func(name string) string {
		if currentIndex < maxIndex {
			currentIndex++
			return fmt.Sprintf("%s_%0*d", name, pad, currentIndex)
		}
		return name
	}

}

func (s FileState) String() string {

	switch s {
	case New:
		return "new"
	case Resume:
		return "resume"
	case Completed:
		return "completed"
	case Broken:
		return "broken"
	default:
		return "unknown"
	}

}

func (fio *FileIO) Size() (int64, error) {

	info, err := os.Stat(fio.Path.Relative)
	if err != nil {
		return UnknownSize, nil
	}
	return info.Size(), nil

}

func (fios FileIOs) TotalSize() int64 {

	var totalSize, size int64

	for _, fio := range fios {
		size, _ = fio.Size()
		totalSize += size
	}
	return totalSize

}

func (fio *FileIO) IsOpen() bool {

	mtx.Lock()
	defer mtx.Unlock()
	return fio.isOpen

}

func (fio *FileIO) Close() error {

	mtx.Lock()
	defer mtx.Unlock()

	if !fio.isOpen {
		return nil
	}

	fio.isOpen = false
	if fio.File != nil {
		if err := fio.File.Close(); err != nil {
			return err
		}
	}
	return nil

}

func (fio *FileIO) PullState() FileState {

	mtx.Lock()
	defer mtx.Unlock()
	return fio.State

}

func (fio *FileIO) PushState(fs FileState) {

	mtx.Lock()
	defer mtx.Unlock()
	fio.State = fs

}
