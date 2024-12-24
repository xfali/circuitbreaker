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

import (
	"sync"
	"time"
)

type Rate int8

type bucket struct {
	failure uint64
	success uint64
}

type SlideWindowCounter struct {
	buckets     map[int64]*bucket
	failureRate Rate
	window      time.Duration

	locker sync.Mutex
}

func NewSlideWindowCounter(failureRate Rate, window time.Duration) *SlideWindowCounter {
	if failureRate < 0 || failureRate > 100 {
		panic("failureRate Must be between 0 and 100 ")
	}
	return &SlideWindowCounter{
		buckets:     map[int64]*bucket{},
		window:      window,
		failureRate: failureRate,
	}
}

func (c *SlideWindowCounter) MarkRequest() {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.purgeBucket()
}

func (c *SlideWindowCounter) currentBucket() *bucket {
	t := time.Now().Unix()
	if b, ok := c.buckets[t]; ok {
		return b
	}
	delete(c.buckets, t-int64(c.window/time.Second))
	b := &bucket{}
	c.buckets[t] = b
	return b
}

func (c *SlideWindowCounter) purgeBucket() {
	t := time.Now().Add(-c.window).Unix()
	for k := range c.buckets {
		if k < t {
			delete(c.buckets, k)
		}
	}
}

func (c *SlideWindowCounter) MarkSuccess() {
	c.locker.Lock()
	defer c.locker.Unlock()

	b := c.currentBucket()
	b.success++
}

func (c *SlideWindowCounter) MarkFailure() {
	c.locker.Lock()
	defer c.locker.Unlock()

	b := c.currentBucket()
	b.failure++
}

func (c *SlideWindowCounter) Triggered() bool {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.purgeBucket()
	return c.calc() > c.failureRate
}

func (c *SlideWindowCounter) calc() Rate {
	var failures uint64 = 0
	var successes uint64 = 0
	for _, b := range c.buckets {
		failures += b.failure
		successes += b.success
	}
	return Rate(failures * 100 / (failures + successes))
}

func (c *SlideWindowCounter) Reset() {
	c.locker.Lock()
	defer c.locker.Unlock()

	for k := range c.buckets {
		delete(c.buckets, k)
	}
}

func (c *SlideWindowCounter) Stop() {

}
