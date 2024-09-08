package main

import (
	"fmt"
	"time"
)

func ShowProgress(d *Download) {
	defer d.WG.Done()

	//ESC := 27
	//lineCount := len(fs)
	//clearLine := fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

	for d.Status == Running {
		for _, f := range d.Files {
			sb := f.Scope.Start
			eb := f.Scope.End
			fmt.Printf("state: %d | %d / %d\n", f.State, f.getSize(), (eb - sb + 1))
		}
		time.Sleep(50 * time.Millisecond)
		//fmt.Printf(strings.Repeat(clearLine, lineCount))
	}

	for _, f := range d.Files {
		sb := f.Scope.Start
		eb := f.Scope.End
		fmt.Printf("state: %d | %d / %d\n", f.State, f.getSize(), (eb - sb + 1))
	}

}
