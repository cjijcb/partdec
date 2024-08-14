package main

import (
    "net/http"
    //"fmt"
	"io"
	"os"
	"sync"
	"bytes"
)


func main() {



	wg := &sync.WaitGroup{}
	
	ch := connWorker(wg)

	f, _ := os.Create("file")
    defer f.Close()

	//f.ReadFrom(<-ch)

	for q := range ch {
		bf := bytes.NewBuffer(q)

		bxf := io.TeeReader(bf, os.Stdout) 
		f.ReadFrom(bxf)
	}


}


func connWorker(wg *sync.WaitGroup) chan []byte {

	ch := make(chan []byte)
	

	bf := make([]byte, 8)


	req, _ := http.NewRequest("GET", "https://example.com", nil)
    req.Proto = "http/2"
    req.ProtoMajor = 2
    req.ProtoMinor = 0

    ct := &http.Client{}

	//mu := &sync.Mutex{}
	resp := &http.Response{} 


	for i := 1; i<=4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()	

			//mu.Lock()

    		resp, _ = ct.Do(req)
			
			resp.Body.Read(bf)
			ch <- bf

			//mu.Unlock()

		}()
	}
	
	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch

}
