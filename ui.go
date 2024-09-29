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
		for _, fio := range d.Files {
			size, _ := fio.Size()
			sb := fio.Scope.Start
			eb := fio.Scope.End
			fmt.Printf(
				"state: %d | %d / %d | %s\n",
				fio.State,
				size,
				(eb - sb + 1),
				fio.Path.Relative,
			)
		}
		time.Sleep(250 * time.Millisecond)
		//fmt.Printf(strings.Repeat(clearLine, lineCount))
	}
	for _, fio := range d.Files {
		size, _ := fio.Size()
		sb := fio.Scope.Start
		eb := fio.Scope.End
		fmt.Printf("state: %d | %d / %d\n", fio.State, size, (eb - sb + 1))
	}

}
