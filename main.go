package main

import (
	"log"
	"runtime"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	partCount := 3
	uri := "http://ipv4.download.thinkbroadband.com/5MB.zip"
	//uri := "trusrc.dat"
	dstDirs := []string{"dir1/", "dir2/"}

	opt := DLOptions{
		URI:       uri,
		BasePath:  "",
		DstDirs:   dstDirs,
		PartCount: partCount,
		ReDL:      map[FileState]bool{Completed: true, Resume: false, Broken: true},
		UI:        ShowProgress,
	}

	d, err := NewDownload(opt)
	if err != nil {
		log.Fatal(err)
	}
	d.Start()

}
