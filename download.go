package main


import (
    "fmt"
    "io"
    "net/http"
    "sync"
)

type Download struct {
    *Netconn
    Files    FileIOs
    IOch     chan io.ReadCloser
    URI      string
    WG       *sync.WaitGroup
    DataSize int
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
        fileNameWithSuffix := fmt.Sprintf("%s_%d", fileName, i)
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
