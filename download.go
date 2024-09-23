package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
)

type (
	DataCaster interface {
		DataCast(ByteRange) (io.ReadCloser, error)
	}

	DLStatus uint8
	DLType   uint8
	EndPoint struct {
		Src DataCaster
		Dst *FileIO
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
		ReDL      map[FileState]bool
		UI        func(*Download)
	}

	Download struct {
		Files    FileIOs
		Sources  []DataCaster
		Flow     *FlowControl
		URI      string
		DataSize int
		Type     DLType
		Status   DLStatus
		ReDL     map[FileState]bool
		UI       func(*Download)
		Cancel   context.CancelFunc
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

	MaxFetch = 1
)

func (d *Download) Start() error {
	defer d.Files.Close()
	var fetchErr error
	var ctx context.Context

	ctx, d.Cancel = context.WithCancel(context.Background())
	defer d.Cancel()

	if d.UI != nil {
		d.Flow.WG.Add(1)
		go d.UI(d)
	}

	d.Status = Running

	errCh := make(chan error, len(d.Files))

	d.Flow.WG.Add(1)
	go d.Fetch(ctx, errCh)

	if fetchErr = CatchErr(errCh); fetchErr != nil {
		d.Cancel()
	}

	d.Files.WaitClosingSIG()
	d.Status = Stopping

	d.Flow.WG.Wait()
	d.Status = Stopped
	return fetchErr
}

func (d *Download) Fetch(ctx context.Context, errCh chan error) {
	defer d.Flow.WG.Done()
	pullDataCaster := DataCasterPuller(d.Sources)

	for _, f := range d.Files {
		src := pullDataCaster()

		if f.State == Completed || f.State == Broken {
			f.ClosingSIG <- true
			continue
		}

		<-d.Flow.Acquire(d.Flow.Limiter)
		d.Flow.WG.Add(1)
		go fetch(ctx, &EndPoint{src, f}, d.Flow, errCh)
	}
}

func fetch(ctx context.Context, ep *EndPoint, fc *FlowControl, errCh chan<- error) {
	defer fc.WG.Done()
	defer fc.Release(fc.Limiter)

	dc := ep.Src
	f := ep.Dst
	defer func() { f.ClosingSIG <- true }()

	if f.State == Unknown {
		err := f.Truncate(0)
		if err != nil {
			errCh <- err
			return
		}
	}

	f.Seek(0, io.SeekEnd)
	r, err := dc.DataCast(f.Scope)
	defer r.Close()

	if err != nil {
		errCh <- err
		return
	}

	_, err = f.ReadFrom(newCtxReader(ctx, r))
	if err != nil {
		errCh <- err
		return
	}
	errCh <- errors.New("test error")

	return

}

func NewDownload(opt DLOptions) (*Download, error) {

	var d *Download
	var err error

	if ok, _ := isFile(opt.URI); ok {
		d, err = NewLocalDownload(opt)
	} else if ok, _ := isURL(opt.URI); ok {
		d, err = NewOnlineDownload(opt)
	} else {
		return nil, errors.New("invalid file or url")
	}

	if err != nil {
		return nil, err
	}

	if err = d.Files.SetByteRange(d.DataSize); err != nil {
		return nil, err
	}

	if err = d.Files.SetInitState(); err != nil {
		return nil, err
	}

	if d.ReDL != nil {
		if err := d.Files.RenewByState(d.ReDL); err != nil {
			return nil, err
		}

		if err := d.Files.SetByteRange(d.DataSize); err != nil {
			return nil, err
		}
	}

	d.Flow = NewFlowControl(MaxFetch)

	return d, nil

}

func NewOnlineDownload(opt DLOptions) (*Download, error) {

	uri := opt.URI
	basePath := opt.BasePath
	dstDirs := opt.DstDirs
	partCount := opt.PartCount

	hdr, cl, err := GetHeaders(uri)
	if err != nil {
		return nil, err
	}

	if cl == UnknownSize {
		partCount = 1
	}

	if basePath == "" {
		basePath = NewFileName(uri, hdr)
	}

	fios, err := BuildFileIOs(partCount, basePath, dstDirs)
	if err != nil {
		return nil, err
	}

	srcs := make([]DataCaster, partCount)
	for i := range partCount {

		ct := NewClient()
		req, err := NewReq(http.MethodGet, uri)
		if err != nil {
			return nil, err
		}
		srcs[i] = NewWebIO(ct, req)

	}

	return &Download{
		Files:    fios,
		Sources:  srcs,
		URI:      uri,
		DataSize: int(cl),
		Type:     Online,
		Status:   Pending,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
	}, nil
}

func NewLocalDownload(opt DLOptions) (*Download, error) {

	uri := opt.URI
	basePath := opt.BasePath
	dstDirs := opt.DstDirs
	partCount := opt.PartCount

	info, err := os.Stat(uri)
	if err != nil {
		return nil, err
	}

	dataSize := info.Size()

	if basePath == "" {
		basePath = NewFileName(uri, nil)
	}

	fios, err := BuildFileIOs(partCount, basePath, dstDirs)
	if err != nil {
		return nil, err
	}

	fio, err := NewFileIO(uri, CurrentDir, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	srcs := make([]DataCaster, 1)
	srcs[0] = fio

	return &Download{
		Files:    fios,
		Sources:  srcs,
		DataSize: int(dataSize),
		Type:     Local,
		Status:   Pending,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
	}, nil

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

func DataCasterPuller(dcs []DataCaster) func() DataCaster {

	maxIndex := len(dcs) - 1
	currentIndex := -1

	return func() DataCaster {
		if currentIndex < maxIndex {
			currentIndex++
			return dcs[currentIndex]
		}
		return dcs[currentIndex]
	}

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
