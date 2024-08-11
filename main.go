package main


import (
  "fmt"
  "io"
  "os"
  "time"
  "net/http"

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
	

	tr := &http.Transport{
		MaxIdleConns:       16,
		IdleConnTimeout:    60 * time.Second,
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	rp, err := client.Get("http://example.com")

        fmt.Println(err)

	b, err  := io.ReadAll(rp.Body)

        fmt.Println(err)
	
  	os.WriteFile(dF.Path, b, 0666)

	fi , _ := os.Stat(dF.Path)
	
	dF.Size = fi.Size()

	fmt.Println(dF.Size)
}





