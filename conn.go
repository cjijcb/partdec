package main

import (
	"net/http"
	"io"
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
    req.Proto = "http/1.1"
    req.ProtoMajor = 1
    req.ProtoMinor = 1
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


