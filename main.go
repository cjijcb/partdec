package main

import (
	"log"
	"net/http"
	"runtime"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	partCount := 2
	partSize := -1 //1747626
	uri := "http://ipv4.download.thinkbroadband.com/5MB.zip"
	//uri := "trusrc.dat"
	dstDirs := []string{"dir1/", "dir2/"}

	hdr := make(http.Header)
	hdr.Add("Range", "bytes=0-127")

	opt := DLOptions{
		URI:       uri,
		BasePath:  "",
		DstDirs:   dstDirs,
		PartCount: partCount,
		PartSize:  partSize,
		ReDL:      map[FileState]bool{Completed: true, Resume: false, Broken: true},
		UI:        ShowProgress,
		IOMode: &IOMode{
			UserHeader: hdr,
		},
	}

	d, err := NewDownload(opt)
	if err != nil {
		log.Fatal("Error From Main:", err)
	}

	err = d.Start()
	if err != nil {
		log.Fatal("Error From Main:", err)
	}

}
