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

package counter

import "sync/atomic"

type SimpleCounter struct {
	RequestNum          uint64
	SuccessNum          uint64
	FailureNum          uint64
	ConsecutiveFailures uint64

	allowFunc func(consecutiveFailures uint64) bool
}

func DefaultAllowFunc(consecutiveFailures uint64) bool {
	return consecutiveFailures > 5
}

func NewCounter(allowFunc func(consecutiveFailures uint64) bool) *SimpleCounter {
	return &SimpleCounter{
		allowFunc: allowFunc,
	}
}

func (c *SimpleCounter) MarkRequest() {
	atomic.AddUint64(&c.RequestNum, 1)
}

func (c *SimpleCounter) MarkSuccess() {
	atomic.AddUint64(&c.SuccessNum, 1)
	atomic.StoreUint64(&c.ConsecutiveFailures, 0)
}

func (c *SimpleCounter) MarkFailure() {
	atomic.AddUint64(&c.FailureNum, 1)
	atomic.AddUint64(&c.ConsecutiveFailures, 1)
}

func (c *SimpleCounter) Triggered() bool {
	return c.allowFunc(atomic.LoadUint64(&c.ConsecutiveFailures))
}

func (c *SimpleCounter) Reset() {
	atomic.StoreUint64(&c.RequestNum, 0)
	atomic.StoreUint64(&c.SuccessNum, 0)
	atomic.StoreUint64(&c.FailureNum, 0)
	atomic.StoreUint64(&c.ConsecutiveFailures, 0)
}

func (c *SimpleCounter) Stop() {

}
