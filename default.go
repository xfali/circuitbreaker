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

type defaultCircuitBreaker struct {
	logger xlog.Logger

	expiry     time.Time
	expiryLock sync.RWMutex
	fsm        fsm.FSM

	interval time.Duration
	counter  counter.Counter
}

func NewCircuitBreaker(conf *Config) *defaultCircuitBreaker {
	conf = loadDefault(conf)
	ret := &defaultCircuitBreaker{
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

func (cb *defaultCircuitBreaker) Close() error {
	return cb.fsm.Close()
}

func (cb *defaultCircuitBreaker) failureActon(p interface{}) (fsm.State, error) {
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

func (cb *defaultCircuitBreaker) halfActon(p interface{}) (fsm.State, error) {
	cb.setExpiry(time.Time{})
	return StateHalfOpen, nil
}

func (cb *defaultCircuitBreaker) halfFailureActon(p interface{}) (fsm.State, error) {
	cb.setExpiry(time.Now().Add(cb.interval))
	return StateOpen, nil
}

func (cb *defaultCircuitBreaker) halfSuccessAction(p interface{}) (fsm.State, error) {
	return StateClosed, nil
}

func (cb *defaultCircuitBreaker) GetState() State {
	state := cb.fsm.Current()
	return (*state).(State)
}

//func (cb *defaultCircuitBreaker) Name() string {
//	return ""
//}

func (cb *defaultCircuitBreaker) Run(runnable Runnable) (err error) {
	return cb.Go(runnable, nil)
}

func (cb *defaultCircuitBreaker) Go(runnable, fallback Runnable) (err error) {
	cb.acquire()
	if cb.GetState() == StateOpen {
		if fallback != nil {
			return fallback()
		} else {
			return OpenStateErr
		}
	}

	defer func(pe *error) {
		if o := recover(); o != nil {
			if fallback != nil {
				*pe = cb.runAndRelease(fallback, false)
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
	}(&err)

	err = runnable()

	if err != nil {
		if fallback != nil {
			//v, err = safeRun(ctx, fallback)
			err = cb.runAndRelease(fallback, false)
		} else {
			cb.release(false)
		}
	} else {
		cb.release(true)
	}

	return err
}

func (cb *defaultCircuitBreaker) runAndRelease(runnable Runnable, success bool) (err error) {
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
	err = runnable()
	cb.release(success)
	return
}

func safeRun[T any](ctx context.Context, runnable Runnable) (err error) {
	defer func(pe *error) {
		if oo := recover(); oo != nil {
			if oErr, ok := oo.(error); ok {
				*pe = oErr
			} else {
				*pe = errors.New("Execute panic error ")
			}
		}
	}(&err)
	err = runnable()
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

func (cb *defaultCircuitBreaker) setExpiry(t time.Time) {
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
