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
		PartSize  int
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

	MaxFetch = 3
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
	partCount := len(d.Files)

	errCh := make(chan error, partCount)

	d.Flow.WG.Add(1)
	go d.Fetch(ctx, errCh)

	if fetchErr = CatchErr(errCh, partCount); fetchErr != nil {
		d.Cancel()
	}
	d.Status = Stopping

	d.Flow.WG.Wait()
	d.Status = Stopped
	return errors.Join(fetchErr, d.Files.Error())
}

func (d *Download) Fetch(ctx context.Context, errCh chan error) {
	defer d.Flow.WG.Done()
	pullDataCaster := DataCasterPuller(d.Sources)

	for _, f := range d.Files {
		src := pullDataCaster()

		if f.State == Completed || f.State == Broken {
			f.Err = f.Close()
			errCh <- nil
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
	defer func() { f.Err = f.Close() }()

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

	errCh <- nil

}

func NewDownload(opt DLOptions) (*Download, error) {

	var d *Download
	var err error

	if ok, _ := isFile(opt.URI); ok {
		d, err = NewLocalDownload(&opt)
	} else if ok, _ := isURL(opt.URI); ok {
		d, err = NewOnlineDownload(&opt)
	} else {
		return nil, errors.New("invalid file or url")
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

	if d.ReDL != nil {
		if err := d.Files.RenewByState(d.ReDL); err != nil {
			return nil, err
		}

		if err := d.Files.SetByteRange(d.DataSize, opt.PartSize); err != nil {
			return nil, err
		}
	}

	d.Flow = NewFlowControl(MaxFetch)

	return d, nil

}

func NewOnlineDownload(opt *DLOptions) (*Download, error) {

	hdr, cl, err := GetHeaders(opt.URI)
	if err != nil {
		return nil, err
	}

	opt.AlignPartCountSize(int(cl))

	if opt.BasePath == "" {
		opt.BasePath = NewFileName(opt.URI, hdr)
	}

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	srcs := make([]DataCaster, opt.PartCount)
	for i := range opt.PartCount {

		ct := NewClient()
		req, err := NewReq(http.MethodGet, opt.URI)
		if err != nil {
			return nil, err
		}
		srcs[i] = NewWebIO(ct, req)

	}

	return &Download{
		Files:    fios,
		Sources:  srcs,
		URI:      opt.URI,
		DataSize: int(cl),
		Type:     Online,
		Status:   Pending,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
	}, nil
}

func NewLocalDownload(opt *DLOptions) (*Download, error) {

	info, err := os.Stat(opt.URI)
	if err != nil {
		return nil, err
	}

	dataSize := info.Size()

	opt.AlignPartCountSize(int(dataSize))

	if opt.BasePath == "" {
		opt.BasePath = NewFileName(opt.URI, nil)
	}

	fios, err := BuildFileIOs(opt.PartCount, opt.BasePath, opt.DstDirs)
	if err != nil {
		return nil, err
	}

	fio, err := NewFileIO(opt.URI, CurrentDir, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	srcs := make([]DataCaster, 1)
	srcs[0] = fio

	return &Download{
		Files:    fios,
		Sources:  srcs,
		URI:      opt.URI,
		DataSize: int(dataSize),
		Type:     Local,
		Status:   Pending,
		ReDL:     opt.ReDL,
		UI:       opt.UI,
	}, nil

}

func (opt *DLOptions) AlignPartCountSize(dataSize int) {

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
		opt.PartCount = dataSize / opt.PartSize
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
