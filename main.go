package main

import (
	"fmt"
	"io"
	"net/http"
	//"os"
	"sync"
	"time"
	//"bytes"
	////"runtime"
)

type Download struct {
	*Netconn
	Files []*FileXtd
	IOch  chan io.ReadCloser
	URI   string
	WG    *sync.WaitGroup
}

func main() {

	const FileNumParts int = 3
	URI := "http://ipv4.download.thinkbroadband.com/5MB.zip"

	d := buildDownload(FileNumParts, URI)
	d.Start()

}

func (d *Download) Start() {

	FileNumParts := len(d.Files)
	headers, contentLength := d.Netconn.getRespHeaders()
	fileName := buildFileName(d.URI, &headers)
	partitionMap := buildBytePartition(int(contentLength), FileNumParts)

	for v := range FileNumParts {

		byteRange := fmt.Sprintf("bytes=%s", partitionMap[v])
		d.Request.Header.Set("Range", byteRange)
		d.Netconn = buildNetconn(d.Client, d.Request)
		fileNameWithSuffix := doAddSuffix(fileName, v)
		d.Files[v] = buildFile(fileNameWithSuffix)

		d.WG.Add(2)
		go doConn(d.Netconn, d.IOch, d.WG)
		go doWriteFile(d.Files[v], d.IOch, d.WG)

	}

	d.WG.Add(1)
	go doPrintDLProgress(d.Files, &contentLength, d.WG)

	d.WG.Wait()
	close(d.IOch)

}

func buildDownload(fnp int, uri string) *Download {

	var wg sync.WaitGroup
	ch := make(chan io.ReadCloser)

	ct := buildClient()
	req := buildReq(http.MethodGet, uri)
	nc := buildNetconn(ct, req)

	files := make([]*FileXtd, fnp)

	d := &Download{
		Netconn: nc,
		Files:   files,
		IOch:    ch,
		URI:     uri,
		WG:      &wg,
	}

	return d
}

func doPrintDLProgress(fs []*FileXtd, n *int64, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, v := range fs {
		<-v.WriteSIG
	}

	for getTotalWriter(fs) > 0 {
		for _, v := range fs {
			fmt.Println(getFileSize(v), "/", *n)
		}
		time.Sleep(50 * time.Millisecond)
	}

}

func getRawURL(a []string) string {
	return a[len(a)-1]
}

func buildBytePartition(byteCount int, parts int) []string {

	s := make([]string, parts)

	// +1 because zero is included
	partSize := (byteCount + 1) / parts
	for i, j := 0, 0; i < parts; i, j = i+1, j+partSize {

		lowerbound := j
		upperbound := lowerbound + partSize - 1
		if i == parts {
			upperbound = byteCount
		}

		s[i] = fmt.Sprintf("%v-%v", lowerbound, upperbound)

	}
	return s

}

func doAddSuffix(s string, index int) string {
	return fmt.Sprintf("%v_%v", s, index)
}
