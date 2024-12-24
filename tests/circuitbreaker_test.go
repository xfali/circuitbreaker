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
	"io"
	"math/rand"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker[string](&circuitbreaker.Config{
		Interval: 3 * time.Second,
		Counter: counter.NewCounter(func(consecutiveFailures uint64) bool {
			return consecutiveFailures == 2
		}),
	})
	defer cb.Close()

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	for i := 0; i < 1000; i++ {
		v, err := cb.Execute(ctx, test)
		if err != nil {
			t.Log(err)
		} else {
			t.Log(v)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func test(ctx context.Context) (string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	ret := r.Int() % 2
	if ret == 1 {
		return "1", io.EOF
	}
	return "0", nil
}
