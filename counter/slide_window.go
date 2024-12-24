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

type Rate int8

type SlideWindowCounter struct {
	requests    []uint64
	size        int
	index       int
	count       uint64
	failures    uint64
	failureRate Rate
}

func NewSlideWindowCounter(size int, failureRate Rate) *SlideWindowCounter {
	return &SlideWindowCounter{
		requests:    make([]uint64, size),
		size:        size,
		failureRate: failureRate,
	}
}

func (c *SlideWindowCounter) MarkRequest() {
	//if c.count < c.size {
	//	c.count++
	//} else {
	//	if c.requests[c.index] == 1 {
	//
	//	}
	//}
}

func (c *SlideWindowCounter) MarkSuccess() {
	c.requests[c.index] = 0
	c.index = (c.index + 1) % c.size
}

func (c *SlideWindowCounter) MarkFailure() {
	c.requests[c.index] = 1
	c.failures++
	c.index = (c.index + 1) % c.size
}

func (c *SlideWindowCounter) Triggered() bool {
	return c.calc() > c.failureRate
}

func (c *SlideWindowCounter) calc() Rate {
	if c.count == 0 {
		return 0
	}
	return Rate(c.failures * 100 / c.count)
}

func (c *SlideWindowCounter) Reset() {

}

func (c *SlideWindowCounter) Stop() {

}
