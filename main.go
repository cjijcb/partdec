package main

import (
    "net/http"
    //"fmt"
	"io"
	"os"
	"sync"
)
	

func main() {


	wg := sync.WaitGroup{}
	
	ch := connWorker(wg)

	f, _ := os.Create("file")
    defer f.Close()

	f.ReadFrom(<-ch)

	for q := range ch {
    	io.Copy(os.Stdout, q)
	}

}


func connWorker(wg sync.WaitGroup) chan io.Reader {

	ch := make(chan io.Reader)

    ct := &http.Client{}

	resp := &http.Response{}

	for i := 1; i<=4; i++ {
		wg.Add(1)
		go func(resp *http.Response, ct *http.Client) {
			defer wg.Done()
			resp, _ = ct.Get("https://example.com")
			ch <- resp.Body
		}(resp, ct)
	}
	
	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch

}
