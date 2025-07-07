package cli

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"
)

// MockLockableCommand is a mock command that implements the Command interface
// and will be wrapped by FsLockableCommand for testing
type MockLockableCommand struct {
	CommandWithoutFlags
	id          string
	description string
	execFunc    func() error
	executed    bool
}

func (m *MockLockableCommand) Id() string {
	return m.id
}

func (m *MockLockableCommand) Description() string {
	return m.description
}

func (m *MockLockableCommand) Exec(_ io.Writer) error {
	m.executed = true
	if m.execFunc != nil {
		return m.execFunc()
	}
	return nil
}

func TestLockableCommandHelper_Lock(t *testing.T) {
	// Create a temporary directory for the lock file
	tempDir, err := os.MkdirTemp("", "lockable-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)

	// Create a mock command
	mockCmd := &MockLockableCommand{
		id:          "test-command",
		description: "Test command for locking",
	}

	// Create a lockable command helper with a custom lock name
	lockName := "test-command"
	helper := NewLockableCommandWithLockName(mockCmd, tempDir, lockName)

	// Test acquiring the lock
	locked1, err := helper.Lock()
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	if !locked1 {
		t.Fatalf("Expected to acquire lock, but it was not acquired")
	}

	// Test that trying to acquire the lock again fails
	helper2 := NewLockableCommandWithLockName(mockCmd, tempDir, lockName)
	locked2, err := helper2.Lock()
	if err != nil {
		t.Fatalf("Lock() returned unexpected error: %v", err)
	}
	if locked2 {
		t.Fatalf("Expected lock acquisition to fail, but it succeeded")
	}

	// Release the lock
	_ = helper.Unlock()

	// Note: The lock file might not be immediately removed by the underlying implementation
	// We'll give it a moment to be released
	time.Sleep(10 * time.Millisecond)

	// Test that we can acquire the lock again after releasing it
	locked3, err := helper2.Lock()
	if err != nil {
		t.Fatalf("Failed to acquire lock after it was released: %v", err)
	}
	if !locked3 {
		t.Fatalf("Expected to acquire lock after it was released, but it was not acquired")
	}
	_ = helper2.Unlock()
}

func TestLockableCommandHelper_Exec(t *testing.T) {
	// Create a temporary directory for the lock file
	tempDir, err := os.MkdirTemp("", "lockable-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)

	// Create a mock command
	mockCmd := &MockLockableCommand{
		id:          "test-command",
		description: "Test command for locking",
	}

	// Create a lockable command helper with a custom lock name
	lockName := "test-command"
	helper := NewLockableCommandWithLockName(mockCmd, tempDir, lockName)

	// Test executing the command
	var buf bytes.Buffer
	err = helper.Exec(&buf)
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Verify the command was executed
	if !mockCmd.executed {
		t.Fatalf("Command was not executed")
	}

	// Note: The lock file might not be immediately removed by the underlying implementation
	// We'll give it a moment to be released
	time.Sleep(10 * time.Millisecond)
}

func TestLockableCommandHelper_ConcurrentExecution(t *testing.T) {
	// Create a temporary directory for the lock file
	tempDir, err := os.MkdirTemp("", "lockable-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)

	// Create a mock command that takes some time to execute
	mockCmd := &MockLockableCommand{
		id:          "slow-command",
		description: "Slow command for testing concurrent execution",
		execFunc: func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	// Create a lockable command helper with a custom lock name
	lockName := "slow-command"
	helper := NewLockableCommandWithLockName(mockCmd, tempDir, lockName)

	// Start executing the command in a goroutine
	var err1 error
	var buf1 bytes.Buffer
	done := make(chan bool)
	go func() {
		err1 = helper.Exec(&buf1)
		done <- true
	}()

	// Give the first execution a chance to acquire the lock
	time.Sleep(10 * time.Millisecond)

	// Try to execute the command again immediately
	var err2 error
	var buf2 bytes.Buffer
	helper2 := NewLockableCommandWithLockName(mockCmd, tempDir, lockName)
	err2 = helper2.Exec(&buf2)

	// Wait for the first execution to complete
	<-done

	// The first execution should succeed
	if err1 != nil {
		t.Fatalf("First execution failed: %v", err1)
	}

	// The second execution should fail with a lock error
	if err2 == nil {
		t.Fatalf("Expected second execution to fail, but it succeeded")
	}
}
