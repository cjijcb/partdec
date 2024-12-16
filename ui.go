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
	"strings"
	"time"
	"unicode/utf8"
)

type (
	textBlock struct {
		b        *strings.Builder
		height   int
		width    int
		hChanged bool
		wChanged bool
	}

	report struct {
		fios       FileIOs
		dsize      int64
		fileReport func(int) int
		rateReport func() int
		text       *textBlock
		finalCh    chan struct{}
		onesec     *time.Ticker
		startTime  time.Time
	}
)

const (
	esc        rune = 27
	clearToEnd      = string(esc) + "[0J"
	hideCursor      = string(esc) + "[?25l"
	showCursor      = string(esc) + "[?25h"
	homeCursor      = string(esc) + "[H"
	div             = "----------------------------------------"
)

var (
	upLine = func(n int) string { return fmt.Sprintf("%c[%dF", esc, n) }
)

func ShowProgress(d *Download) {

	defer d.Flow.WG.Done()
	defer d.Stop()

	r := newReport(d.Files, d.DataSize)
	defer r.flush()

	interrupted := false
	fmt.Print(hideCursor)
	var resetDisplay string
	for {

		select {
		case <-d.Ctx.Done():
			interrupted = true
		default:
			fmt.Print(r.show())
			switch {
			case !r.text.wChanged:
				resetDisplay = upLine(r.text.height)
			case r.text.hChanged:
				resetDisplay = homeCursor
				fallthrough
			default:
				resetDisplay += clearToEnd
			}
		}

		if interrupted {
			break
		}

		time.Sleep(200 * time.Millisecond)
		fmt.Print(resetDisplay)
	}

	close(r.finalCh)
	fmt.Print(r.show() + showCursor)

}

func (r *report) show() string {

	defer r.text.b.Reset()

	width := termWidth()

	fmt.Fprintf(r.text.b, "%-*s\n", width, div)
	lineCount := 1

	lineCount += r.fileReport(width)

	fmt.Fprintf(r.text.b, "%s\n", div)
	lineCount += 1

	lineCount += r.rateReport()

	r.text.hChanged = (r.text.height != lineCount)
	r.text.wChanged = (r.text.width != width)

	r.text.height = lineCount
	r.text.width = width

	return r.text.b.String()

}

func newReport(fios FileIOs, dataSize int64) *report {

	tb := &textBlock{
		b:      new(strings.Builder),
		height: len(fios) + 3, //2 div plus 1 rate report
		width:  termWidth(),
	}

	r := &report{
		fios:      fios,
		dsize:     dataSize,
		text:      tb,
		finalCh:   make(chan struct{}, 1),
		onesec:    time.NewTicker(time.Second),
		startTime: time.Now(),
	}

	r.rateReport = r.rateReporter()
	r.fileReport = r.fileReporter()
	return r

}

func (r *report) rateReporter() func() int {

	var bytes, cachedTotal int64
	var percent float32

	rateReport := new(string)

	refresh := func(final bool) {
		currentTotal := r.fios.TotalSize()
		if r.dsize > 0 {
			percent = (float32(currentTotal) / float32(r.dsize)) * 100
		}
		if !final || bytes == 0 {
			bytes = currentTotal - cachedTotal
			cachedTotal = currentTotal
		}
		*rateReport = fmt.Sprintf("%6.2f%%  |%12s%s  |  %s",
			percent,
			toEIC(bytes), "/s",
			r.elapsed(),
		)
	}

	refresh(true)
	return func() int {
		select {
		case <-r.onesec.C:
			refresh(false)
			fmt.Fprintf(r.text.b, "%s\n", *rateReport)
		case <-r.finalCh:
			refresh(true)
			fmt.Fprintf(r.text.b, "%s\n", *rateReport)
		default:
			fmt.Fprintf(r.text.b, "%s\n", *rateReport)
		}
		return 1
	}

}

func (r *report) fileReporter() func(int) int {

	csize := len(r.fios)
	cachedPartSize := make([]string, csize)
	cachedPath := make([]string, csize)
	cachedRuneCount := make([]int, csize)

	for i, fio := range r.fios {
		ps := (fio.Scope.End - fio.Scope.Start) + 1
		pt := fio.Path.Relative
		cachedPartSize[i] = toEIC(ps)
		cachedPath[i] = pt
		cachedRuneCount[i] = utf8.RuneCountInString(pt) + 36
	}

	return func(width int) (lineCount int) {
		for i, fio := range r.fios {
			size, _ := fio.Size()
			path := cachedPath[i]
			runeCount := cachedRuneCount[i]

			pad := 0
			if width >= runeCount {
				pad = width - runeCount
			} else if width > 0 {
				lineCount += runeCount / width
			}

			fmt.Fprintf(r.text.b,
				"%-9s->%11s/%-11s| %-*s\n", //36 chars minus path
				fio.PullState().String(),
				toEIC(size),
				cachedPartSize[i],
				pad,
				path,
			)
			lineCount++

		}
		return lineCount
	}

}

func (r *report) elapsed() (elapsed string) {

	t := time.Since(r.startTime)
	h := int(t.Hours())
	m := int(t.Minutes()) % 60
	s := int(t.Seconds()) % 60

	if h > 0 {
		elapsed += fmt.Sprintf("%dh", h)
	}
	if m > 0 {
		elapsed += fmt.Sprintf("%dm", m)
	}
	if s > 0 || elapsed == "" {
		elapsed += fmt.Sprintf("%ds", s)
	}

	return elapsed

}

func toEIC(b int64) string {

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

func termWidth() int {
	width, _, _ := term.GetSize(int(os.Stdin.Fd()))
	return width
}

func (r *report) flush() {
	r.onesec.Stop()
}
