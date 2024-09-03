package main

import (
	"fmt"
	//"strings"
	"sync"
	"time"
)

func doPrintDLProgress(fs FileIOs, wg *sync.WaitGroup) {
	defer wg.Done()

	//ESC := 27
	//lineCount := len(fs)
	//clearLine := fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

	for _, f := range fs {
		<-f.WriteSIG
	}

	for fs.getTotalWriter() > 0 {
		fmt.Println("writer:", fs.getTotalWriter())
		for _, f := range fs {
			sb := f.StartByte
			eb := f.EndByte
			fmt.Printf("state: %d | %d / %d\n", f.State, f.getSize(), (eb - sb + 1))
		}
		time.Sleep(50 * time.Millisecond)
		//fmt.Printf(strings.Repeat(clearLine, lineCount))

	}

}
