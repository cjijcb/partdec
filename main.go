package main

import (
    "net/http"
	"net/url"
    "fmt"
	"io"
	"os"
	"sync"
	"time"
	"mime"
	"path"
	//"bytes"
	//"log"
)



var wg sync.WaitGroup



type netconn struct {
	Client	*http.Client
	Request	*http.Request
}

type fileDL struct {
	*os.File
	ActiveWriter *int64
	WriteSIG	chan struct{}
}

func main() {


	chR := make(chan io.ReadCloser)

	rawURL := "http://examplefile.com/file-download/27"


	f := &fileDL{
		File: buildFile("file.dat"),
		ActiveWriter: new(int64),
		WriteSIG: make(chan struct{}),
	}

	nc := &netconn{
		Client: buildClient(),
		Request: buildReq("GET", rawURL),
	}


	hd, cl := getHeaders(nc)
	fln := buildFileName(rawURL, &hd)

	fmt.Println("c:",cl)
	//fmt.Println("hd:", hd )
	fmt.Println(fln)



	wg.Add(3)

	go doConn(nc, chR)
	go doWriteFile(f,chR)
	go doPrintDLProgress(f, &cl)
	
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
		MaxIdleConns:	0,
	}	
	ct := &http.Client{Transport: tr}
	return ct	
}

func doConn(nc *netconn, chR chan io.ReadCloser) {
	defer wg.Done() 
	resp, _ := nc.Client.Do(nc.Request)
	fmt.Println(nc.Request)
	//defer resp.Body.Close()
	chR <- resp.Body
}

func getHeaders(nc *netconn) (http.Header, int64) {
	newReq := *nc.Request
	newReq.Method = "HEAD"
	resp, _ := nc.Client.Do(&newReq)
	return resp.Header, resp.ContentLength
}

func doWriteFile(f *fileDL, chR chan io.ReadCloser) {
	defer wg.Done()
	*f.ActiveWriter += 1
	f.WriteSIG <- struct{}{}
	io.Copy(f, <-chR)
	*f.ActiveWriter -= 1
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
	<-f.WriteSIG
	for *f.ActiveWriter > 0 {
		fmt.Println(getFileSize(f), "/", *n)
		time.Sleep(50 * time.Millisecond)
	}
}
