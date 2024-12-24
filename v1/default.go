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
	"errors"
	"github.com/xfali/circuitbreaker/counter"
	"github.com/xfali/fsm"
	"github.com/xfali/xlog"
	"sync"
	"time"
)

type Event int8

const (
	EventTriggered Event = iota
	EventFailure
	EventSuccess
	EventExpired
)

var (
	OpenStateErr  = errors.New("Circuit breaker is opened ")
	defaultConfig = &Config{
		Interval: 60 * time.Second,
		Counter:  counter.NewCounter(counter.DefaultAllowFunc),
	}
	eventNameMap = map[Event]string{
		EventTriggered: "EventTriggered",
		EventFailure:   "EventFailure",
		EventSuccess:   "EventSuccess",
		EventExpired:   "EventExpired",
	}
)

func (e Event) String() string {
	return eventNameMap[e]
}

type defaultCircuitBreaker[T any] struct {
	logger xlog.Logger

	expiry     time.Time
	expiryLock sync.RWMutex
	fsm        fsm.FSM

	interval time.Duration
	counter  counter.Counter
}

func NewCircuitBreaker[T any](conf *Config) *defaultCircuitBreaker[T] {
	conf = loadDefault(conf)
	ret := &defaultCircuitBreaker[T]{
		logger:   xlog.GetLogger(),
		interval: conf.Interval,
		counter:  conf.Counter,
	}
	if conf.Settings != nil {
		if s, ok := conf.Settings.(*defaultSettings); ok {
			ret.fsm = s.fsm
		}
	}
	if ret.fsm == nil {
		ret.fsm = fsm.NewSimpleFSM()
		ret.fsm.SetListener(&fsm.DefaultListener{Silent: true})
	}

	ret.fsm.Initial(StateClosed)
	_ = ret.fsm.AddState(StateClosed, EventFailure, ret.failureActon)
	_ = ret.fsm.AddState(StateOpen, EventExpired, ret.halfActon)
	_ = ret.fsm.AddState(StateHalfOpen, EventFailure, ret.halfFailureActon)
	_ = ret.fsm.AddState(StateHalfOpen, EventSuccess, ret.halfSuccessAction)

	err := ret.fsm.Start()
	if err != nil {
		ret.logger.Errorln(err)
	}
	return ret
}

func (cb *defaultCircuitBreaker[T]) Close() error {
	return cb.fsm.Close()
}

func (cb *defaultCircuitBreaker[T]) failureActon(p interface{}) (fsm.State, error) {
	if cb.counter.Triggered() {
		cb.setExpiry(time.Now().Add(cb.interval))
		err := cb.fsm.SendEvent(EventTriggered, nil)
		if err != nil {
			cb.logger.Errorln(err)
		}
		return StateOpen, nil
	} else {
		return StateClosed, nil
	}
}

func (cb *defaultCircuitBreaker[T]) halfActon(p interface{}) (fsm.State, error) {
	cb.setExpiry(time.Time{})
	return StateHalfOpen, nil
}

func (cb *defaultCircuitBreaker[T]) halfFailureActon(p interface{}) (fsm.State, error) {
	cb.setExpiry(time.Now().Add(cb.interval))
	return StateOpen, nil
}

func (cb *defaultCircuitBreaker[T]) halfSuccessAction(p interface{}) (fsm.State, error) {
	return StateClosed, nil
}

func (cb *defaultCircuitBreaker[T]) GetState() State {
	state := cb.fsm.Current()
	return (*state).(State)
}

//func (cb *defaultCircuitBreaker) Name() string {
//	return ""
//}

func (cb *defaultCircuitBreaker[T]) Run(ctx context.Context, runnable Runnable[T]) (v T, err error) {
	return cb.ExecuteWithFallback(ctx, runnable, nil)
}

func (cb *defaultCircuitBreaker[T]) ExecuteWithFallback(ctx context.Context, runnable, fallback Runnable[T]) (v T, err error) {
	cb.acquire()
	if cb.GetState() == StateOpen {
		if fallback != nil {
			return fallback(ctx)
		} else {
			return v, OpenStateErr
		}
	}

	defer func(pv *T, pe *error) {
		if o := recover(); o != nil {
			if fallback != nil {
				*pv, *pe = cb.runAndRelease(ctx, fallback, false)
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

	if err != nil {
		if fallback != nil {
			//v, err = safeRun(ctx, fallback)
			v, err = cb.runAndRelease(ctx, fallback, false)
		} else {
			cb.release(false)
		}
	} else {
		cb.release(true)
	}

	return v, err
}

func (cb *defaultCircuitBreaker[T]) runAndRelease(ctx context.Context, runnable Runnable[T], success bool) (v T, err error) {
	defer func(pe *error) {
		if o := recover(); o != nil {
			if oErr, ok := o.(error); ok {
				*pe = oErr
			} else {
				*pe = errors.New("Execute panic error ")
			}
			cb.release(success)
			panic(o)
		}
	}(&err)
	v, err = runnable(ctx)
	cb.release(success)
	return
}

func safeRun[T any](ctx context.Context, runnable Runnable[T]) (v T, err error) {
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

func (cb *defaultCircuitBreaker[T]) isExpired(now time.Time) bool {
	cb.expiryLock.RLock()
	defer cb.expiryLock.RUnlock()
	if !cb.expiry.IsZero() && cb.expiry.Before(now) {
		return true
	}
	return false
}

func (cb *defaultCircuitBreaker[T]) setExpiry(t time.Time) {
	cb.expiryLock.Lock()
	defer cb.expiryLock.Unlock()

	cb.expiry = t
}

func (cb *defaultCircuitBreaker[T]) acquire() {
	cb.counter.MarkRequest()
	now := time.Now()
	if cb.isExpired(now) {
		err := cb.fsm.SendEvent(EventExpired, now)
		if err != nil {
			cb.logger.Errorln(err)
		}
	}
}

func (cb *defaultCircuitBreaker[T]) release(success bool) {
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

type defaultSettings struct {
	fsm fsm.FSM
}

func NewSettings() *defaultSettings {
	return &defaultSettings{}
}

func (s *defaultSettings) SetStateMachine(fsm fsm.FSM) *defaultSettings {
	s.fsm = fsm
	return s
}

func (s *defaultSettings) Set(key string, value interface{}) Settings {
	if key == "fsm" {
		s.fsm = value.(fsm.FSM)
	}
	return s
}

type defaultOpts struct {
}

var DefaultOpts defaultOpts

func (o defaultOpts) SetFsm(stateMachine fsm.FSM) Opt {
	return func(settings Settings) {
		settings.Set("fsm", stateMachine)
	}
}
