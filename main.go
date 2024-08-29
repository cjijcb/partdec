package main

import (
	"fmt"
	"io"
	"net/http"
	//"os"
	"strings"
	"sync"
	"time"
	//"bytes"
	////"runtime"
)

type Download struct {
	*Netconn
	Files    FileIOs
	IOch     chan io.ReadCloser
	URI      string
	WG       *sync.WaitGroup
	DataSize int
}

func main() {

	const FileNumParts int = 3
	URI := "http://ipv4.download.thinkbroadband.com/5MB.zip"

	d := buildDownload(FileNumParts, URI)
	d.Start()

}

func (d *Download) Start() {

	FileNumParts := len(d.Files)

	d.Files.setByteOffsetRange(d.DataSize)

	for i := range FileNumParts {

		byteRange := fmt.Sprintf("bytes=%d-%d", d.Files[i].bOffS, d.Files[i].bOffE)

		d.Request.Header.Set("Range", byteRange)

		d.WG.Add(2)
		go doConn(d.Netconn, d.IOch, d.WG)
		go doWriteFile(d.Files[i], d.IOch, d.WG)

	}

	d.WG.Add(1)
	go doPrintDLProgress(d.Files, d.WG)

	d.WG.Wait()
	close(d.IOch)

}

func buildDownload(fnp int, uri string) *Download {

	ch := make(chan io.ReadCloser)

	ct := buildClient()
	req := buildReq(http.MethodGet, uri)
	nc := buildNetconn(ct, req)

	headers, contentLength := nc.getRespHeaders()

	fileName := buildFileName(uri, &headers)

	var files FileIOs = make([]*FileIO, fnp)

	for i := range files {
		fileNameWithSuffix := doAddSuffix(fileName, i)
		files[i] = buildFile(fileNameWithSuffix)
	}

	d := &Download{
		Netconn:  nc,
		Files:    files,
		IOch:     ch,
		URI:      uri,
		WG:       &sync.WaitGroup{},
		DataSize: int(contentLength),
	}

	return d
}

func doPrintDLProgress(fs FileIOs, wg *sync.WaitGroup) {
	defer wg.Done()

	ESC := 27
	lineCount := len(fs)
	clearLine := fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

	for _, f := range fs {
		<-f.WriteSIG
	}


	for fs.getTotalWriter() > 0 {
		for _, f := range fs {
			fmt.Printf("%d / %d\n", f.getSize(), (f.bOffE - f.bOffS))
		}
		time.Sleep(50 * time.Millisecond)
		fmt.Printf(strings.Repeat(clearLine, lineCount))

	}

}

func getRawURL(a []string) string {
	return a[len(a)-1]
}

func doAddSuffix(s string, index int) string {
	return fmt.Sprintf("%v_%v", s, index)
}
