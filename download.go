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
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type (
	DataCaster interface {
		DataCast(ByteRange) (io.ReadCloser, error)
		Close() error
		IsOpen() bool
	}

	IOMod struct {
		Retry       int
		Timeout     time.Duration
		UserHeader  http.Header
		NoConnReuse bool
	}

	DLType uint8

	DLOptions struct {
		URI       string
		BasePath  string
		DstDirs   []string
		PartCount int
		PartSize  int64
		ReDL      FileResets
		UI        func(*Download)
		Force     bool
		Mod       *IOMod
	}

	Download struct {
		Files     FileIOs
		Sources   []DataCaster
		URI       string
		DataSize  int64
		Type      DLType
		UI        func(*Download)
		Resumable bool
		Mod       *IOMod
		Flow      *FlowControl
		Stop      context.CancelFunc
		Ctx       context.Context
	}

	endpoint struct {
		c   context.Context
		dc  DataCaster
		fio *FileIO
		r   io.ReadCloser
		w   io.WriteCloser
	}
)

const (
	File DLType = iota
	HTTP

	PartSoftLimit      = 128
	MaxConcurrentFetch = 32
)

func (d *Download) Start() (err error) {

	d.Ctx, d.Stop = signal.NotifyContext(context.Background(), os.Interrupt)
	defer d.Stop()

	if d.UI != nil {
		d.Flow.WG.Add(1)
		go d.UI(d)
	}

	partCount := len(d.Files)
	errCh := make(chan error, partCount)

	d.Flow.WG.Add(1)
	go d.fetchAll(errCh)

	err = catchErr(errCh, partCount)
	d.Stop()

	d.Flow.WG.Wait()
	return err

}

func (d *Download) fetchAll(errCh chan error) {

	defer d.Flow.WG.Done()

	gendc := d.DataCasterGenerator()

	for _, fio := range d.Files {
		dc, err := gendc()

		if err != nil {
			errCh <- JoinErr(err, ErrAbort)
			return
		}

		if fio.State == Completed || fio.State == Broken {
			dc.Close()
			fio.Close()
			errCh <- nil
			continue
		}

		d.Flow.Acquire()
		d.Flow.WG.Add(1)
		go d.fetch(
			&endpoint{c: d.Ctx, dc: dc, fio: fio},
			errCh,
		)
	}

}

func (d *Download) fetch(e *endpoint, errCh chan<- error) {

	defer d.Flow.WG.Done()
	defer d.Flow.Release()
	defer e.dc.Close()

	if err := e.fio.Open(); err != nil {
		e.fio.PushState(Broken)
		errCh <- err
		return
	}
	defer e.fio.Close()

	if _, err := e.fio.Seek(0, io.SeekEnd); err != nil {
		e.fio.PushState(Broken)
		errCh <- err
		return
	}

	err := e.copyWithRetry(d.Mod.Retry)
	if err != nil {
		if !IsErr(err, context.Canceled) {
			e.fio.PushState(Broken)
		}
		errCh <- err
		return
	}

	e.fio.PushState(Completed)
	errCh <- nil

}

func (d *Download) InitFiles(partSize int64, fr FileResets) (err error) {

	if err := d.Files.SetByteRange(d.DataSize, partSize); err != nil {
		return err
	}

	switch {
	case !d.Resumable:
		for _, fio := range d.Files {
			fio.State = Unknown
			fio.Scope.Indeterminate = true
		}
	default:
		if err = d.Files.SetInitialState(); err != nil {
			return err
		}
	}

	if err = d.Files.RenewByState(fr); err != nil {
		return err
	}

	return nil

}

func NewDownload(opt *DLOptions) (d *Download, err error) {

	switch {
	case IsFile(opt.URI):
		d, err = newFileDownload(opt)
	case IsURL(opt.URI):
		d, err = newHTTPDownload(opt)
	default:
		return nil, NewErr("%s: %s", ErrFileURL, opt.URI)
	}

	if err != nil {
		return nil, err
	}

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	d.Files = fios

	if err = d.InitFiles(opt.PartSize, opt.ReDL); err != nil {
		return nil, err
	}

	d.Sources = make([]DataCaster, 2*MaxConcurrentFetch) //ring buffer
	d.URI = opt.URI
	d.UI = opt.UI
	d.Flow = NewFlowControl(MaxConcurrentFetch)
	d.Mod = opt.Mod

	return d, nil

}

