package main

import (
    "net/http"
    //"fmt"
	"io"
	"os"
)
	

func main() {


	ch := connWorker()

	f, _ := os.Create("file")
    defer f.Close()

	f.ReadFrom(<-ch)

}


func connWorker() chan io.Reader {

	ch := make(chan io.Reader)

    c := &http.Client{}

	go func() {
		resp, _ := c.Get("https://example.com")
		ch <- resp.Body
	}()

	return ch

}

