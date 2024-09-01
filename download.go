package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

type Download struct {
	Files    FileIOs
	NetConns []*NetConn
	Readers  []*io.PipeReader
	Writers  []*io.PipeWriter
	URI      string
	WG       *sync.WaitGroup
	DataSize int
}

func (d *Download) Start() {

	filePartCount := len(d.Files)

	d.Files.setByteRange(d.DataSize)

	for i := range filePartCount {

		byteRange := fmt.Sprintf("bytes=%d-%d", d.Files[i].bOffS, d.Files[i].bOffE)

		d.NetConns[i].Request.Header.Set("Range", byteRange)

		d.WG.Add(2)
		go Fetch(d.NetConns[i], d.Writers[i], d.WG)
		go WriteToFile(d.Files[i], d.Readers[i], d.WG)

	}

	d.WG.Add(1)
	go doPrintDLProgress(d.Files, d.WG)

	d.WG.Wait()

}

func buildDownload(filePartCount int, uri string) *Download {

	headers, contentLength := GetHeaders(uri)

	fmt.Println(headers)

	fileName := buildFileName(uri, &headers)

	var files FileIOs = make([]*FileIO, filePartCount)

	rs := make([]*io.PipeReader, filePartCount)
	ws := make([]*io.PipeWriter, filePartCount)
	ncs := make([]*NetConn, filePartCount)

	for i := range filePartCount {

		fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
		files[i] = buildFile(fileNameWithSuffix)
		r, w := io.Pipe()
		rs[i] = r
		ws[i] = w

		ct := buildClient()
		req := buildReq(http.MethodGet, uri)
		nc := buildNetConn(ct, req)

		ncs[i] = nc

	}

	d := &Download{
		Files:    files,
		NetConns: ncs,
		Readers:  rs,
		Writers:  ws,
		URI:      uri,
		WG:       &sync.WaitGroup{},
		DataSize: int(contentLength),
	}

	return d
}
