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
	"time"
)

type (
	DataCaster interface {
		DataCast(ByteRange) (io.ReadCloser, error)
		Close() error
		IsOpen() bool
	}

	DataCasters []DataCaster

	DLStatus uint8
	DLType   uint8
	EndPoint struct {
		Src DataCaster
		Dst *FileIO
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
		Flow     *FlowControl
		URI      string
		DataSize int64
		Type     DLType
		Status   DLStatus
		ReDL     map[FileState]bool
		UI       func(*Download)
		Cancel   context.CancelFunc
		*IOMode
	}
)

const (
	Pending DLStatus = iota
	Running
	Stopping
	Stopped
	Local DLType = iota
	Online

	PartSoftLimit = 128
	MaxFetch      = 32
)

func (d *Download) Start() error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "%s\n",
				JoinErr(ToErr(r), d.Files.Close(), d.Sources.Close()))
			d.PushStatus(Stopped)
		}
	}()

	var fetchErr error
	var ctx context.Context

	ctx, d.Cancel = context.WithCancel(context.Background())
	defer d.Cancel()

	if d.UI != nil {
		d.Flow.WG.Add(1)
		go d.UI(d)
	}

	d.PushStatus(Running)
	partCount := len(d.Files)
	errCh := make(chan error, partCount)

	d.Flow.WG.Add(1)
	go d.Fetch(ctx, errCh)

	if fetchErr = CatchErr(errCh, partCount); fetchErr != nil {
		d.Cancel()
	}

	d.PushStatus(Stopping)

	d.Flow.WG.Wait()
	d.PushStatus(Stopped)
	return fetchErr
}

func (d *Download) Fetch(ctx context.Context, errCh chan error) {
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
			fio.Close()
			errCh <- nil
			continue
		}

		<-d.Flow.Acquire(d.Flow.Limiter)
		d.Flow.WG.Add(1)
		go d.fetch(ctx, &EndPoint{dc, fio}, errCh)
	}
}