func newHTTPDownload(opt *DLOptions) (*Download, error) {

	if md := opt.Mod; md != nil {
		for k := range md.UserHeader {
			SharedHeader.Set(k, md.UserHeader.Get(k))
		}
		SharedTransport.DisableKeepAlives = md.NoConnReuse
		SharedTransport.ResponseHeaderTimeout = md.Timeout
	}

	var cl int64 = UnknownSize
	resumable := true

	hdr, err := getRespInfo(&opt.URI, &cl)
	if err != nil {
		return nil, err
	}

	if hdr.Get("Accept-Ranges") != "bytes" || cl < 0 {
		resumable = false
	}

	if !resumable && (opt.PartCount > 1 || opt.PartSize > 0) {
		fmt.Fprintf(Stderr, "%s\n", ErrMultPart)
		opt.PartCount = 1
	}

	if err := opt.AlignPartCountSize(cl); err != nil {
		return nil, err
	}

	opt.ParseBasePath(hdr)

	return &Download{
		DataSize:  cl,
		Type:      HTTP,
		Resumable: resumable,
	}, nil

}

func newFileDownload(opt *DLOptions) (*Download, error) {

	info, err := os.Stat(opt.URI)
	if err != nil {
		return nil, err
	}
	fs := info.Size()

	if err := opt.AlignPartCountSize(fs); err != nil {
		return nil, err
	}

	opt.ParseBasePath(nil)

	return &Download{
		DataSize:  fs,
		Type:      File,
		Resumable: true,
	}, nil

}

func NewFileName(uri string, hdr http.Header) string {

	if fileName := newFileNameFromHeader(hdr); fileName != "" {
		return fileName
	}

	if fileName := newFileNameFromURL(uri); fileName != "" {
		return fileName
	}

	if fileName := newFileNameFromPath(uri); fileName != "" {
		return fileName
	}

	return "unknown.partdec"

}

func (opt *DLOptions) AlignPartCountSize(dataSize int64) error {

	if dataSize < 1 {
		opt.PartCount = 1
		opt.PartSize = UnknownSize
		return nil
	}

	switch {
	case opt.PartSize < 1:
		opt.PartSize = UnknownSize
	default:
		opt.PartCount = 1 + int((dataSize-1)/opt.PartSize) //ceiling division
	}

	if opt.PartCount < 1 {
		opt.PartCount = 1
	}

	if opt.PartCount > int(dataSize) || opt.PartSize > dataSize {
		return NewErr("%s: %d", ErrPartExceed, dataSize)
	}

	if opt.PartCount > PartSoftLimit && !opt.Force {
		return NewErr("%s of %d: %d", ErrPartLimit, PartSoftLimit, opt.PartCount)
	}

	return nil

}

func (opt *DLOptions) ParseBasePath(hdr http.Header) {

	switch {
	case opt.BasePath == "":
		opt.BasePath = NewFileName(opt.URI, hdr)
	case IsEndSeparator(opt.BasePath):
		opt.BasePath += NewFileName(opt.URI, hdr)
	}

}

func (d *Download) DataCasterGenerator() func() (DataCaster, error) {

	var (
		gendc   func(string) (DataCaster, error)
		dcs     = d.Sources
		retries = len(dcs) + 1
		x       = 0
	)
	switch d.Type {
	case File:
		gendc = NewFileDataCaster
	case HTTP:
		gendc = NewHTTPDataCaster
	default:
		gendc = func(string) (DataCaster, error) {
			return nil, ErrDLType
		}
	}

	return func() (dc DataCaster, err error) {
		if dc, err = gendc(d.URI); err != nil {
			return nil, err
		}
		for range retries {
			x = (x + 1) % len(dcs) //circular indexing
			if dcs[x] == nil || !dcs[x].IsOpen() {
				dcs[x] = dc
				return dcs[x], nil
			}
		}
		return nil, ErrExhaust
	}

}

func (e *endpoint) copyWithRetry(retries int) (err error) {

	go func() {
		<-e.c.Done()
		if e.r != nil {
			e.r.Close()
		}
		e.fio.Close()
	}()

	delay := time.Duration(0)
	t := 0
	for {
		select {
		case <-time.After(delay):

			if e.r, err = e.dc.DataCast(e.fio.Scope); err == nil {
				if _, err = io.Copy(e.fio, e.r); err == nil {
					return nil
				}
			}

			if t++; t >= retries {
				if e.c.Err() != nil {
					return e.c.Err()
				}
				return err
			}

			if err = e.fio.SetOffset(); err != nil {
				return err
			}

			delay = time.Second * 1 << min(t-1, 5) //32s max delay

		case <-e.c.Done():
			return e.c.Err()
		}

	}

}
