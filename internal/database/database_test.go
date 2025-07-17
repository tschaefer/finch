/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package database

import (
	"fmt"
	"strings"
	"testing"
)

type mockConfig struct {
	version   string
	hostname  string
	database  string
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
func (m *mockConfig) Id() string                    { return m.id }
func (m *mockConfig) CreatedAt() string             { return m.createdAt }
func (m *mockConfig) Library() string               { return m.library }
func (m *mockConfig) Secret() string                { return m.secret }
func (m *mockConfig) Credentials() (string, string) { return m.username, m.password }

func Test_NewReturnsError_BadSchema(t *testing.T) {
	mockedConfig := mockConfig{
		database: "psql://user:pass@localhost/dbname",
	}

	_, err := New(&mockedConfig)
	if err == nil {
		t.Error("Expected error for bad schema, got nil")
	}
	expected := "unsupported database scheme:"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func Test_NewReturnsError_PathNotExist(t *testing.T) {
	mockedConfig := mockConfig{
		database: "sqlite:///nonexistent/path/to/database.db",
	}

	_, err := New(&mockedConfig)
	if err == nil {
		t.Error("Expected error for non-existent path, got nil")
	}
	expected := "unable to open database file:"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expected, err.Error())
	}
}

func Test_ConnectionReturnsGormDB(t *testing.T) {
	mockedConfig := mockConfig{
		database: "sqlite://:memory:",
	}

	db, err := New(&mockedConfig)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if db == nil {
		t.Fatal("Expected non-nil Gorm DB instance, got nil")
	}

	connection := db.Connection()
	if connection == nil {
		t.Fatal("Expected non-nil connection, got nil")
	}

	if !strings.Contains(fmt.Sprintf("%T", connection), "gorm.DB") {
		t.Errorf("Expected connection type to be 'gorm.DB', got '%T'", connection)
	}
}

func Test_MigrateSucceeds(t *testing.T) {
	mockedConfig := mockConfig{
		database: "sqlite://:memory:",
	}

	db, err := New(&mockedConfig)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = db.Migrate()
	if err != nil {
		t.Fatalf("Expected no error during migration, got %v", err)
	}
}
