package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"
)

// MockCommand is a simple implementation of the Command interface for testing
type MockCommand struct {
	CommandWithoutFlags
	id          string
	description string
	execFunc    func(writer io.Writer) error
}

func (m *MockCommand) Id() string {
	return m.id
}

func (m *MockCommand) Description() string {
	return m.description
}

func (m *MockCommand) Exec(writer io.Writer) error {
	if m.execFunc != nil {
		return m.execFunc(writer)
	}
	return nil
}

// MockCommandWithFlags is a Command implementation with flagSet for testing
type MockCommandWithFlags struct {
	id          string
	description string
	execFunc    func(writer io.Writer) error
	validateErr error
	flagSet     *flag.FlagSet
}

func (m *MockCommandWithFlags) Id() string {
	return m.id
}

func (m *MockCommandWithFlags) Description() string {
	return m.description
}

func (m *MockCommandWithFlags) Exec(writer io.Writer) error {
	if m.execFunc != nil {
		return m.execFunc(writer)
	}
	return nil
}

func (m *MockCommandWithFlags) DefineFlags(flagSet *flag.FlagSet) {
	m.flagSet = flagSet
	m.flagSet.String("test-flag", "", "A test flag")
}

func (m *MockCommandWithFlags) ValidateFlags() error {
	return m.validateErr
}

func TestItCanParseCmdInput(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCmdName string
		wantCmdArgs []string
	}{
		{
			name:        "empty args",
			args:        []string{},
			wantCmdName: "",
			wantCmdArgs: nil,
		},
		{
			name:        "command only",
			args:        []string{"test-cmd"},
			wantCmdName: "test-cmd",
			wantCmdArgs: []string{},
		},
		{
			name:        "command with args",
			args:        []string{"test-cmd", "arg1", "arg2"},
			wantCmdName: "test-cmd",
			wantCmdArgs: []string{"arg1", "arg2"},
		},
		{
			name:        "with -- prefix and args",
			args:        []string{"--", "test-cmd", "arg1"},
			wantCmdName: "test-cmd",
			wantCmdArgs: []string{"arg1"},
		},
		{
			name:        "with -- prefix and no args",
			args:        []string{"--"},
			wantCmdName: "",
			wantCmdArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				gotCmdName, gotCmdArgs := parseCmdInput(tt.args)
				if gotCmdName != tt.wantCmdName {
					t.Errorf("parseCmdInput() gotCmdName = %v, want %v", gotCmdName, tt.wantCmdName)
				}
				if len(gotCmdArgs) != len(tt.wantCmdArgs) {
					t.Errorf(
						"parseCmdInput() gotCmdArgs length = %v, want %v",
						len(gotCmdArgs),
						len(tt.wantCmdArgs),
					)
				} else {
					for i, arg := range gotCmdArgs {
						if arg != tt.wantCmdArgs[i] {
							t.Errorf(
								"parseCmdInput() gotCmdArgs[%d] = %v, want %v",
								i,
								arg,
								tt.wantCmdArgs[i],
							)
						}
					}
				}
			},
		)
	}
}

func TestItCanRegisterCommandsWithoutDuplicates(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	cmd := &MockCommand{id: "test-cmd", description: "Test command"}

	// Test successful registration
	err := registry.Register(cmd)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}

	// Test duplicate registration
	err = registry.Register(cmd)
	if err == nil {
		t.Error("Register() error = nil, want error for duplicate command")
	}
}

func TestItCanRegisterMultipleCommandsAndExposeACopyOfThem(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	cmd1 := &MockCommand{id: "cmd1", description: "Command 1"}
	cmd2 := &MockCommand{id: "cmd2", description: "Command 2"}

	_ = registry.Register(cmd1)
	_ = registry.Register(cmd2)

	commands := registry.Commands()
	if len(commands) != 2 {
		t.Errorf("Commands() returned %d commands, want 2", len(commands))
	}

	// Verify that modifying the returned map doesn't affect the registry
	delete(commands, "cmd1")
	if _, exists := registry.Command("cmd1"); !exists {
		t.Error("Commands() should return a copy, but modification affected original")
	}
}

func TestRegistryAllowsToFindACommandById(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	cmd := &MockCommand{id: "test-cmd", description: "Test command"}
	_ = registry.Register(cmd)

	// Test finding existing command
	foundCmd, exists := registry.Command("test-cmd")
	if !exists {
		t.Error("Command() exists = false, want true")
	}
	if foundCmd.Id() != "test-cmd" {
		t.Errorf("Command() returned command with ID = %s, want test-cmd", foundCmd.Id())
	}

	// Test finding non-existent command
	_, exists = registry.Command("non-existent")
	if exists {
		t.Error("Command() exists = true, want false for non-existent command")
	}
}

func TestItCanRunCommand(t *testing.T) {
	tests := []struct {
		name       string
		cmd        Command
		args       []string
		wantOutput string
		wantErr    bool
	}{
		{
			name: "successful command",
			cmd: &MockCommand{
				id:          "test-cmd",
				description: "Test command",
				execFunc: func(writer io.Writer) error {
					_, _ = fmt.Fprint(writer, "Command executed successfully")
					return nil
				},
			},
			args:       []string{},
			wantOutput: "Command executed successfully",
			wantErr:    false,
		},
		{
			name: "command with error",
			cmd: &MockCommand{
				id:          "error-cmd",
				description: "Error command",
				execFunc: func(writer io.Writer) error {
					return errors.New("command execution failed")
				},
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "command with flagSet validation error",
			cmd: &MockCommandWithFlags{
				id:          "flag-error-cmd",
				description: "Flag error command",
				validateErr: errors.New("flag validation failed"),
			},
			args:    []string{"--test-flag", "value"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				var buf bytes.Buffer
				err := runCommand(tt.cmd, tt.args, &buf)

				if (err != nil) != tt.wantErr {
					t.Errorf("runCommand() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if !tt.wantErr && !strings.Contains(buf.String(), tt.wantOutput) {
					t.Errorf(
						"runCommand() output = %v, want to contain %v",
						buf.String(),
						tt.wantOutput,
					)
				}
			},
		)
	}
}

// TestBootstrap tests the Bootstrap function
func TestItCanBootstrapCliApp(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}

	// Register a test command
	testCmd := &MockCommand{
		id:          "test-cmd",
		description: "Test command",
		execFunc: func(writer io.Writer) error {
			_, _ = fmt.Fprint(writer, "Test command executed")
			return nil
		},
	}
	_ = registry.Register(testCmd)

	// Test successful command execution
	var buf bytes.Buffer
	exitCode := -1
	Bootstrap(
		[]string{"test-cmd"},
		&registry,
		&buf,
		func(code int) { exitCode = code },
	)

	if exitCode != StatusOk {
		t.Errorf("Bootstrap() exitCode = %v, want %v", exitCode, StatusOk)
	}

	// Test command not found
	buf.Reset()
	exitCode = -1
	Bootstrap(
		[]string{"non-existent-cmd"},
		&registry,
		&buf,
		func(code int) { exitCode = code },
	)

	if exitCode != StatusErr {
		t.Errorf("Bootstrap() exitCode = %v, want %v", exitCode, StatusErr)
	}
	if !strings.Contains(buf.String(), "does not exist") {
		t.Errorf("Bootstrap() output should contain 'does not exist', got %v", buf.String())
	}
}
