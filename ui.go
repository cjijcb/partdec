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
		PercentBytesPerSecFunc func() (float32, int64)
		tkr                    *time.Ticker
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

	fr := NewFileReport(d.Files, d.DataSize)
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

	percentSec, bytesSec := fr.PercentBytesPerSecFunc()
	fmt.Fprintf(tl, "bps:%d %.2f", bytesSec, percentSec)
	tl.WriteString("%%\n")

	return tl.String()

}

func NewFileReport(fios FileIOs, dataSize int64) *FileReport {

	bpsTicker := time.NewTicker(time.Second)

	return &FileReport{
		FileIOs:                fios,
		PercentBytesPerSecFunc: fios.PercentBytesPerSec(dataSize, bpsTicker),
		tkr:                    bpsTicker,
	}

}

func (fr *FileReport) Flush() {
	fr.tkr.Stop()
}

func (fios FileIOs) PercentBytesPerSec(dataSize int64, tkr *time.Ticker) func() (float32, int64) {

	var percentSec, bytesSec, cachedTotal = new(float32), new(int64), new(int64)

	currentTotal := func() int64 {
		var totalSize int64
		for _, fio := range fios {
			size, _ := fio.Size()
			totalSize += size
		}
		return totalSize
	}

	return func() (float32, int64) {

		select {
		case <-tkr.C:
			currentTotal := currentTotal()
			*percentSec = (float32(currentTotal) / float32(dataSize)) * 100
			*bytesSec = currentTotal - *cachedTotal
			*cachedTotal = currentTotal
			return *percentSec, *bytesSec
		default:
			return *percentSec, *bytesSec
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
