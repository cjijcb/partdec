package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
	//"bytes"
	"runtime"
)

var wg sync.WaitGroup

type Netconn struct {
	Client  *http.Client
	Request *http.Request
}


func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	chR := make(chan io.ReadCloser)

	//fmt.Println(getRawURL(os.Args))

	rawURL := "http://ipv4.download.thinkbroadband.com/5MB.zip"

	ct := buildClient()
	req := buildReq(http.MethodGet, rawURL)

	nc := buildNetconn(ct, req)
	headers, contentLength := getHeaders(nc)

	fmt.Println(buildBytePartition(contentLength, 3))

	fileName := buildFileName(rawURL, &headers)

	//file := buildFile(fileName)

	files := make([]*FileXtd,3)

	fmt.Println(doAddSuffix(fileName, 1))

	//fmt.Println("c:", contentLength)
	//fmt.Println("hd:", hd )
	//fmt.Println(fileName)

	partitionMap := buildBytePartition(contentLength, 3)

	for v := range 3 {

		byteRange := fmt.Sprintf("bytes=%s", partitionMap[v])

		req.Header.Set("Range", byteRange)
		nc := buildNetconn(ct, req)
		fileNameWithSuffix := doAddSuffix(fileName, v)
		files[v] = buildFile(fileNameWithSuffix)

		wg.Add(2)
		go doConn(nc, chR)
		go doWriteFile(files[v], chR)

	}

	wg.Add(1)
	go doPrintDLProgress(files, &contentLength)

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


func doPrintDLProgress(fs []*FileXtd, n *int64) {
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


func buildBytePartition(byteCount int64, parts int) []string {

	s := make([]string, parts)

	// +1 because zero is included
	partSize := int(byteCount+1) / parts
	for i, j := 0, 0; i < parts; i, j = i+1, j+partSize {

		lowerbound := j
		upperbound := lowerbound + partSize - 1
		if i == parts {
			upperbound = int(byteCount)
		}

		s[i] = fmt.Sprintf("%v-%v", lowerbound, upperbound)

	}
	return s

}

func doAddSuffix(s string, index int) string {
	return fmt.Sprintf("%v_%v", s, index)
}
