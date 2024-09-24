package main

import (
	"log"
)

func FetchErrHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func CatchErr(errCh chan error, maxErrCount int) error {

	errCount := 0
	for err := range errCh {
		if err != nil {
			return err
		}
		if errCount++; errCount == maxErrCount {
			break
		}
	}

	return nil

}
