package main

import (
	"errors"
	"fmt"
)

var (
	JoinErr = errors.Join
	IsErr   = errors.Is
	NewErr  = fmt.Errorf

	ErrCancel     = NewErr("canceled") //context.Canceled
	ErrAbort      = NewErr("aborted")
	ErrPartExceed = NewErr("partition size or total count exceeds the data size")
	ErrFileURL    = NewErr("inaccessible file or invalid URL")
	ErrDLType     = NewErr("unknown download type")
	ErrExhaust    = NewErr("cache resource exhausted")
	ErrArgs       = NewErr("invalid argument")
	ErrPartLimit  = NewErr("exceeds partition limit")
)

func ToErr(a any) error {
	return fmt.Errorf(fmt.Sprintf("%v", a))
}

func CatchErr(errCh chan error, maxErrCount int) error {

	var err error
	errCount := 0
	for catchedErr := range errCh {

		if catchedErr != nil {
			err = errors.Join(err, catchedErr)
			if IsErr(catchedErr, ErrCancel) || IsErr(catchedErr, ErrAbort) {
				break
			}
		}

		if errCount++; errCount == maxErrCount {
			break
		}
	}

	return err

}
