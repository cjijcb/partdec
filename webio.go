package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type (
	WebIO struct {
		Client  *http.Client
		Request *http.Request
	}
)

func NewWebIO(ct *http.Client, req *http.Request) *WebIO {
	wbio := &WebIO{
		Client:  ct,
		Request: req,
	}

	return wbio
}

func NewReq(method string, rawURL string) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Proto = "http/1.1"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "fssn/1.0.0")
	return req, nil
}

func NewClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns: 0,
	}
	ct := &http.Client{Transport: tr}
	return ct
}

func (wbio *WebIO) DataCast(br ByteRange) (io.ReadCloser, error) {

	wbio.Request.Header.Set("Range", BuildRangeHeader(br))

	resp, err := wbio.Client.Do(wbio.Request)
	if err != nil {
		return nil, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return nil, errors.New(resp.Status)
	}

	return resp.Body, nil
}

func BuildRangeHeader(br ByteRange) string {

	if br.Start == UnknownSize || br.End == UnknownSize {
		return "none"
	}

	rangeStart := br.Start + br.Offset
	rangeEnd := br.End

	if rangeStart > rangeEnd {
		rangeStart = rangeEnd
	}

	return fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd)

}

func GetHeaders(rawURL string) (http.Header, int64, error) {

	ct := &http.Client{}
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)

	if err != nil {
		return nil, UnknownSize, err
	}

	req.Header.Set("User-Agent", "fssn/1.0.0")
	resp, err := ct.Do(req)

	if err != nil {
		return nil, UnknownSize, err
	}

	return resp.Header, resp.ContentLength, nil

}
