package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
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



func buildDownload(filePartCount int, uri string) *Download {

	hdr, cl := GetHeaders(uri)

	cl = -1

	if cl == UnknownSize {
		filePartCount = 1
	}

	files := make([]*FileIO, filePartCount)
	srcs := make([]DataCaster, filePartCount)

	fileName := buildFileName(uri, hdr)

	for i := range filePartCount {

		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
		files[i] = buildFile(fileNameWithSuffix, os.O_WRONLY)

		ct := buildClient()
		req := buildReq(http.MethodGet, uri)
		srcs[i] = buildNetConn(ct, req)

	}

	d := &Download{
		Files:    files,
		Sources:  srcs,
		WG:       &sync.WaitGroup{},
		URI:      uri,
		DataSize: int(cl),
		Type:     Online,
		Status:   Starting,
	}

	return d
}


func buildLocalDownload(filePartCount int, srcFilePath string) *Download {

	files := make([]*FileIO, filePartCount)
	srcs := make([]DataCaster, filePartCount)
	
	fileName := buildFileName(srcFilePath, nil)

	for i := range filePartCount {

		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
		files[i] = buildFile(fileNameWithSuffix, os.O_WRONLY)

		srcs[i] = buildFile(srcFilePath, os.O_RDONLY)
		
	}

	srcf := srcs[0].(*FileIO)

	d := &Download{
		Files:       files,
		Sources: 	 srcs,
		WG:          &sync.WaitGroup{},
		DataSize:    srcf.getSize(),
		Type:        Local,
		Status:      Starting,
	}

	return d

}
