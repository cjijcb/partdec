package main

import (
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
	errNew  = fmt.Errorf

	CancelErr     = errNew("canceled") //context.Canceled
	AbortErr      = errNew("aborted")
	PartExceedErr = errNew("The size of each or the number of parts exceeds the data size.")
	FileURLErr    = errNew("inaccessible file or invalid URL")
	DLTypeErr     = errNew("unknown download type")
	ExhaustErr    = errNew("cache resource exhausted")
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
			if errIs(catchedErr, CancelErr) || errIs(catchedErr, AbortErr) {
				break
			}
		}

		if errCount++; errCount == maxErrCount {
			break
		}
	}

	return err

}
