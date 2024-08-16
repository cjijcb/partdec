package main

import (
    "net/http"
    //"fmt"
	"io"
	"os"
	"sync"
	//"bytes"
	//"log"
)



var wg sync.WaitGroup


func main() {


	chR := make(chan io.ReadCloser)

	req := buildReq()
	ct := buildClient()	
	

	wg.Add(1)

	go doConn(ct, req, chR)
	go doWriteFile(chR)
	
	wg.Wait()
	close(chR)
	os.Exit(0)
}



func buildReq() *http.Request {
	req, _ := http.NewRequest("GET", "https://example.com", nil)
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
	resp, _ := ct.Do(req)
	defer resp.Body.Close()
	chR <- resp.Body
	wg.Wait()
}


func doWriteFile(chR chan io.ReadCloser) {
	defer wg.Done()
	io.CopyBuffer(os.Stdout, <-chR, make([]byte, 1024))
}
