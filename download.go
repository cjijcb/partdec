package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type (
	DLStatus uint8

	DataStream struct {
		R      *io.PipeReader
		W      *io.PipeWriter
		RWDone chan bool
	}

	Download struct {
		Files       FileIOs
		NetConns    []*NetConn
		DataStreams []*DataStream
		URI         string
		WG          *sync.WaitGroup
		Status      DLStatus
		DataSize    int
	}
)

const (
	Starting DLStatus = iota
	Running
	Stopping
	Stopped
	SIG = true
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
		nc := d.NetConns[i]
		ds := d.DataStreams[i]

		if f.State == Completed || f.State == Corrupted {
			close(ds.RWDone)
			continue
		}

		nc.Request.Header.Set("Range", buildRangeHeader(f))

		d.WG.Add(2)
		go Fetch(nc, ds, d.WG)
		go WriteToFile(f, ds, d.WG)

	}

	d.WG.Add(1)
	go func() {
		defer d.WG.Done()
		for _, ds := range d.DataStreams {
			<-ds.RWDone
		}
		d.Status = Stopping
	}()

	d.WG.Wait()
	d.Files.Close()
	d.Status = Stopped
}

func buildDownload(filePartCount int, uri string) *Download {

	files := make([]*FileIO, filePartCount)
	dss := make([]*DataStream, filePartCount)
	ncs := make([]*NetConn, filePartCount)

	headers, contentLength := GetHeaders(uri)
	fileName := buildFileName(uri, &headers)

	for i := range filePartCount {

		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
		files[i] = buildFile(fileNameWithSuffix)

		dss[i] = buildDataStream()

		ct := buildClient()
		req := buildReq(http.MethodGet, uri)
		ncs[i] = buildNetConn(ct, req)

	}

	d := &Download{
		Files:       files,
		NetConns:    ncs,
		DataStreams: dss,
		URI:         uri,
		WG:          &sync.WaitGroup{},
		DataSize:    int(contentLength),
		Status:      Starting,
	}

	return d
}

func buildRangeHeader(f *FileIO) string {
	rangeStart := f.StartByte + f.getSize()
	rangeEnd := f.EndByte
	if rangeStart > rangeEnd {
		rangeStart = rangeEnd
	}
	rh := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd)
	fmt.Println(rh)
	return rh
}

func buildDataStream() *DataStream {

	rwc := make(chan bool)

	r, w := io.Pipe()
	ds := &DataStream{
		R:      r,
		W:      w,
		RWDone: rwc,
	}

	return ds
}