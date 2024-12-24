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
	"github.com/xfali/circuitbreaker/counter"
	"time"
)

type Opt func(Settings)

type Settings interface {
	Set(key string, value interface{}) Settings
}

type Config struct {
	Interval time.Duration

	Counter counter.Counter

	Settings Settings
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
