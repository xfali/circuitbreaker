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
	"context"
	"errors"
	"github.com/xfali/circuitbreaker/counter"
	"github.com/xfali/fsm"
	"github.com/xfali/xlog"
	"sync"
	"time"
)

const (
	EventTriggered = iota
	EventFailure
	EventSuccess
	EventExpired
)

var (
	UnknownState  = errors.New("Unsupported state ")
	defaultConfig = &Config{
		Interval: 60 * time.Second,
		Counter:  counter.NewCounter(counter.DefaultAllowFunc),
	}
)

type Opt[T any] func(*defaultCircuitBreaker[T])

type defaultCircuitBreaker[T any] struct {
	logger xlog.Logger

	expiry     time.Time
	expiryLock sync.RWMutex

	fsm      fsm.FSM
	interval time.Duration
	counter  counter.Counter
}

type Config struct {
	Interval time.Duration

	Counter counter.Counter
}

func NewCircuitBreaker[T any](conf *Config) *defaultCircuitBreaker[T] {
	conf = loadDefault(conf)
	ret := &defaultCircuitBreaker[T]{
		logger:   xlog.GetLogger(),
		interval: conf.Interval,
		counter:  conf.Counter,
		fsm:      fsm.NewSimpleFSM(),
	}

	ret.fsm.Initial(StateClosed)
	_ = ret.fsm.AddState(StateClosed, EventFailure, ret.failureActon)
	_ = ret.fsm.AddState(StateOpen, EventExpired, ret.halfActon)
	_ = ret.fsm.AddState(StateHalfOpen, EventFailure, ret.halfFailureActon)
	_ = ret.fsm.AddState(StateHalfOpen, EventSuccess, ret.halfSuccessAction)
	return ret
}

func (cb *defaultCircuitBreaker) failureActon(p interface{}) (fsm.State, error) {
	if cb.counter.Triggered() {
		cb.SetExpiry(time.Now().Add(cb.interval))
		err := cb.fsm.SendEvent(EventTriggered, nil)
		if err != nil {
			cb.logger.Errorln(err)
		}
		return StateOpen, nil
	} else {
		return StateClosed, nil
	}
}

func (cb *defaultCircuitBreaker) halfActon(p interface{}) (fsm.State, error) {
	cb.SetExpiry(time.Time{})
	return StateHalfOpen, nil
}

func (cb *defaultCircuitBreaker) halfFailureActon(p interface{}) (fsm.State, error) {
	cb.SetExpiry(time.Now().Add(cb.interval))
	return StateOpen, nil
}

func (cb *defaultCircuitBreaker) halfSuccessAction(p interface{}) (fsm.State, error) {
	return StateClosed, nil
}

func (cb *defaultCircuitBreaker) GetState() State {
	state := cb.fsm.Current()
	return (*state).(State)
}

func (cb *defaultCircuitBreaker) Name() string {
	return ""
}

func (cb *defaultCircuitBreaker[T]) Execute(ctx context.Context, runnable Runnable[T]) (v T, err error) {
	return cb.ExecuteWithFallback(ctx, runnable, nil)
}

func (cb *defaultCircuitBreaker[T]) ExecuteWithFallback(ctx context.Context, runnable, fallback Runnable[T]) (v T, err error) {
	cb.acquire()
	if cb.GetState() == StateOpen {
		if fallback != nil {
			return safeRun(ctx, fallback)
		}
	}

	defer func(pv *T, pe *error) {
		if o := recover(); o != nil {
			if fallback != nil {
				*pv, *pe = safeRun(ctx, fallback)
				cb.release(false)
			} else {
				if oErr, ok := o.(error); ok {
					*pe = oErr
				} else {
					*pe = errors.New("Execute panic error ")
				}
				cb.release(false)
				panic(o)
			}
		}
	}(&v, &err)

	v, err = runnable(ctx)

	cb.release(err == nil)
	return v, err
}

func safeRun[T](ctx context.Context, runnable Runnable[T]) (v T, err error) {
	defer func(pe *error) {
		if oo := recover(); oo != nil {
			if oErr, ok := oo.(error); ok {
				*pe = oErr
			} else {
				*pe = errors.New("Execute panic error ")
			}
		}
	}(&err)
	v, err = runnable(ctx)
	return
}

func (cb *defaultCircuitBreaker) isExpired(now time.Time) bool {
	cb.expiryLock.RLock()
	defer cb.expiryLock.RUnlock()
	if !cb.expiry.IsZero() && cb.expiry.Before(now) {
		return true
	}
	return false
}

func (cb *defaultCircuitBreaker) SetExpiry(t time.Time) {
	cb.expiryLock.Lock()
	defer cb.expiryLock.Unlock()

	cb.expiry = t
}

func (cb *defaultCircuitBreaker) acquire() {
	cb.counter.MarkRequest()
	now := time.Now()
	if cb.isExpired(now) {
		err := cb.fsm.SendEvent(EventExpired, now)
		if err != nil {
			cb.logger.Errorln(err)
		}
	}
}

func (cb *defaultCircuitBreaker) release(success bool) {
	if success {
		cb.counter.MarkSuccess()
		err := cb.fsm.SendEvent(EventSuccess, nil)
		if err != nil {
			cb.logger.Errorln(err)
		}
	} else {
		cb.counter.MarkFailure()
		err := cb.fsm.SendEvent(EventFailure, nil)
		if err != nil {
			cb.logger.Errorln(err)
		}
	}
}

func loadDefault(conf *Config) *Config {
	if conf == nil {
		return defaultConfig
	}

	if conf.Counter == nil {
		conf.Counter = defaultConfig.Counter
	}

	if conf.Interval == 0 {
		conf.Interval = defaultConfig.Interval
	}
	return conf
}
