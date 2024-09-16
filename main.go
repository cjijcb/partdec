package main

import (
	"log"
)

func main() {

	const FileNumParts int = 3
	URI := "http://ipv4.download.thinkbroadband.com/5MB.zip"
	//URI := "trusrc.dat"
	dstDirs := []string{"dir1/", "dir2/"}

	d, err := buildDownload(FileNumParts, dstDirs, URI)
	if err != nil {
		log.Fatal(err)
	}
	d.Start()


}
