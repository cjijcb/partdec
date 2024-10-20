package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type (
	DataCaster interface {
		DataCast(ByteRange) (io.Reader, error)
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
		Timeout    time.Duration
		UserHeader http.Header
		O_FLAGS    int
	}
	FlowControl struct {
		WG      *sync.WaitGroup
		Limiter chan struct{}
		Acquire func(chan<- struct{}) <-chan struct{}
		Release func(<-chan struct{})
	}

	DLOptions struct {
		URI       string
		BasePath  string
		DstDirs   []string
		PartCount int
		PartSize  int64
		ReDL      map[FileState]bool
		UI        func(*Download)
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

	ctxReader struct {
		ctx context.Context
		r   io.Reader
	}
)

const (
	Pending DLStatus = iota
	Running
	Stopping
	Stopped
	Local DLType = iota
	Online

	MaxFetch = 32
)

func (d *Download) Start() error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(errJoin(toErr(r), d.Files.Close(), d.Sources.Close()))
			d.Status = Stopped
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

	d.Status = Running
	partCount := len(d.Files)
	errCh := make(chan error, partCount)

	d.Flow.WG.Add(1)
	go d.Fetch(ctx, errCh)

	if fetchErr = ErrCatch(errCh, partCount); fetchErr != nil {
		d.Cancel()
	}
	d.Status = Stopping

	d.Flow.WG.Wait()
	d.Status = Stopped
	return errJoin(fetchErr, d.Files.Close(), d.Sources.Close())
}

func (d *Download) Fetch(ctx context.Context, errCh chan error) {
	defer func() {
		d.Flow.WG.Done()
		if r := recover(); r != nil {
			errCh <- toErr(r)
		}
	}()

	gendc := d.DataCasterGenerator()

	for _, fio := range d.Files {
		dc, err := gendc()
		if err != nil {
			errCh <- errJoin(err, abortErr)
			return
		}

		if fio.State == Completed || fio.State == Broken {
			fio.Close()
			errCh <- nil
			continue
		}

		<-d.Flow.Acquire(d.Flow.Limiter)
		d.Flow.WG.Add(1)
		go fetch(ctx, &EndPoint{dc, fio}, d.Flow, errCh)
	}
}

func fetch(ctx context.Context, ep *EndPoint, fc *FlowControl, errCh chan<- error) {
	defer func() {
		fc.WG.Done()
		fc.Release(fc.Limiter)
		if r := recover(); r != nil {
			errCh <- toErr(r)
		}
	}()

	dc := ep.Src
	fio := ep.Dst
	defer fio.Close()

	if err := fio.Open(); err != nil {
		errCh <- err
		return
	}

	if _, err := fio.Seek(0, io.SeekEnd); err != nil {
		errCh <- err
		return
	}

	r, err := dc.DataCast(fio.Scope)
	defer dc.Close()
	if err != nil {
		errCh <- err
		return
	}

	_, err = fio.ReadFrom(newCtxReader(ctx, r))
	if err != nil {
		errCh <- err
		return
	}

	errCh <- nil

}

func NewDownload(opt DLOptions) (*Download, error) {

	var d *Download
	var err error

	switch {
	case isFile(opt.URI):
		d, err = NewLocalDownload(&opt)
	case isURL(opt.URI):
		d, err = NewOnlineDownload(&opt)
	default:
		return nil, errNew("%s: %s", opt.URI, fileURLErr)
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

	hdr, cl, err := GetHeaders(opt.URI)
	if err != nil {
		return nil, err
	}

	opt.AlignPartCountSize(cl)

	if opt.PartCount > int(cl) || opt.PartSize > cl {
		return nil, partExceedErr
	}

	if opt.BasePath == "" {
		opt.BasePath = NewFileName(opt.URI, hdr)
	}

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	return &Download{
		Files:    fios,
		Sources:  make([]DataCaster, 2*MaxFetch),
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
		return nil, partExceedErr
	}

	if opt.BasePath == "" {
		opt.BasePath = NewFileName(opt.URI, nil)
	}

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	return &Download{
		Files:    fios,
		Sources:  make([]DataCaster, 2*MaxFetch),
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

func NewFlowControl(limit int) *FlowControl {

	limiter := make(chan struct{}, limit)
	acq := func(l chan<- struct{}) <-chan struct{} {
		succeed := make(chan struct{})
		l <- struct{}{}
		close(succeed)
		return succeed
	}
	rls := func(l <-chan struct{}) { <-l }

	return &FlowControl{
		WG:      &sync.WaitGroup{},
		Limiter: limiter,
		Acquire: acq,
		Release: rls,
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
			return nil, dltypeErr
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
		return nil, exhaustErr
	}

}

func (dcs DataCasters) Close() error {

	var err error
	for _, dc := range dcs {
		if dc != nil && dc.IsOpen() {
			err = errJoin(err, dc.Close())
		}
	}
	return err

}

func (r *ctxReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}

func newCtxReader(ctx context.Context, r io.Reader) *ctxReader {
	return &ctxReader{
		ctx: ctx,
		r:   r,
	}
}
