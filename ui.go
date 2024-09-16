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
		time.Sleep(50 * time.Millisecond)
		//fmt.Printf(strings.Repeat(clearLine, lineCount))
	}

	for _, f := range d.Files {
		size, _ := f.Size()
		sb := f.Scope.Start
		eb := f.Scope.End
		fmt.Printf("state: %d | %d / %d\n", f.State, size, (eb - sb + 1))
	}

}
