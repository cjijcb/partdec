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
)

type (
	WebIO struct {
		*http.Client
		*http.Request
		Body   io.ReadCloser
		isOpen bool
	}
)

const (
	UserAgent = "partdec/0.2.6"
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

func NewWebIO(ct *http.Client, rawURL string) (*WebIO, error) {

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header = SharedHeader.Clone()

	wbio := &WebIO{
		Client:  ct,
		Request: req,
		isOpen:  true,
	}

	return wbio, nil

}

func (wbio *WebIO) DataCast(br ByteRange) (io.ReadCloser, error) {

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

	wbio, err := NewWebIO(
		&http.Client{Transport: SharedTransport},
		rawURL,
	)

	if err != nil {
		return nil, err
	}

	return wbio, nil

}

func (wbio *WebIO) IsOpen() bool {

	mtx.Lock()
	defer mtx.Unlock()
	return wbio.isOpen

}

func (wbio *WebIO) Close() error {

	mtx.Lock()
	defer mtx.Unlock()

	wbio.isOpen = false
	if wbio.Body != nil {
		if err := wbio.Body.Close(); err != nil {
			return err
		}
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

func GetHeaders(rawURL string) (http.Header, int64, error) {

	ct := &http.Client{Transport: SharedTransport}

	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return nil, UnknownSize, err
	}

	req.Header = SharedHeader

	contLen := func(resp *http.Response) int64 {
		if resp.Header.Get("Accept-Ranges") != "bytes" {
			return UnknownSize
		}
		return resp.ContentLength
	}

	resp, err := ct.Do(req)
	if err == nil && resp.ContentLength != UnknownSize {
		return resp.Header, contLen(resp), nil
	}

	req.Method = http.MethodGet
	resp, err = ct.Do(req)

	if err == nil {
		defer resp.Body.Close()
		return resp.Header, contLen(resp), nil
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

	url, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	if url.Path != "" {
		return path.Base(url.Path)
	}

	return "index.html"

}
