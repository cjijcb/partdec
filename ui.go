package main

import (
	"fmt"
	"golang.org/x/term"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

type (
	Textile struct {
		*strings.Builder
		Height, Width int
	}

	BytesPerSec   = int64
	PercentPerSec = float32

	FileReport struct {
		FileIOs
		ReportFunc func() (PercentPerSec, BytesPerSec)
		UpdateCh   chan struct{}
		tkr        *time.Ticker
		startTime  time.Time
	}
)

const (
	ESC rune = 27

	clearToEnd = string(ESC) + "[0J"

	Kibi = 1024
	Mebi = 1024 * 1024
	Gibi = 1024 * 1024 * 1024
	Tebi = 1024 * 1024 * 1024 * 1024
)

var (
	ClearLine = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)
)

func ShowProgress(d *Download) {
	defer d.Flow.WG.Done()
	defer d.Cancel()

	HandleInterrupts(d)

	baseWidth := TermWidth()
	tl := &Textile{new(strings.Builder), 0, baseWidth}

	fr := NewFileReport(d.Files, d.DataSize)
	defer fr.Flush()

	for d.Status == Pending || d.Status == Running {

		fmt.Print(Progress(fr, tl))
		upLine := fmt.Sprintf("%c[%dF", ESC, tl.Height)

		if baseWidth != tl.Width {
			baseWidth = tl.Width
			upLine += clearToEnd
		}

		time.Sleep(150 * time.Millisecond)
		fmt.Print(upLine)
	}

	close(fr.UpdateCh)
	fmt.Print(Progress(fr, tl))

}

func Progress(fr *FileReport, tl *Textile) string {
	defer tl.Reset()

	termWidth := TermWidth()
	lineCount := 0
	for _, fio := range fr.FileIOs {
		size, _ := fio.Size()
		partSize := fio.Scope.End - fio.Scope.Start + 1
		path := fio.Path.Relative
		pad := 0

		lineCount++
		runeCount := utf8.RuneCountInString(path) + 36 //(%-9s->%11s/%-11s| ) = 36 char
		if termWidth >= runeCount {
			pad = termWidth - runeCount
		} else if termWidth > 0 {
			lineCount += runeCount / termWidth
		}

		fmt.Fprintf(tl,
			"%-9s->%11s/%-11s| %-*s\n",
			fio.State.String(),
			ToEIC(size),
			ToEIC(partSize),
			pad,
			path,
		)
	}

	percentSec, bytesSec := fr.ReportFunc()

	fmt.Fprintf(tl, "%6.2f%% %15s/s %19s\n",
		percentSec,
		ToEIC(bytesSec),
		fr.Elapsed(),
	)

	lineCount++
	tl.Height = lineCount
	tl.Width = termWidth

	return tl.String()

}

func NewFileReport(fios FileIOs, dataSize int64) *FileReport {

	persec := time.NewTicker(time.Second)

	fr := &FileReport{
		FileIOs:   fios,
		UpdateCh:  make(chan struct{}, 1),
		tkr:       persec,
		startTime: time.Now(),
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
		case <-fr.UpdateCh:
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

func ToEIC(b int64) string {

	switch {
	case b < Kibi:
		return fmt.Sprintf("%d B", b)
	case b >= Kibi && b < Mebi:
		return fmt.Sprintf("%.2f KiB", float32(b)/Kibi)
	case b >= Mebi && b < Gibi:
		return fmt.Sprintf("%.2f MiB", float32(b)/Mebi)
	case b >= Gibi && b < Tebi:
		return fmt.Sprintf("%.2f GiB", float32(b)/Gibi)
	default:
		return fmt.Sprintf("%.2f TiB", float32(b)/Tebi)
	}

}

func TermWidth() int {
	width, _, _ := term.GetSize(int(os.Stdin.Fd()))
	return width
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
