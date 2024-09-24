package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func ShowProgress(d *Download) {
	defer d.Flow.WG.Done()
	defer d.Cancel()
	//ESC := 27
	//lineCount := len(fs)
	//clearLine := fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		d.Cancel()
	}()

	for d.Status == Pending || d.Status == Running {
		for _, f := range d.Files {
			size, _ := f.Size()
			sb := f.Scope.Start
			eb := f.Scope.End
			fmt.Printf(
				"state: %d | %d / %d | %s\n",
				f.State,
				size,
				(eb - sb + 1),
				f.Path.Relative,
			)
		}
		time.Sleep(250 * time.Millisecond)
		//fmt.Printf(strings.Repeat(clearLine, lineCount))
	}
	for _, f := range d.Files {
		size, _ := f.Size()
		sb := f.Scope.Start
		eb := f.Scope.End
		fmt.Printf("state: %d | %d / %d\n", f.State, size, (eb - sb + 1))
	}

}
