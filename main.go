package main


import (
  "fmt"
  //"io"
  "bufio"
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


type myResponse http.Response

func main() {

	dF := downloadedFile{
		Path: "file",
	}
	
	dClient := &client{}

	dClient.setTransport()

	fmt.Printf("%+v\n", (*dClient).Transport.(*http.Transport))

	myResponse, _ := dClient.Get("https://examplefile.com/file-download/48")

	dF.File, _ = os.Create(dF.Path)

	nR  := bufio.NewReader(myResponse.Body)

	nR.WriteTo(dF.File)



	//io.Copy(dF.File, rp.Body)

        //fmt.Println(err)
	
  	//os.WriteFile(dF.Path, b, 0666)

	fi , _ := os.Stat(dF.Path)
	
	dF.Size = fi.Size()

	fmt.Println(dF.Size)
}


func (r myResponse) Read(p []byte) (n int, err error){
	fmt.Println("this")
	return 0, nil

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





