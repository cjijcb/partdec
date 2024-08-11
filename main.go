package main


import (
 "fmt"
//  "io/fs"
  "os"
//  "time"
//  "net/http"

)


type downloadedFile struct {
	os.File
	Path string
	Size int64
} 


func main() {

	dF := downloadedFile{
		Path: "file",
	}
	
	fi , _ := os.Stat(dF.Path)
	
	dF.Size = fi.Size()


//	tr := &http.Transport{
//		MaxIdleConns:       16,
//		IdleConnTimeout:    60 * time.Second,
//		DisableCompression: true,
//	}
//
//	client := &http.Client{Transport: tr}
//
//	rp, _ := client.Get("http://example.com")
//
//        fmt.Println("Print Me!")
//
//	b, _  :=io.ReadAll(rp.Body)
//
//
//  	os.WriteFile(dF.Path, b, 0666)

	fmt.Println(dF.Size)
}





