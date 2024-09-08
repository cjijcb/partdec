package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type (
	DataCaster interface {
		SetScope(ByteRange)
		DataCast() io.ReadCloser
	}

	DLStatus uint8
	DLType   uint8

	Download struct {
		Files       FileIOs
		Sources		[]DataCaster
		WG          *sync.WaitGroup
		URI         string
		DataSize    int
		Type        DLType
		Status      DLStatus
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

func buildDownload(filePartCount int, uri string) *Download {

	hdrs, cl := GetHeaders(uri)

	if cl == UnknownSize {
		filePartCount = 1
	}

	files := make([]*FileIO, filePartCount)
	srcs := make([]DataCaster, filePartCount)

	fileName := buildFileName(uri, &hdrs)

	for i := range filePartCount {

		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
		files[i] = buildFile(fileNameWithSuffix)

		ct := buildClient()
		req := buildReq(http.MethodGet, uri)
		srcs[i] = buildNetConn(ct, req)

	}

	d := &Download{
		Files:       files,
		Sources:     srcs,
		WG:          &sync.WaitGroup{},
		URI:         uri,
		DataSize:    int(cl),
		Type:        Online,
		Status:      Starting,
	}

	return d
}

func Fetch(dc DataCaster, f *FileIO, wg *sync.WaitGroup) {
	defer wg.Done()

	dc.SetScope(f.Scope)

	r := dc.DataCast()
	
	f.Seek(0, io.SeekEnd)
	io.Copy(f, r)
	f.ClosingSIG <- true
	r.Close()
}



//func buildLocalDownload(filePartCount int, srcFilePath string) *Download {
//
//	files := make([]*FileIO, filePartCount)
//	dss := make([]*DataStream, filePartCount)
//
//	fileName := buildFileName(srcFilePath, nil)
//	srcFile := buildFile(srcFilePath)
//
//	for i := range filePartCount {
//
//		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
//		files[i] = buildFile(fileNameWithSuffix)
//
//		dss[i] = buildDataStream()
//
//	}
//
//	d := &Download{
//		Files:       files,
//		DataStreams: dss,
//		WG:          &sync.WaitGroup{},
//		DataSize:    srcFile.getSize(),
//		Type:        Local,
//		Status:      Starting,
//	}
//
//	return d
//
//}
