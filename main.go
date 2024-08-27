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
