package main

import (
	//"fmt"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
)

type (
	DataCaster interface {
		DataCast(ByteRange) (io.ReadCloser, error)
	}

	DLStatus uint8
	DLType   uint8

	DLOptions struct {
		URI       string
		BasePath  string
		DstDirs   []string
		PartCount int
		UI        func(*Download)
	}

	Download struct {
		Files    FileIOs
		Sources  []DataCaster
		WG       *sync.WaitGroup
		URI      string
		DataSize int
		Type     DLType
		Status   DLStatus
		UI       func(*Download)
	}
)

const (
	Pending DLStatus = iota
	Running
	Stopping
	Stopped
	Local DLType = iota
	Online
)

func (d *Download) Start() error {

	if d.UI != nil {
		d.WG.Add(1)
		go d.UI(d)
	}

	partCount := len(d.Files)

	if err := d.Files.SetByteRange(d.DataSize); err != nil {
		return err
	}

	if err := d.Files.SetInitState(); err != nil {
		return err
	}

	d.Status = Running

	for i := range partCount {

		f := d.Files[i]
		src := d.Sources[i]

		if f.State == Completed || f.State == Corrupted {
			f.ClosingSIG <- true
			continue
		}

		d.WG.Add(1)
		go Fetch(src, f, d.WG)

	}

	d.Files.WaitClosingSIG()
	d.Status = Stopping

	d.WG.Wait()
	d.Files.Close()
	d.Status = Stopped
	return nil
}

func Fetch(dc DataCaster, f *FileIO, wg *sync.WaitGroup) {
	defer wg.Done()

	if f.State == Unknown {
		err := f.Truncate(0)
		FetchErrHandle(err)
	}

	f.Seek(0, io.SeekEnd)
	r, err := dc.DataCast(f.Scope)
	FetchErrHandle(err)

	_, err = io.Copy(f, r)
	FetchErrHandle(err)

	f.ClosingSIG <- true
	r.Close()

}

func NewDownload(opt DLOptions) (*Download, error) {

	if ok, _ := isFile(opt.URI); ok {
		d, err := NewLocalDownload(opt)
		return d, err
	}

	if ok, _ := isURL(opt.URI); ok {
		d, err := NewOnlineDownload(opt)
		return d, err
	}

	return nil, errors.New("invalid file or url")
}

func NewOnlineDownload(opt DLOptions) (*Download, error) {

	uri := opt.URI
	basePath := opt.BasePath
	dstDirs := opt.DstDirs
	partCount := opt.PartCount

	hdr, cl, err := GetHeaders(uri)
	if err != nil {
		return nil, err
	}

	if cl == UnknownSize {
		partCount = 1
	}

	if basePath == "" {
		basePath = NewFileName(uri, hdr)
	}

	fios, err := BuildFileIOs(partCount, basePath, dstDirs)
	if err != nil {
		return nil, err
	}

	srcs := make([]DataCaster, partCount)
	for i := range partCount {

		ct := NewClient()
		req, err := NewReq(http.MethodGet, uri)
		if err != nil {
			return nil, err
		}
		srcs[i] = NewWebIO(ct, req)

	}

	d := &Download{
		Files:    fios,
		Sources:  srcs,
		WG:       &sync.WaitGroup{},
		URI:      uri,
		DataSize: int(cl),
		Type:     Online,
		Status:   Pending,
		UI:       opt.UI,
	}

	return d, nil
}

func NewLocalDownload(opt DLOptions) (*Download, error) {

	uri := opt.URI
	basePath := opt.BasePath
	dstDirs := opt.DstDirs
	partCount := opt.PartCount

	
	info, err := os.Stat(uri)
	if err != nil {
		return nil, err
	}

	dataSize := info.Size()

	if basePath == "" {
		basePath = NewFileName(uri, nil)
	}

	fios, err := BuildFileIOs(partCount, basePath, dstDirs)
	if err != nil {
		return nil, err
	}

	srcs := make([]DataCaster, partCount)
	for i := range partCount {

		fio, err := NewFileIO(uri, ".", os.O_RDONLY)
		if err != nil {
			return nil, err
		}
		srcs[i] = fio

	}


	d := &Download{
		Files:    fios,
		Sources:  srcs,
		WG:       &sync.WaitGroup{},
		DataSize: int(dataSize),
		Type:     Local,
		Status:   Pending,
		UI:       opt.UI,
	}

	return d, nil

}
