/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockConfig struct {
	version   string
	hostname  string
	database  string
	profiler  string
	id        string
	createdAt string
	library   string
	secret    string
	username  string
	password  string
}

func (m *mockConfig) Version() string               { return m.version }
func (m *mockConfig) Hostname() string              { return m.hostname }
func (m *mockConfig) Database() string              { return m.database }
func (m *mockConfig) Profiler() string              { return m.profiler }
func (m *mockConfig) Id() string                    { return m.id }
func (m *mockConfig) CreatedAt() string             { return m.createdAt }
func (m *mockConfig) Library() string               { return m.library }
func (m *mockConfig) Secret() string                { return m.secret }
func (m *mockConfig) Credentials() (string, string) { return m.username, m.password }

var mockedConfig = mockConfig{
	version:   "1.0.0",
	hostname:  "localhost",
	database:  "test.db",
	id:        "test-id",
	createdAt: "2025-01-01T00:00:00Z",
	library:   "test-library",
	secret:    "1suNCrW7sWlPbU+YCfdGQI7z3ZMo9Ru2GNV4h69QzaM=",
	username:  "test-user",
	password:  "test-password",
}

func Test_NewReturnsObject_DefaultListenAddressAndLoggingOff(t *testing.T) {
	profiler := New(&mockedConfig, false)
	assert.NotNil(t, profiler, "create controller")

	v := reflect.ValueOf(profiler).Elem()
	config := v.FieldByName("config")
	addr := config.FieldByName("ServerAddress")
	assert.Equal(t, "http://pyroscope:4040", addr.String(), "default listen address")

	logger := config.FieldByName("Logger")
	assert.True(t, logger.IsNil(), "logging is off")
}

func Test_NewReturnsObject_SetListenAddressAndLoggingOn(t *testing.T) {
	mockedConfigWithAddress := mockConfig{
		profiler: "https://pyroscope.example.com",
	}
	profiler := New(&mockedConfigWithAddress, true)
	assert.NotNil(t, profiler, "create controller")

	v := reflect.ValueOf(profiler).Elem()
	config := v.FieldByName("config")
	addr := config.FieldByName("ServerAddress")
	assert.Equal(t, "https://pyroscope.example.com", addr.String(), "set listen address")

	logger := config.FieldByName("Logger")
	assert.False(t, logger.IsNil(), "logging is on")
}
