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

import "context"

type State = int32

const (
	StateClosed   State = 0
	StateOpen     State = 1
	StateHalfOpen State = 2
)

type Runnable[T any] func(ctx context.Context) (T, error)

type CircuitBreaker[T any] interface {
	GetState() State

	Execute(ctx context.Context, runnable Runnable[T]) (T, error)

	ExecuteWithFallback(ctx context.Context, runnable, fallback Runnable[T]) (T, error)
}
