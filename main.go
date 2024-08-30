package main

import ()

func main() {

	const FileNumParts int = 3
	URI := "http://ipv4.download.thinkbroadband.com/5MB.zip"

	d := buildDownload(FileNumParts, URI)
	d.Start()

}
