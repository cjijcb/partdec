package main

import (
    "net/http"
    "fmt"
	"io"
	"os"
	"sync"
	"time"
	//"bytes"
	//"log"
)



var wg sync.WaitGroup

type fileDL struct {
	*os.File
	activeWriter *int64
	writeSIG	chan struct{}
}

func main() {


	chR := make(chan io.ReadCloser)


	f := &fileDL{
		File: buildFile("file.dat"),
		activeWriter: new(int64),
		writeSIG: make(chan struct{}),
	}

	ct := buildClient()	
	req := buildReq("HEAD")

	clh := getContentLengthHeader(ct, req)

	req = buildReq("GET")

	wg.Add(3)

	go doConn(ct, req, chR)
	go doWriteFile(f,chR)
	go doPrintDLProgress(f, &clh)
	
	wg.Wait()
	close(chR)
	os.Exit(0)
}


func buildFile(p string) *os.File {

    file, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
    if err != nil {
        panic(err)
    }
    //defer file.Close()
	return file
}


func buildReq(method string) *http.Request {
	req, _ := http.NewRequest(method, "http://examplefile.com/file-download/25", nil)
    req.Proto = "http/2"
    req.ProtoMajor = 2
    req.ProtoMinor = 0
	return req
}

func buildClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:	0,
	}	
	ct := &http.Client{Transport: tr}
	return ct	
}

func doConn(ct *http.Client, req *http.Request, chR chan io.ReadCloser) {
	defer wg.Done()
	resp, _ := ct.Do(req)
	fmt.Println("CL:", resp.ContentLength)
	//defer resp.Body.Close()
	chR <- resp.Body
}

func getContentLengthHeader(ct *http.Client, req *http.Request) int64 {
	resp, _ := ct.Do(req)
	return resp.ContentLength
}

func doWriteFile(f *fileDL, chR chan io.ReadCloser) {
	defer wg.Done()
	*f.activeWriter += 1
	f.writeSIG <- struct{}{}
	//f.ReadFrom(<-chR)
	io.Copy(f, <-chR)
	*f.activeWriter -= 1
	f.Sync()
}

func getFileSize(f *fileDL) int64 {
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(err)
	}
	return fi.Size()
}


func doPrintDLProgress(f *fileDL, n *int64) {
	defer wg.Done()
	<-f.writeSIG
	for *f.activeWriter > 0 {
		fmt.Println(getFileSize(f), "/", *n)
		time.Sleep(50 * time.Millisecond)
	}
}
