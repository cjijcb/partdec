package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type (
	DataStream struct {
		*io.PipeReader
		*io.PipeWriter
	}

	Download struct {
		Files       FileIOs
		NetConns    []*NetConn
		DataStreams []*DataStream
		URI         string
		WG          *sync.WaitGroup
		DataSize    int
	}
)

func (d *Download) Start() {

	filePartCount := len(d.Files)

	d.Files.setByteRange(d.DataSize)
	d.Files.setInitState()

	for i := range filePartCount {

		f := d.Files[i]
		nc := d.NetConns[i]
		ds := d.DataStreams[i]

		nc.Request.Header.Set("Range", buildRangeHeader(f))

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

	r, w := io.Pipe()
	ds := &DataStream{
		PipeReader: r,
		PipeWriter: w,
	}

	return ds
}
