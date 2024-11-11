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
	"sync"
    "os"
    "os/signal"
    "syscall"
)
	

type (

    FlowControl struct {
		WG      *sync.WaitGroup
		Limiter chan struct{}
		Acquire func(chan<- struct{}) <-chan struct{}
		Release func(<-chan struct{})
	}

)


func NewFlowControl(limit int) *FlowControl {

    limiter := make(chan struct{}, limit)
    acq := func(l chan<- struct{}) <-chan struct{} {
        succeed := make(chan struct{})
        l <- struct{}{}
        close(succeed)
        return succeed
    }
    rls := func(l <-chan struct{}) { <-l }

    return &FlowControl{
        WG:      &sync.WaitGroup{},
        Limiter: limiter,
        Acquire: acq,
        Release: rls,
    }
}

func Interrupt() <-chan os.Signal {

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		sigCh <- sig
	}()

	return sigCh
}

