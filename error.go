package main

import (
	"context"
	"errors"
	"fmt"
	"log"
)

//TODO
// partCount cant be morethan dataSize
// unknown dltype
//no datacaster slot
// invalid url or filepath
// inaccesable file or dir

// var joinErr func(...error) error = errors.Join

var (
	errJoin = errors.Join
	errIs   = errors.Is

	cancelErr     = context.Canceled
	abortErr      = errors.New("aborted")
	partExceedErr = errors.New("number of parts exceed data size")
)

func toErr(a any) error {
	return fmt.Errorf(fmt.Sprintf("%v", a))
}

func FetchErrHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func CatchErr(errCh chan error, maxErrCount int) error {

	var err error
	errCount := 0
	for catchedErr := range errCh {

		if catchedErr != nil {
			err = errors.Join(err, catchedErr)
			if errIs(catchedErr, cancelErr) || errIs(catchedErr, abortErr) {
				break
			}
		}

		if errCount++; errCount == maxErrCount {
			break
		}
	}

	return err

}
