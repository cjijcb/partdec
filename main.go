package main


import (
  "fmt"
  "io"
  "os"
  "time"
  "net/http"

)


type downloadedFile struct {
	*os.File
	Path string
	Size int64
}

type client struct {
	http.Client
}

func main() {

	dF := downloadedFile{
		Path: "file",
	}
	
	dClient := &client{}

	dClient.setTransport()

	fmt.Printf("%+v\n", (*dClient).Transport.(*http.Transport))


	rp, err := dClient.Get("https://examplefile.com/file-download/48")
	
	fmt.Println(rp.Header)

        fmt.Println(err)

	//b, err  := io.ReadAll(rp.Body)

	dF.File, _ = os.Create(dF.Path)

	io.Copy(dF.File, rp.Body)

        fmt.Println(err)
	
  	//os.WriteFile(dF.Path, b, 0666)

	fi , _ := os.Stat(dF.Path)
	
	dF.Size = fi.Size()

	fmt.Println(dF.Size)
}


func (c *client) setTransport() {
  
	tr := &http.Transport{
		MaxIdleConns:       16,
		MaxConnsPerHost:    16,
		IdleConnTimeout:    60 * time.Second,
		DisableCompression: true,
	}
	
	c.Transport = tr

}





