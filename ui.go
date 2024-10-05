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
		FileSizeFuncs
	}

	SizeFunc      func() (int64, error)
	FileSizeFuncs []SizeFunc
)

var (
	clearLine = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)
)

func ShowProgress(d *Download) {
	defer d.Flow.WG.Done()
	defer d.Cancel()

	HandleInterrupts(d)

	tl := &Textile{new(strings.Builder)}

	fr := NewFileReport(d.Files, 500*time.Millisecond)

	for d.Status == Pending || d.Status == Running {

		s := Progress(fr, tl)
		fmt.Printf(s)
		time.Sleep(100 * time.Millisecond)

	}

	fmt.Printf(Progress(fr, tl))

}

func Progress(fr *FileReport, tl *Textile) string {
	defer tl.Reset()

	for i, fio := range fr.FileIOs {
		size, _ := fr.FileSizeFuncs[i]()
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

	return tl.String()

}

func NewFileReport(fios FileIOs, precision time.Duration) *FileReport {

	fSizeFuncs := make([]SizeFunc, len(fios))
	for i, fio := range fios {
		fSizeFuncs[i] = fio.TimedSizer(time.NewTicker(precision))
	}

	return &FileReport{
		FileIOs:       fios,
		FileSizeFuncs: fSizeFuncs,
	}
}

func (fio *FileIO) TimedSizer(tkr *time.Ticker) SizeFunc {

	var cache = new(int64)
	var err error

	return func() (int64, error) {

		select {
		case <-tkr.C:
			*cache, err = fio.Size()
			return *cache, err
		default:
			return *cache, err
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
