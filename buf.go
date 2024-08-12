package main

import (
    "io"
    "net/http"
    "os"
	//"bufio"
)

type MyReader struct {
    *io.Reader
}


func main() {

    c := &http.Client{}

    resp, err := c.Get("https://example.com")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

	rr := MyReader{bufio.NewReader(resp.Body)}

	rr.byteTransferTo(os.Stdout)

    //r := MyReader{resp.Body}
    //r.byteTransferTo(os.Stdout)
}

func (r *MyReader) byteTransferTo(w io.Writer) {
    buf := make([]byte, 1024)
    for {
        n, err := r.Read(buf)
        if err != nil && err != io.EOF {
            panic(err)
        }
        if n == 0 {
            break
        }

        if _, err := w.Write(buf[:n]); err != nil {
            panic(err)
        }
    }
}


