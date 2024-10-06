package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	ESC uint32 = 27
)

type (
	Textile struct {
		*strings.Builder
	}

	FileReport struct {
		FileIOs
		BytesPerSecFunc func() int64
		tkr             *time.Ticker
	}
)

var (
	clearLine = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)
)

func ShowProgress(d *Download) {
	defer d.Flow.WG.Done()
	defer d.Cancel()

	HandleInterrupts(d)

	tl := &Textile{new(strings.Builder)}

	fr := NewFileReport(d.Files)
	defer fr.Flush()

	for d.Status == Pending || d.Status == Running {

		s := Progress(fr, tl)
		fmt.Printf(s)
		time.Sleep(100 * time.Millisecond)

	}

	fmt.Printf(Progress(fr, tl))

}

func Progress(fr *FileReport, tl *Textile) string {
	defer tl.Reset()

	for _, fio := range fr.FileIOs {
		size, _ := fio.Size()
		rs := fio.Scope.Start
		re := fio.Scope.End

		fmt.Fprintf(tl,
			"state: %d | %d / %d | %s\n",
			fio.State,
			size,
			(re - rs + 1),
			fio.Path.Relative,
		)
	}

	fmt.Fprintf(tl, "bps:%d\n", fr.BytesPerSecFunc())

	return tl.String()

}

func NewFileReport(fios FileIOs) *FileReport {

	bpsTicker := time.NewTicker(1 * time.Second)

	return &FileReport{
		FileIOs:         fios,
		BytesPerSecFunc: fios.BytesPerSec(bpsTicker),
		tkr:             bpsTicker,
	}

}

func (fr *FileReport) Flush() {
	fr.tkr.Stop()
}

func (fios FileIOs) BytesPerSec(tkr *time.Ticker) func() int64 {

	var cachedTotal, bps = new(int64), new(int64)

	currentTotal := func() int64 {
		var totalSize int64
		for _, fio := range fios {
			size, _ := fio.Size()
			totalSize += size
		}
		return totalSize
	}

	return func() int64 {

		select {
		case <-tkr.C:
			currentTotal := currentTotal()
			*bps = currentTotal - *cachedTotal
			*cachedTotal = currentTotal
			return *bps
		default:
			return *bps
		}

	}

}

func HandleInterrupts(d *Download) <-chan os.Signal {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		d.Cancel()
		sigCh <- sig
	}()

	return sigCh
}

//func (fio *FileIO) TimedSizer(tkr *time.Ticker) SizeFunc {
//
//	var cachedSize = new(int64)
//	var err error
//
//	return func() (int64, error) {
//
//		select {
//		case <-tkr.C:
//			*cachedSize, err = fio.Size()
//			return *cachedSize, err
//		default:
//			return *cachedSize, err
//		}
//
//	}
//
//}
