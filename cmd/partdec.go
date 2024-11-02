package main

import (
	"fmt"
	"github.com/cjijcb/partdec"
	"os"
	"runtime"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	var d *partdec.Download

	opt, err := partdec.NewDLOptions()
	if err != nil {
		os.Exit(1)
	}

	d, err = partdec.NewDownload(opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	err = d.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

}
