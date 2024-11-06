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

	safeWidth = 40

	clearToEnd = string(ESC) + "[0J"
	hideCursor = string(ESC) + "[?25l"
	showCursor = string(ESC) + "[?25h"
	homeCursor = string(ESC) + "[H"
)

var (
	Div    string = strings.Repeat("-", safeWidth)
	upLine        = func(n int) string { return fmt.Sprintf("%c[%dF", ESC, n) }
)

func ShowProgress(d *Download) {
	defer d.Flow.WG.Done()
	defer d.Cancel()

	sig, interrSig := os.Signal(nil), Interrupt()

	tl := &Textile{new(strings.Builder), 0, TermWidth()}

	baseWidth := tl.Width
	baseHeight := tl.Height

	fr := NewFileReport(d.Files, d.DataSize)
	defer fr.Flush()

	fmt.Print(hideCursor)
	var resetDisplay string
	for d.Status == Pending || d.Status == Running {

		select {
		case sig = <-interrSig:
			d.Cancel()
		default:
			fmt.Print(tl.ShowReport(fr))
		}

		if sig != nil {
			break
		}

		switch {
		case baseHeight == 0:
			baseHeight = tl.Height
			fallthrough
		case baseWidth == tl.Width:
			resetDisplay = upLine(tl.Height)
		case baseHeight != tl.Height:
			baseHeight = tl.Height
			resetDisplay = homeCursor
			fallthrough
		case baseWidth != tl.Width:
			baseWidth = tl.Width
			resetDisplay += clearToEnd
		}
		time.Sleep(150 * time.Millisecond)
		fmt.Print(resetDisplay)
	}

	close(fr.UpdateCh)
	fmt.Print(tl.ShowReport(fr) + showCursor)

}

func (tl *Textile) ShowReport(fr *FileReport) string {
	defer tl.Reset()

	termWidth := TermWidth()

	fmt.Fprintf(tl, "%-*s\n", termWidth, Div)
	lineCount := 1

	for _, fio := range fr.FileIOs {
		size, _ := fio.Size()
		partSize := (fio.Scope.End - fio.Scope.Start) + 1
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

	fmt.Fprintf(tl, "%s\n%6.2f%%  |%12s%s  |  %s\n", //two lines
		Div,
		percentSec,
		ToEIC(bytesSec), "/s",
		fr.Elapsed(),
	)

	lineCount += 2

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

	*percentSec = 0

	update := func() {
		currentTotal := fr.FileIOs.TotalSize()

		if dataSize != UnknownSize {
			*percentSec = (float32(currentTotal) / float32(dataSize)) * 100
		}

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

	var report string
	if hours > 0 {
		report += fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		report += fmt.Sprintf("%dm", minutes)
	}
	if seconds > 0 || report == "" {
		report += fmt.Sprintf("%ds", seconds)
	}

	return report
}

func Interrupt() <-chan os.Signal {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		sigCh <- sig
	}()

	return sigCh
}

func ToEIC(b int64) string {

	switch {
	case b < 0:
		return fmt.Sprintf("unknown")
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
