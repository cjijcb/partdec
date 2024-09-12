package main

import ()

func main() {

	const FileNumParts int = 3
	//URI := "http://ipv4.download.thinkbroadband.com/5MB.zip"
	URI := "trusrc.dat"

	d, err  := buildDownload(FileNumParts, URI)
	doHandle(err)
	d.Start()

}
