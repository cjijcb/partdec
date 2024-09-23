package main

import (
	"log"
)

func FetchErrHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func CatchErr(errCh chan error) error {

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil

}
