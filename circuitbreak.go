/*
 * Copyright (C) 2024, Xiongfa Li.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package circuitbreaker

import (
	"io"
)

type State int32

const (
	StateClosed   State = 0
	StateOpen     State = 1
	StateHalfOpen State = 2
)

var (
	stateNameMap = map[State]string{
		StateClosed:   "CLOSED",
		StateOpen:     "OPEN",
		StateHalfOpen: "HALF_OPEN",
	}
)

func (s State) String() string {
	return stateNameMap[s]
}

type Runnable func() error

type CircuitBreaker interface {
	GetState() State

	Run(runnable Runnable) error

	Go(runnable, fallback Runnable) error

	io.Closer
}
