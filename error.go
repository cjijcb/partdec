/*
Copyright 2024 Carlo Jay I. Jacaba

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package partdec

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
	ErrPartExceed = NewErr("output file size or total count exceeds the source file size")
	ErrFileURL    = NewErr("inaccessible file or invalid URL")
	ErrDLType     = NewErr("unknown download type")
	ErrExhaust    = NewErr("resource exhausted")
	ErrArgs       = NewErr("invalid argument")
	ErrPartLimit  = NewErr("exceeds output file count limit")
	ErrVer        = NewErr("flag: version requested")
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
