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
	"context"
	"errors"
	"fmt"
)

var (
	JoinErr = errors.Join
	IsErr   = errors.Is
	NewErr  = fmt.Errorf

	ErrCancel     = NewErr("canceled")
	ErrAbort      = NewErr("aborted")
	ErrPartExceed = NewErr("part total count or size exceeds the source file size")
	ErrFileURL    = NewErr("inaccessible file or invalid URI")
	ErrDLType     = NewErr("unknown download type")
	ErrExhaust    = NewErr("resource exhausted")
	ErrArgs       = NewErr("invalid argument")
	ErrParse      = NewErr("parse error")
	ErrPartLimit  = NewErr("exceeds output file count limit")
	ErrVer        = NewErr("version requested")
)

func catchErr(errCh chan error, maxErrCount int) (err error) {

	errCount := 0
	for catched := range errCh {
		if catched != nil {

			if IsErr(catched, context.Canceled) {
				err = JoinErr(err, ErrCancel)
				break
			}

			if IsErr(catched, ErrAbort) {
				err = JoinErr(err, ErrAbort)
				break
			}

			err = JoinErr(err, catched)
		}
		if errCount++; errCount >= maxErrCount {
			break
		}
	}

	return err

}
