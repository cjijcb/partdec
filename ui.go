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
)

var (
	clearLine = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)
)

func ShowProgress(d *Download) {
	defer d.Flow.WG.Done()
	defer d.Cancel()

	HandleInterrupts(d)

	tl := &Textile{new(strings.Builder)}
	for d.Status == Pending || d.Status == Running {

		s := d.Files.Progress(tl)
		fmt.Printf(s)
		time.Sleep(250 * time.Millisecond)
	}

	fmt.Printf(d.Files.Progress(tl))

}

func (fios FileIOs) Progress(tl *Textile) string {
	defer tl.Reset()

	for _, fio := range fios {
		size, _ := fio.Size()
		sb := fio.Scope.Start
		eb := fio.Scope.End

		fmt.Fprintf(tl,
			"state: %d | %d / %d | %s\n",
			fio.State,
			size,
			(eb - sb + 1),
			fio.Path.Relative,
		)
	}

	return tl.String()

}

func HandleInterrupts(d *Download) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		d.Cancel()
	}()
}
