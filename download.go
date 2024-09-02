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

		f := d.Files[i]
		nc := d.NetConns[i]
		ds := d.DataStreams[i]

		rangeStart := f.bOffS + int(f.getSize())
		rangeEnd := f.bOffE 

		byteRange := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd) 	

		nc.Request.Header.Set("Range", byteRange)

		d.WG.Add(2)
		go Fetch(nc, ds, d.WG)
		go WriteToFile(f, ds, d.WG)

	}

	d.WG.Add(1)
	go doPrintDLProgress(d.Files, d.WG)

	d.WG.Wait()
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
	}

	return d
}

func buildDataStream() *DataStream {

	r, w := io.Pipe()
	ds := &DataStream{
		PipeReader: r,
		PipeWriter: w,
	}

	return ds
}


