package main


import (
	"fmt"
	"net/http"
)

func main(){

	//buf := make([]byte, 1024)
 
	c := &http.Client{}

	resp, _ := c.Get("https://example.com")

	fmt.Println((*resp).Body)

}

