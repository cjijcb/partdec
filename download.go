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

	DataCasters []DataCaster

	DLType uint8

	endpoint struct {
		src DataCaster
		dst *FileIO
	}

	IOMode struct {
		Timeout     time.Duration
		UserHeader  http.Header
		NoConnReuse bool
		O_FLAGS     int
	}

	DLOptions struct {
		URI       string
		BasePath  string
		DstDirs   []string
		PartCount int
		PartSize  int64
		ReDL      map[FileState]bool
		UI        func(*Download)
		Force     bool
		*IOMode
	}

	Download struct {
		Files    FileIOs
		Sources  DataCasters
		URI      string
		DataSize int64
		Type     DLType
		ReDL     map[FileState]bool
		UI       func(*Download)
		Flow     *FlowControl
		Stop     context.CancelFunc
		ctx      context.Context
	}
)

const (
	File DLType = iota
	HTTP

	PartSoftLimit      = 128
	MaxConcurrentFetch = 32
)

func (d *Download) Start() (err error) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "%s\n",
				JoinErr(ToErr(r), d.Files.Close(), d.Sources.Close()))
		}
	}()

	d.ctx, d.Stop = signal.NotifyContext(context.Background(), os.Interrupt)
	defer d.Stop()

	if d.UI != nil {
		d.Flow.WG.Add(1)
		go d.UI(d)
	}

	partCount := len(d.Files)
	errCh := make(chan error, partCount)

	d.Flow.WG.Add(1)
	go d.fetchAll(errCh)

	err = CatchErr(errCh, partCount)
	d.Stop()

	d.Flow.WG.Wait()
	return err

}

func (d *Download) fetchAll(errCh chan error) {

	defer func() {
		d.Flow.WG.Done()
		if r := recover(); r != nil {
			errCh <- ToErr(r)
		}
	}()

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
		go d.fetch(&endpoint{dc, fio}, errCh)
	}

}

func (d *Download) fetch(ep *endpoint, errCh chan<- error) {

	defer func() {
		d.Flow.WG.Done()
		d.Flow.Release()
		if r := recover(); r != nil {
			errCh <- ToErr(r)
		}
	}()

	dc := ep.src
	fio := ep.dst
	defer fio.Close()

	if err := fio.Open(); err != nil {
		fio.PushState(Broken)
		errCh <- err
		return
	}

	if _, err := fio.Seek(0, io.SeekEnd); err != nil {
		fio.PushState(Broken)
		errCh <- err
		return
	}

	r, err := dc.DataCast(fio.Scope)
	defer dc.Close()
	if err != nil {
		errCh <- err
		return
	}

	err = copyX(d.ctx, fio, r)
	if err != nil {
		if IsErr(err, context.Canceled) {
			errCh <- ErrCancel
		} else {
			errCh <- err
			fio.PushState(Broken)
		}
		return
	}

	fio.PushState(Completed)
	errCh <- nil

}

func (d *Download) InitFiles(partSize int64) (err error) {

	if err := d.Files.SetByteRange(d.DataSize, partSize); err != nil {
		return err
	}

	if err = d.Files.SetInitialState(); err != nil {
		return err
	}

	if err := d.Files.RenewByState(d.ReDL); err != nil {
		return err
	}

	return nil

}

func NewDownload(opt *DLOptions) (d *Download, err error) {

	switch {
	case IsFile(opt.URI):
		d, err = NewFileDownload(opt)
	case IsURL(opt.URI):
		d, err = NewHTTPDownload(opt)
	default:
		return nil, NewErr("%s: %s", ErrFileURL, opt.URI)
	}

	if err != nil {
		return nil, err
	}

	if err = d.InitFiles(opt.PartSize); err != nil {
		return nil, err
	}

	d.Flow = NewFlowControl(MaxConcurrentFetch)
	return d, nil

}

func NewHTTPDownload(opt *DLOptions) (*Download, error) {

	if md := opt.IOMode; md != nil {
		for k := range md.UserHeader {
			SharedHeader.Set(k, md.UserHeader.Get(k))
		}
		SharedTransport.DisableKeepAlives = md.NoConnReuse
		SharedTransport.ResponseHeaderTimeout = md.Timeout
	}

	hdr, cl, err := GetHeaders(opt.URI)
	if err != nil {
		return nil, err
	}

	if err := opt.AlignPartCountSize(cl); err != nil {
		return nil, err
	}

	opt.ParseBasePath(hdr)

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	return &Download{
		Files:    fios,
		Sources:  make([]DataCaster, 2*MaxConcurrentFetch), //ring buffer
		URI:      opt.URI,
		DataSize: cl,
		Type:     HTTP,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
	}, nil

}

func NewFileDownload(opt *DLOptions) (*Download, error) {

	info, err := os.Stat(opt.URI)
	if err != nil {
		return nil, err
	}
	dataSize := info.Size()

	if err := opt.AlignPartCountSize(dataSize); err != nil {
		return nil, err
	}

	opt.ParseBasePath(nil)

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	return &Download{
		Files:    fios,
		Sources:  make([]DataCaster, 2*MaxConcurrentFetch), //ring buffer
		URI:      opt.URI,
		DataSize: dataSize,
		Type:     File,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
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
		opt.PartCount = 1 + int((dataSize-1)/opt.PartSize) // Ceiling division
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
		gendc               func(string) (DataCaster, error)
		dcs                 = d.Sources
		maxRetry, lastIndex = len(dcs) + 1, len(dcs) - 1
		i                   = -1
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

	return func() (DataCaster, error) {
		dc, err := gendc(d.URI)
		if err != nil {
			return nil, err
		}
		for range maxRetry {
			i++
			if i > lastIndex {
				i = 0
			}
			if dcs[i] == nil || !dcs[i].IsOpen() {
				dcs[i] = dc
				return dcs[i], nil
			}
		}
		return nil, ErrExhaust
	}

}

func (dcs DataCasters) Close() (err error) {

	for _, dc := range dcs {
		if dc != nil && dc.IsOpen() {
			err = JoinErr(err, dc.Close())
		}
	}
	return err

}

func copyX(ctx context.Context, w io.WriteCloser, r io.ReadCloser) (err error) {

	go func() {
		<-ctx.Done()
		r.Close()
		w.Close()
	}()
	_, err = io.Copy(w, r)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err

}
