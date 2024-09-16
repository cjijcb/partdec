package main

import (
	"log"
)

func main() {

	partCount := 3
	uri := "http://ipv4.download.thinkbroadband.com/5MB.zip"
	//uri := "trusrc.dat"
	dstDirs := []string{"dir1/", "dir2/"}

	opt := DLOptions{
		URI:       uri,
		BasePath:  "",
		DstDirs:   dstDirs,
		PartCount: partCount,
		UI:        ShowProgress,
	}

	d, err := buildDownload(opt)
	if err != nil {
		log.Fatal(err)
	}
	d.Start()

}
