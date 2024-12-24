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

package circuitbreakerv1

import (
	"context"
	"github.com/xfali/circuitbreaker"
)

type defaultCircuitBreaker[T any] struct {
	impl circuitbreaker.CircuitBreaker
}

func NewCircuitBreaker[T any](conf *circuitbreaker.Config) *defaultCircuitBreaker[T] {
	ret := &defaultCircuitBreaker[T]{
		impl: circuitbreaker.NewCircuitBreaker(conf),
	}
	return ret
}

func (cb *defaultCircuitBreaker[T]) Close() error {
	return cb.impl.Close()
}

func (cb *defaultCircuitBreaker[T]) GetState() State {
	return cb.impl.GetState()
}

func (cb *defaultCircuitBreaker[T]) Run(ctx context.Context, runnable Runnable[T]) (v T, err error) {
	return cb.Go(ctx, runnable, nil)
}

func (cb *defaultCircuitBreaker[T]) Go(ctx context.Context, runnable, fallback Runnable[T]) (v T, err error) {
	var fallbackFunc circuitbreaker.Runnable
	if fallback != nil {
		fallbackFunc = func() error {
			v, err = fallback(ctx)
			return err
		}
	}
	err = cb.impl.Go(func() error {
		v, err = runnable(ctx)
		return err
	}, fallbackFunc)
	return
}
