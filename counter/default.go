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

type Data struct {
	RequestNum          uint64
	SuccessNum          uint64
	FailureNum          uint64
	ConsecutiveFailures uint64
}

type simpleCounter struct {
	requestNum          uint64
	successNum          uint64
	failureNum          uint64
	consecutiveFailures uint64

	checkFunc func(data Data) bool
}

func DefaultAllowFunc(data Data) bool {
	return data.ConsecutiveFailures > 5
}

func NewCounter(checkFunc func(data Data) bool) *simpleCounter {
	return &simpleCounter{
		checkFunc: checkFunc,
	}
}

func (c *simpleCounter) SetCheckFunc(checkFunc func(data Data) bool) {
	c.checkFunc = checkFunc
}

func (c *simpleCounter) MarkRequest() {
	atomic.AddUint64(&c.requestNum, 1)
}

func (c *simpleCounter) MarkSuccess() {
	atomic.AddUint64(&c.successNum, 1)
	atomic.StoreUint64(&c.consecutiveFailures, 0)
}

func (c *simpleCounter) MarkFailure() {
	atomic.AddUint64(&c.failureNum, 1)
	atomic.AddUint64(&c.consecutiveFailures, 1)
}

func (c *simpleCounter) Triggered() bool {
	d := Data{
		RequestNum:          atomic.LoadUint64(&c.requestNum),
		SuccessNum:          atomic.LoadUint64(&c.successNum),
		FailureNum:          atomic.LoadUint64(&c.failureNum),
		ConsecutiveFailures: atomic.LoadUint64(&c.consecutiveFailures),
	}
	return c.checkFunc(d)
}

func (c *simpleCounter) Reset() {
	atomic.StoreUint64(&c.requestNum, 0)
	atomic.StoreUint64(&c.successNum, 0)
	atomic.StoreUint64(&c.failureNum, 0)
	atomic.StoreUint64(&c.consecutiveFailures, 0)
}

func (c *simpleCounter) Stop() {

}
