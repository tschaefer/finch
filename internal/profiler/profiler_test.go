/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tschaefer/finch/internal/config"
)

func Test_NewReturnsObjectWithDefaultListenAddressAndLoggingOff(t *testing.T) {
	cfg := config.NewFromData(&config.Data{Profiler: ""}, "")

	profiler := New(cfg, false)
	assert.NotNil(t, profiler, "create controller")

	v := reflect.ValueOf(profiler).Elem()
	config := v.FieldByName("config")
	addr := config.FieldByName("ServerAddress")
	assert.Equal(t, "http://pyroscope:4040", addr.String(), "default listen address")

	logger := config.FieldByName("Logger")
	assert.True(t, logger.IsNil(), "logging is off")
}

func Test_NewReturnsObjectSetsListenAddressAndLoggingOn(t *testing.T) {
	cfg := config.NewFromData(&config.Data{Profiler: "https://pyroscope.example.com"}, "")

	profiler := New(cfg, true)
	assert.NotNil(t, profiler, "create controller")

	v := reflect.ValueOf(profiler).Elem()
	config := v.FieldByName("config")
	addr := config.FieldByName("ServerAddress")
	assert.Equal(t, "https://pyroscope.example.com", addr.String(), "set listen address")

	logger := config.FieldByName("Logger")
	assert.False(t, logger.IsNil(), "logging is on")
}
