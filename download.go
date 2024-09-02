package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type DataStream struct {
	*io.PipeReader
	*io.PipeWriter
}

type Download struct {
	Files       FileIOs
	NetConns    []*NetConn
	DataStreams []*DataStream
	URI         string
	WG          *sync.WaitGroup
	DataSize    int
}

func (d *Download) Start() {

	filePartCount := len(d.Files)

	d.Files.setByteRange(d.DataSize)

	for i := range filePartCount {

		byteRange := fmt.Sprintf("bytes=%d-%d", d.Files[i].bOffS, d.Files[i].bOffE)

		d.NetConns[i].Request.Header.Set("Range", byteRange)

		d.WG.Add(2)
		go Fetch(d.NetConns[i], d.DataStreams[i], d.WG)
		go WriteToFile(d.Files[i], d.DataStreams[i], d.WG)

	}

	d.WG.Add(1)
	go doPrintDLProgress(d.Files, d.WG)

	d.WG.Wait()

}

func buildDownload(filePartCount int, uri string) *Download {

	files := make([]*FileIO, filePartCount)
	ds := make([]*DataStream, filePartCount)
	ncs := make([]*NetConn, filePartCount)

	headers, contentLength := GetHeaders(uri)
	fileName := buildFileName(uri, &headers)

	for i := range filePartCount {

		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
		files[i] = buildFile(fileNameWithSuffix)

		r, w := io.Pipe()
		ds[i] = &DataStream{
			PipeReader: r,
			PipeWriter: w,
		}

		ct := buildClient()
		req := buildReq(http.MethodGet, uri)
		ncs[i] = buildNetConn(ct, req)

	}

	d := &Download{
		Files:       files,
		NetConns:    ncs,
		DataStreams: ds,
		URI:         uri,
		WG:          &sync.WaitGroup{},
		DataSize:    int(contentLength),
	}

	return d
}
