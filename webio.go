/*
Copyright 2024 Carlo Jay I. Jacaba

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package partdec

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"time"
)

type (
	WebIO struct {
		Client  *http.Client
		Request *http.Request
		Body    io.ReadCloser
		isOpen  bool
	}
)

const (
	UserAgent = "partdec/1.0.0"
)

var (
	SharedTransport = &http.Transport{
		MaxIdleConnsPerHost: MaxFetch,
		DisableKeepAlives:   false,
	}

	SharedHeader = http.Header{
		"Accept":     []string{"*/*"},
		"User-Agent": []string{UserAgent},
	}
)

func NewWebIO(ct *http.Client, req *http.Request) *WebIO {
	wbio := &WebIO{
		Client:  ct,
		Request: req,
		isOpen:  true,
	}

	return wbio
}

func NewReq(method string, rawURL string) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header = SharedHeader.Clone()

	return req, nil
}

func NewClient() *http.Client {
	ct := &http.Client{Transport: SharedTransport}
	return ct
}

func (wbio *WebIO) DataCast(br ByteRange) (io.Reader, error) {

	if !br.isFullRange { //overwrite Range header when there's partitioning
		wbio.Request.Header.Set("Range", BuildRangeHeader(br))
	}

	resp, err := wbio.Client.Do(wbio.Request)
	if err != nil {
		return nil, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return nil, NewErr(resp.Status)
	}

	wbio.Body = resp.Body

	return wbio.Body, nil
}

func NewWebDataCaster(rawURL string, md *IOMode) (DataCaster, error) {
	req, err := NewReq(http.MethodGet, rawURL)
	if err != nil {
		return nil, err
	}
	wbio := NewWebIO(NewClient(), req)

	wbio.Client.Timeout = md.Timeout

	return wbio, nil
}

func (wbio *WebIO) IsOpen() bool {
	return wbio.isOpen
}

func (wbio *WebIO) Close() error {
	if wbio.Body != nil {
		if err := wbio.Body.Close(); err != nil {
			return err
		}
		wbio.isOpen = false
	}
	return nil
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

func GetHeaders(rawURL string, to time.Duration) (http.Header, int64, error) {

	ct := &http.Client{Transport: SharedTransport, Timeout: to}

	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return nil, UnknownSize, err
	}

	req.Header = SharedHeader

	resp, err := ct.Do(req)
	if err == nil && resp.ContentLength != UnknownSize {
		return resp.Header, resp.ContentLength, nil
	}

	req.Method = http.MethodGet
	resp, err = ct.Do(req)

	if err == nil {
		defer resp.Body.Close()
		return resp.Header, resp.ContentLength, nil
	}

	return nil, UnknownSize, err
}

func newFileNameFromHeader(hdr http.Header) string {
	if hdr == nil {
		return ""
	}

	cd := hdr.Get("Content-Disposition")
	if cd == "" {
		return ""
	}

	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}

	fileName, ok := params["filename"]
	if !ok || fileName == "" {
		return ""
	}

	return fileName
}

func newFileNameFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Base(parsedURL.Path)
}
