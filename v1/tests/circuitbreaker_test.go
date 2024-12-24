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
	"context"
	"github.com/xfali/circuitbreaker"
	"github.com/xfali/circuitbreaker/counter"
	"github.com/xfali/circuitbreaker/v1"
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cb := circuitbreakerv1.NewCircuitBreaker[string](&circuitbreaker.Config{
		Interval: 3 * time.Second,
		Counter: counter.NewCounter(func(consecutiveFailures uint64) bool {
			return consecutiveFailures == 2
		}),
	})
	defer cb.Close()

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	t.Run("Execute", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			v, err := cb.Run(ctx, test)
			if err != nil {
				t.Log(err)
			} else {
				t.Log(v)
			}
			time.Sleep(500 * time.Millisecond)
		}
	})
	t.Run("ExecuteWithFallback", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			v, err := cb.Go(ctx, test, fallback)
			if err != nil {
				t.Log(err)
			} else {
				t.Log(v)
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
				v, err := cb.Run(ctx, testPanic)
				if err != nil {
					t.Log(err)
				} else {
					t.Log(v)
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
				v, err := cb.Go(ctx, testPanic, fallbackPanic)
				if err != nil {
					t.Log(err)
				} else {
					t.Log(v)
				}
				time.Sleep(500 * time.Millisecond)
			}()
		}
	})
}

var testRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func test(ctx context.Context) (string, error) {
	ret := testRand.Int() % 2
	if ret == 1 {
		return "[Failure] from test", io.EOF
	}
	return "[Normal] test", nil
}

func fallback(ctx context.Context) (string, error) {
	return "[Fallback]", nil
}

func testPanic(ctx context.Context) (string, error) {
	ret := testRand.Int() % 2
	if ret == 1 {
		panic("[Panic] from testPanic")
	}
	return "[Normal] testPanic", nil
}

func fallbackPanic(ctx context.Context) (string, error) {
	ret := testRand.Int() % 2
	if ret == 1 {
		panic("[Panic] from fallbackPanic")
	}
	return "[Fallback] from fallbackPanic", nil
}
