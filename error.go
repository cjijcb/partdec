package main

import (
	"log"
)

func doHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
