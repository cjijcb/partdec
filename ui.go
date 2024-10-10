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

	BytesPerSec   = int64
	PercentPerSec = float32

	FileReport struct {
		FileIOs
		ReportFunc func() (PercentPerSec, BytesPerSec)
		tkr        *time.Ticker
		startTime  time.Time
		final      chan struct{}
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

	close(fr.final)
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

	percentSec, bytesSec := fr.ReportFunc()
	fmt.Fprintf(tl, "bps:%d %9.2f", bytesSec, percentSec)
	fmt.Fprint(tl, "%%")
	fmt.Fprintf(tl, " %20s\n", fr.Elapsed())

	return tl.String()

}

func NewFileReport(fios FileIOs, dataSize int64) *FileReport {

	persec := time.NewTicker(time.Second)

	fr := &FileReport{
		FileIOs:   fios,
		tkr:       persec,
		startTime: time.Now(),
		final:     make(chan struct{}),
	}

	fr.ReportFunc = fr.Reporter(dataSize)

	return fr

}

func (fr *FileReport) Flush() {
	fr.tkr.Stop()
}

func (fr *FileReport) Reporter(dataSize int64) func() (PercentPerSec, BytesPerSec) {

	var percentSec, bytesSec, cachedTotal = new(float32), new(int64), new(int64)

	update := func() {
		currentTotal := fr.FileIOs.TotalSize()
		*percentSec = (float32(currentTotal) / float32(dataSize)) * 100
		*bytesSec = currentTotal - *cachedTotal
		*cachedTotal = currentTotal
	}
	return func() (PercentPerSec, BytesPerSec) {

		select {
		case <-fr.tkr.C:
			update()
			return *percentSec, *bytesSec
		case <-fr.final:
			update()
			return *percentSec, *bytesSec
		default:
			return *percentSec, *bytesSec
		}

	}

}

func (fr *FileReport) Elapsed() string {

	elapsed := time.Since(fr.startTime)
	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
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