func (d *Download) fetch(ctx context.Context, ep *EndPoint, errCh chan<- error) {
	defer func() {
		d.Flow.WG.Done()
		d.Flow.Release(d.Flow.Limiter)
		if r := recover(); r != nil {
			errCh <- ToErr(r)
		}
	}()

	dc := ep.Src
	fio := ep.Dst
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

	err = CopyX(ctx, fio, r)
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

func NewDownload(opt *DLOptions) (*Download, error) {

	var d *Download
	var err error

	switch {
	case IsFile(opt.URI):
		d, err = NewLocalDownload(opt)
	case IsURL(opt.URI):
		d, err = NewOnlineDownload(opt)
	default:
		return nil, NewErr("%s: %s", ErrFileURL, opt.URI)
	}

	if err != nil {
		return nil, err
	}

	if err := d.Files.SetByteRange(d.DataSize, opt.PartSize); err != nil {
		return nil, err
	}

	if err = d.Files.SetInitState(); err != nil {
		return nil, err
	}

	if err := d.Files.RenewByState(d.ReDL); err != nil {
		return nil, err
	}

	for _, b := range d.ReDL {
		if !b {
			continue
		}
		if err := d.Files.SetByteRange(d.DataSize, opt.PartSize); err != nil {
			return nil, err
		}
		break
	}

	d.Flow = NewFlowControl(MaxFetch)
	return d, nil

}

func NewOnlineDownload(opt *DLOptions) (*Download, error) {

	SharedTransport.DisableKeepAlives = opt.IOMode.NoConnReuse
	SharedTransport.ResponseHeaderTimeout = opt.IOMode.Timeout

	if opt.IOMode != nil {
		md := opt.IOMode
		for k := range md.UserHeader {
			SharedHeader.Set(k, md.UserHeader.Get(k))
		}
	}

	hdr, cl, err := GetHeaders(opt.URI)
	if err != nil {
		return nil, err
	}

	opt.AlignPartCountSize(cl)

	if cl != UnknownSize && (opt.PartCount > int(cl) || opt.PartSize > cl) {
		return nil, ErrPartExceed
	}

	if opt.PartCount > PartSoftLimit && !opt.Force {
		return nil, NewErr("%s of %d: %d ", ErrPartLimit, PartSoftLimit, opt.PartCount)
	}

	basePath := opt.BasePath
	switch {
	case basePath == "":
		basePath = NewFileName(opt.URI, hdr)
	case IsEndSeparator(basePath):
		basePath += NewFileName(opt.URI, hdr)
	}

	fios, err := BuildFileIOs(opt.PartCount, basePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	return &Download{
		Files:    fios,
		Sources:  make([]DataCaster, 2*MaxFetch), //ring buffer
		URI:      opt.URI,
		DataSize: cl,
		Type:     Online,
		Status:   Pending,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
		IOMode:   opt.IOMode,
	}, nil

}

func NewLocalDownload(opt *DLOptions) (*Download, error) {

	info, err := os.Stat(opt.URI)
	if err != nil {
		return nil, err
	}
	dataSize := info.Size()

	opt.AlignPartCountSize(dataSize)

	if opt.PartCount > int(dataSize) || opt.PartSize > dataSize {
		return nil, ErrPartExceed
	}

	if opt.PartCount > PartSoftLimit && !opt.Force {
		return nil, NewErr("%s of %s: %s ", ErrPartLimit, PartSoftLimit, opt.PartCount)
	}

	basePath := opt.BasePath
	switch {
	case basePath == "":
		basePath = NewFileName(opt.URI, nil)
	case IsEndSeparator(basePath):
		basePath += NewFileName(opt.URI, nil)
	}

	fios, err := BuildFileIOs(opt.PartCount, basePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	return &Download{
		Files:    fios,
		Sources:  make([]DataCaster, 2*MaxFetch), //ring buffer
		URI:      opt.URI,
		DataSize: dataSize,
		Type:     Local,
		Status:   Pending,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
		IOMode:   opt.IOMode,
	}, nil

}

func NewFileName(uri string, hdr http.Header) string {

	if fileName := newFileNameFromHeader(hdr); fileName != "" {
		return fileName
	}

	if fileName := newFileNameFromPath(uri); fileName != "" {
		return fileName
	}

	if fileName := newFileNameFromURL(uri); fileName != "" {
		return fileName
	}

	return "unknown.partdec"

}

func (opt *DLOptions) AlignPartCountSize(dataSize int64) {

	if dataSize == UnknownSize {
		opt.PartCount = 1
		opt.PartSize = UnknownSize
		return
	}

	if opt.PartCount < 0 {
		opt.PartCount = 1
	}

	if opt.PartSize < 1 {
		opt.PartSize = UnknownSize
	}

	if opt.PartSize > 1 {
		opt.PartCount = int(dataSize / opt.PartSize)
		if dataSize%opt.PartSize != 0 {
			opt.PartCount++
		}
	}

}

func (d *Download) DataCasterGenerator() func() (DataCaster, error) {

	var (
		gendc               func(string, *IOMode) (DataCaster, error)
		dcs                 = d.Sources
		maxRetry, lastIndex = len(dcs) + 1, len(dcs) - 1
		i                   = -1
	)
	switch d.Type {
	case Local:
		gendc = NewFileDataCaster
	case Online:
		gendc = NewWebDataCaster
	default:
		gendc = func(string, *IOMode) (DataCaster, error) {
			return nil, ErrDLType
		}
	}

	return func() (DataCaster, error) {
		dc, err := gendc(d.URI, d.IOMode)
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

func (d *Download) PullStatus() DLStatus {

	mtx.Lock()
	defer mtx.Unlock()
	return d.Status
}

func (d *Download) PushStatus(ds DLStatus) {

	mtx.Lock()
	defer mtx.Unlock()
	d.Status = ds
}

func (dcs DataCasters) Close() error {

	var err error
	for _, dc := range dcs {
		if dc != nil && dc.IsOpen() {
			err = JoinErr(err, dc.Close())
		}
	}
	return err

}

func CopyX(ctx context.Context, w io.WriteCloser, r io.ReadCloser) (err error) {

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
