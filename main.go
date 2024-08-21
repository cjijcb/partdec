package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"
	//"bytes"
	//"log"
)

var wg sync.WaitGroup

type Netconn struct {
	Client  *http.Client
	Request *http.Request
}

type File struct {
	*os.File
	ActiveWriter *int64
	WriteSIG     chan struct{}
}

func main() {

	chR := make(chan io.ReadCloser)

	rawURL := "http://examplefile.com/file-download/27"

	nc := &Netconn{
		Client:  buildClient(),
		Request: buildReq(http.MethodGet, rawURL),
	}

	headers, contentLength := getHeaders(nc)
	fileName := buildFileName(rawURL, &headers)

	file := &File{
		File:         buildFile(fileName),
		ActiveWriter: new(int64),
		WriteSIG:     make(chan struct{}),
	}

	fmt.Println("c:", contentLength)
	//fmt.Println("hd:", hd )
	fmt.Println(fileName)

	wg.Add(3)

	go doConn(nc, chR)
	go doWriteFile(file, chR)
	go doPrintDLProgress(file, &contentLength)

	wg.Wait()
	close(chR)
	os.Exit(0)
}

func buildFileName(rawURL string, hdr *http.Header) string {
	_, params, _ := mime.ParseMediaType(hdr.Get("Content-Disposition"))
	fileName := params["filename"]

	if fileName != "" {
		return fileName
	}

	url, _ := url.Parse(rawURL)

	fileName = path.Base(url.Path)
	return fileName

}

func buildFile(p string) *os.File {

	file, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	//defer file.Close()
	return file
}

func buildReq(method string, rawURL string) *http.Request {
	req, _ := http.NewRequest(method, rawURL, nil)
	req.Proto = "http/2"
	req.ProtoMajor = 2
	req.ProtoMinor = 0
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "fssn/1.0.0")
	return req
}

func buildClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns: 0,
	}
	ct := &http.Client{Transport: tr}
	return ct
}

func doConn(nc *Netconn, chR chan io.ReadCloser) {
	defer wg.Done()
	resp, _ := nc.Client.Do(nc.Request)
	fmt.Println(nc.Request)
	//defer resp.Body.Close()
	chR <- resp.Body
}

func getHeaders(nc *Netconn) (http.Header, int64) {
	newReq := *nc.Request
	newReq.Method = http.MethodHead
	resp, _ := nc.Client.Do(&newReq)
	return resp.Header, resp.ContentLength
}

func doWriteFile(f *File, chR chan io.ReadCloser) {
	defer wg.Done()
	*f.ActiveWriter += 1
	f.WriteSIG <- struct{}{}
	io.Copy(f, <-chR)
	*f.ActiveWriter -= 1
	f.Sync()
}

func getFileSize(f *File) int64 {
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(err)
	}
	return fi.Size()
}

func doPrintDLProgress(f *File, n *int64) {
	defer wg.Done()
	<-f.WriteSIG
	for *f.ActiveWriter > 0 {
		fmt.Println(getFileSize(f), "/", *n)
		time.Sleep(50 * time.Millisecond)
	}
}
