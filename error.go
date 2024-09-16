package main

import (
	"log"
)

func FetchErrHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
