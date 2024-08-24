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
	"log"
)

var wg sync.WaitGroup

type Netconn struct {
	Client  *http.Client
	Request *http.Request
}

type FileXtd struct {
	*os.File
	ActiveWriter *int
	WriteSIG     chan struct{}
}

func main() {

	chR := make(chan io.ReadCloser)

	//fmt.Println(getRawURL(os.Args))

	rawURL := "http://ipv4.download.thinkbroadband.com/5MB.zip"

	ct := buildClient()
	req := buildReq(http.MethodGet, rawURL)

	nc := buildNetconn(ct, req)
	headers, contentLength := getHeaders(nc)

	fmt.Println(buildBytePartition(contentLength, 3))

	fileName := buildFileName(rawURL, &headers)
	file := buildFile(fileName)

	fmt.Println(doAddSuffix(fileName, 1))
	

	//fmt.Println("c:", contentLength)
	//fmt.Println("hd:", hd )
	//fmt.Println(fileName)

	wg.Add(3)

	go doConn(nc, chR)
	go doWriteFile(file, chR)
	go doPrintDLProgress(file, &contentLength)

	wg.Wait()
	close(chR)
	os.Exit(0)

}

func buildNetconn(ct *http.Client, req *http.Request) *Netconn {
	nc := &Netconn{
		Client:  ct,
		Request: req,
	}

	return nc
}

func buildFileName(rawURL string, hdr *http.Header) string {
	_, params, err := mime.ParseMediaType(hdr.Get("Content-Disposition"))
	doHandle(&err)
	fileName := params["filename"]

	if fileName != "" {
		return fileName
	}

	url, err := url.Parse(rawURL)
	doHandle(&err)

	fileName = path.Base(url.Path)
	return fileName

}

func buildFile(name string) *FileXtd {

	f, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	
	doHandle(&err)

	file := &FileXtd{
		File:         f,
		ActiveWriter: new(int),
		WriteSIG:     make(chan struct{}),
	}
	//defer file.Close()
	return file
}

func buildReq(method string, rawURL string) *http.Request {
	req, err := http.NewRequest(method, rawURL, nil)
	doHandle(&err)
	req.Proto = "http/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Header.Set("Accept", "*/*")
	//req.Header.Set("Range", "bytes=1024-6000")
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
	resp, err := nc.Client.Do(nc.Request)
	doHandle(&err)
	//fmt.Println(nc.Request)
	//defer resp.Body.Close()
	chR <- resp.Body
}

func getHeaders(nc *Netconn) (http.Header, int64) {
	newReq := *nc.Request
	newReq.Method = http.MethodHead
	resp, _ := nc.Client.Do(&newReq)
	return resp.Header, resp.ContentLength
}

func doWriteFile(f *FileXtd, chR chan io.ReadCloser) {
	defer wg.Done()
	//*f.ActiveWriter += 1
	f.addWriter(1)
	f.WriteSIG <- struct{}{}
	io.Copy(f, <-chR)
	f.addWriter(-1)
	//*f.ActiveWriter -= 1
	f.Sync()
}

func (f *FileXtd) addWriter(n int) {
	*f.ActiveWriter += n
}


func getFileSize(f *FileXtd) int64 {
	fi, err := f.Stat()
	doHandle(&err)
	return fi.Size()
}

func doPrintDLProgress(f *FileXtd, n *int64) {
	defer wg.Done()
	<-f.WriteSIG
	for *f.ActiveWriter > 0 {
		fmt.Println(getFileSize(f), "/", *n)
		time.Sleep(50 * time.Millisecond)
	}
}

func getRawURL(a []string) string {
	return a[len(a)-1]
}

func doHandle(err *error) {
	if *err != nil {
		log.Println(*err)
	}
}


func buildBytePartition(byteCount int64, parts int) map[int]string {

	m := make(map[int]string)

    // +1 because zero is included
	partSize := int(byteCount + 1) / parts
    for i, j := 1, 0; i <= parts; i, j = i+1, j+partSize {

        lowerbound := j
        upperbound := lowerbound + partSize - 1
        if i == parts {
            upperbound = int(byteCount)
        }

		mk := i
        mv := fmt.Sprintf("%v-%v", lowerbound, upperbound)
		m[mk] = mv

    }
	return m

}

func doAddSuffix (s string, index int) string {
	return fmt.Sprintf("%v_%v", s, index)
}
