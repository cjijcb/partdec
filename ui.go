package main

import (
    "fmt"
    "strings"
    "sync"
    "time"
)


func doPrintDLProgress(fs FileIOs, wg *sync.WaitGroup) {
    defer wg.Done()

    ESC := 27
    lineCount := len(fs)
    clearLine := fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

    for _, f := range fs {
        <-f.WriteSIG
    }


    for fs.getTotalWriter() > 0 {
        for _, f := range fs {
            fmt.Printf("%d / %d\n", f.getSize(), (f.bOffE - f.bOffS))
        }
        time.Sleep(50 * time.Millisecond)
        fmt.Printf(strings.Repeat(clearLine, lineCount))

    }

}
