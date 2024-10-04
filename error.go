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
	errNew  = errors.New

	cancelErr     = context.Canceled
	abortErr      = errNew("aborted")
	partExceedErr = errNew("The size of each or the number of parts exceeds the data size.")
	fileURLErr    = errNew("inaccessible file or invalid URL")
	dltypeErr     = errNew("unknown download type")
	exhaustErr    = errNew("cache resource exhausted")
)

func toErr(a any) error {
	return fmt.Errorf(fmt.Sprintf("%v", a))
}

func FetchErrHandle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ErrCatch(errCh chan error, maxErrCount int) error {

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
