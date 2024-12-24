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

package tests

import (
	"github.com/xfali/circuitbreaker"
	"github.com/xfali/circuitbreaker/counter"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	c := counter.NewCounter(nil)
	c.SetCheckFunc(func(data counter.Data) bool {
		v := data.ConsecutiveFailures == 2
		if v {
			c.Reset()
		}
		return v
	})
	cb := circuitbreaker.NewCircuitBreaker(&circuitbreaker.Config{
		Interval: 3 * time.Second,
		Counter:  c,
	})
	defer cb.Close()

	d := &data{}

	t.Run("Execute", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			err := cb.Run(d.test)
			if err != nil {
				t.Log(err)
			} else {
				t.Log(d.s)
			}
			time.Sleep(500 * time.Millisecond)
		}
	})
	t.Run("ExecuteWithFallback", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			err := cb.Go(d.test, d.fallback)
			if err != nil {
				t.Log(err)
			} else {
				t.Log(d.s)
			}
			time.Sleep(500 * time.Millisecond)
		}
	})
	t.Run("Execute panic", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			func() {
				defer func() {
					if o := recover(); o != nil {
						t.Log(o)
					}
				}()
				err := cb.Run(d.testPanic)
				if err != nil {
					t.Log(err)
				} else {
					t.Log(d.s)
				}
				time.Sleep(500 * time.Millisecond)
			}()
		}
	})
	t.Run("ExecuteWithFallback panic", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			func() {
				defer func() {
					if o := recover(); o != nil {
						t.Log(o)
					}
				}()
				err := cb.Go(d.testPanic, d.fallbackPanic)
				if err != nil {
					t.Log(err)
				} else {
					t.Log(d.s)
				}
				time.Sleep(500 * time.Millisecond)
			}()
		}
	})
}

var testRand = rand.New(rand.NewSource(time.Now().UnixNano()))

type data struct {
	s string
}

func (d *data) test() error {
	ret := testRand.Int() % 2
	if ret == 1 {
		return io.EOF
	}
	d.s = "[Normal] test"
	return nil
}

func (d *data) fallback() error {
	d.s = "[Fallback]"
	return nil
}

func (d *data) testPanic() error {
	ret := testRand.Int() % 2
	if ret == 1 {
		panic("[Panic] from testPanic")
	}
	d.s = "[Normal] testPanic"
	return nil
}

func (d *data) fallbackPanic() error {
	ret := testRand.Int() % 2
	if ret == 1 {
		panic("[Panic] from fallbackPanic")
	}
	d.s = "[Fallback] from fallbackPanic"
	return nil
}
