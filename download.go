package main

import (
	//"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"errors"
)

type (
	DataCaster interface {
		DataCast(ByteRange) io.ReadCloser
	}

	DLStatus uint8
	DLType   uint8

	Download struct {
		Files    FileIOs
		Sources  []DataCaster
		WG       *sync.WaitGroup
		URI      string
		DataSize int
		Type     DLType
		Status   DLStatus
	}
)

const (
	Starting DLStatus = iota
	Running
	Stopping
	Stopped
	Local DLType = iota
	Online
)

func (d *Download) Start() {

	filePartCount := len(d.Files)

	d.Files.setByteRange(d.DataSize)
	d.Files.setInitState()
	d.Status = Running

	d.WG.Add(1)
	go ShowProgress(d)
	for i := range filePartCount {

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
}

func Fetch(dc DataCaster, f *FileIO, wg *sync.WaitGroup) {
	defer wg.Done()

	if f.State == Unknown {
		f.Truncate(0)
	}

	f.Seek(0, io.SeekEnd)
	r := dc.DataCast(f.Scope)

	io.Copy(f, r)
	f.ClosingSIG <- true
	r.Close()
}


func buildDownload(partCount int, dstDirs []string, uri string) (*Download, error) {

	if ok, _ := isFile(uri); ok {
        return buildLocalDownload(partCount, dstDirs, uri), nil
    }

    if ok, _ := isURL(uri); ok {
        return buildOnlineDownload(partCount, dstDirs, uri), nil
    } 

    return nil, errors.New("invalid file or url")
}


func buildOnlineDownload(partCount int, dstDirs []string, uri string) *Download {

	hdr, cl := GetHeaders(uri)

	//cl = -1

	if cl == UnknownSize {
		partCount = 1
	}

	basePath := buildFileName(uri, hdr)

	fios, err := buildFileIOs(partCount, basePath, dstDirs)
	doHandle(err)
	
	srcs := make([]DataCaster, partCount)
	for i := range partCount {

		ct := buildClient()
		req := buildReq(http.MethodGet, uri)
		srcs[i] = buildNetConn(ct, req)

	}

	d := &Download{
		Files:    fios,
		Sources:  srcs,
		WG:       &sync.WaitGroup{},
		URI:      uri,
		DataSize: int(cl),
		Type:     Online,
		Status:   Starting,
	}

	return d
}

func buildLocalDownload(partCount int, dstDirs []string, srcFilePath string) *Download {


	basePath := buildFileName(srcFilePath, nil)

	fios, err := buildFileIOs(partCount, basePath, dstDirs)
	doHandle(err)
	

	srcs := make([]DataCaster, partCount)
	for i := range partCount {

		fio, _ := buildFileIO(srcFilePath, os.O_RDONLY)
		srcs[i] = fio 

	}


	srcf := srcs[0].(*FileIO)

	d := &Download{
		Files:    fios,
		Sources:  srcs,
		WG:       &sync.WaitGroup{},
		DataSize: srcf.getSize(),
		Type:     Local,
		Status:   Starting,
	}

	return d

}
