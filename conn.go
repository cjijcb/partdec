package main

import (
	"io"
	"net/http"
	"sync"
)

type Netconn struct {
	Client  *http.Client
	Request *http.Request
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
	req.Proto = "http/2.0"
	req.ProtoMajor = 2
	req.ProtoMinor = 0
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "curl/8.9.1")
	return req
}

func buildClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns: 0,
	}
	ct := &http.Client{Transport: tr}
	return ct
}

func doConn(nc *Netconn, chR chan io.ReadCloser, wg *sync.WaitGroup) {
	defer wg.Done()
	resp, err := nc.Client.Do(nc.Request)
	doHandle(&err)
	//defer resp.Body.Close()
	chR <- resp.Body
}

func (nc *Netconn) getRespHeaders() (http.Header, int64) {
	newReq := *nc.Request
	newReq.Method = http.MethodHead
	resp, _ := nc.Client.Do(&newReq)
	return resp.Header, resp.ContentLength
}
