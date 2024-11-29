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
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type FlowControl struct {
	WG      *sync.WaitGroup
	Limiter chan struct{}
}

var mtx = &sync.Mutex{}

func Interrupt() <-chan os.Signal {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	return sigCh

}

func NewFlowControl(limit int) *FlowControl {

	return &FlowControl{
		WG:      &sync.WaitGroup{},
		Limiter: make(chan struct{}, limit),
	}

}

func (fc *FlowControl) Acquire() {
	fc.Limiter <- struct{}{}
}

func (fc *FlowControl) Release() {
	<-fc.Limiter
}
