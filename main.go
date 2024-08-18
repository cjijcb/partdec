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

	req := buildReq()
	ct := buildClient()	

	f := &fileDL{
		File: buildFile("file.dat"),
		activeWriter: new(int64),
		writeSIG: make(chan struct{}),
	}


	
	wg.Add(3)

	go doConn(ct, req, chR)
	go doWriteFile(f,chR)
	go doPrintDLProgress(f)
	
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


func buildReq() *http.Request {
	req, _ := http.NewRequest("GET", "http://examplefile.com/file-download/25", nil)
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
	//defer resp.Body.Close()
	chR <- resp.Body
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


func doPrintDLProgress(f *fileDL) {
	defer wg.Done()
	<-f.writeSIG
	for *f.activeWriter > 0 {
		fmt.Println(getFileSize(f))
		time.Sleep(50 * time.Millisecond)
	}
}
