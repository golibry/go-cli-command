package cli

import (
	"bytes"
	"context"
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

type MockContextCommand struct {
	MockCommand
	ctx           context.Context
	execContext   func(ctx context.Context, writer io.Writer) error
	execWasCalled bool
}

func (m *MockContextCommand) Exec(writer io.Writer) error {
	m.execWasCalled = true
	return m.MockCommand.Exec(writer)
}

func (m *MockContextCommand) ExecContext(ctx context.Context, writer io.Writer) error {
	m.ctx = ctx
	if m.execContext != nil {
		return m.execContext(ctx, writer)
	}
	return nil
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
	if !errors.Is(err, ErrCommandAlreadyRegistered) {
		t.Errorf("Register() error = %v, want ErrCommandAlreadyRegistered", err)
	}
}

func TestItCanRegisterCommandUsingZeroValueRegistry(t *testing.T) {
	var registry CommandsRegistry
	cmd := &MockCommand{id: "test-cmd", description: "Test command"}

	err := registry.Register(cmd)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}

	if _, exists := registry.Command("test-cmd"); !exists {
		t.Error("Command() exists = false, want true")
	}
}

func TestRegisterRejectsInvalidCommands(t *testing.T) {
	tests := []struct {
		name    string
		cmd     Command
		wantErr error
	}{
		{
			name:    "nil command",
			cmd:     nil,
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "typed nil command",
			cmd:     (*MockCommand)(nil),
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "empty command id",
			cmd:     &MockCommand{id: "", description: "Empty command"},
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "whitespace command id",
			cmd:     &MockCommand{id: "   ", description: "Whitespace command"},
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "leading whitespace command id",
			cmd:     &MockCommand{id: " test-cmd", description: "Whitespace command"},
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "trailing whitespace command id",
			cmd:     &MockCommand{id: "test-cmd ", description: "Whitespace command"},
			wantErr: ErrInvalidCommand,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				registry := NewCommandsRegistry()

				err := registry.Register(tt.cmd)
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Register() error = %v, want %v", err, tt.wantErr)
				}
			},
		)
	}
}

func TestRegisterRejectsNilRegistry(t *testing.T) {
	var registry *CommandsRegistry

	err := registry.Register(&MockCommand{id: "test-cmd", description: "Test command"})
	if !errors.Is(err, ErrInvalidCommandsRegistry) {
		t.Errorf("Register() error = %v, want ErrInvalidCommandsRegistry", err)
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

func TestRegistryCanReturnCommandsOrderedById(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	_ = registry.Register(&MockCommand{id: "zebra", description: "Zebra command"})
	_ = registry.Register(&MockCommand{id: "alpha", description: "Alpha command"})
	_ = registry.Register(&MockCommand{id: "middle", description: "Middle command"})

	commands := registry.OrderedCommands()
	if len(commands) != 3 {
		t.Fatalf("OrderedCommands() returned %d commands, want 3", len(commands))
	}

	wantIds := []string{"alpha", "middle", "zebra"}
	for i, wantId := range wantIds {
		if commands[i].Id() != wantId {
			t.Errorf("OrderedCommands()[%d].Id() = %s, want %s", i, commands[i].Id(), wantId)
		}
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
				err := runCommand(context.Background(), tt.cmd, tt.args, &buf)

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

func TestRunContextPassesContextToContextCommand(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	type contextKey string
	key := contextKey("request-id")
	ctx := context.WithValue(context.Background(), key, "abc-123")
	contextCmd := &MockContextCommand{
		MockCommand: MockCommand{
			id:          "context-cmd",
			description: "Context command",
		},
		execContext: func(ctx context.Context, writer io.Writer) error {
			_, _ = fmt.Fprint(writer, ctx.Value(key))
			return nil
		},
	}
	_ = registry.Register(contextCmd)

	var buf bytes.Buffer
	result := RunContext(ctx, []string{"context-cmd"}, &registry, &buf)

	if result.ExitCode != StatusOk {
		t.Fatalf("RunContext() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}
	if contextCmd.ctx != ctx {
		t.Error("RunContext() did not pass provided context to ExecContext")
	}
	if contextCmd.execWasCalled {
		t.Error("RunContext() called Exec on a ContextCommand")
	}
	if buf.String() != "abc-123" {
		t.Errorf("RunContext() output = %q, want abc-123", buf.String())
	}
}

func TestRunContextUsesExecForRegularCommand(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	executed := false
	cmd := &MockCommand{
		id:          "regular-cmd",
		description: "Regular command",
		execFunc: func(writer io.Writer) error {
			executed = true
			_, _ = fmt.Fprint(writer, "regular")
			return nil
		},
	}
	_ = registry.Register(cmd)

	var buf bytes.Buffer
	result := RunContext(context.Background(), []string{"regular-cmd"}, &registry, &buf)

	if result.ExitCode != StatusOk {
		t.Fatalf("RunContext() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}
	if !executed {
		t.Error("RunContext() did not call Exec for a regular command")
	}
	if buf.String() != "regular" {
		t.Errorf("RunContext() output = %q, want regular", buf.String())
	}
}

func TestRunContextUsesBackgroundWhenContextIsNil(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	contextCmd := &MockContextCommand{
		MockCommand: MockCommand{
			id:          "context-cmd",
			description: "Context command",
		},
	}
	_ = registry.Register(contextCmd)

	var buf bytes.Buffer
	result := RunContext(nil, []string{"context-cmd"}, &registry, &buf)

	if result.ExitCode != StatusOk {
		t.Fatalf("RunContext() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}
	if contextCmd.ctx == nil {
		t.Error("RunContext() passed nil context to ExecContext")
	}
}

func TestItCanRunCliAppWithoutExitingProcess(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	testCmd := &MockCommand{
		id:          "test-cmd",
		description: "Test command",
		execFunc: func(writer io.Writer) error {
			_, _ = fmt.Fprint(writer, "Test command executed")
			return nil
		},
	}
	_ = registry.Register(testCmd)

	var buf bytes.Buffer
	result := Run([]string{"test-cmd"}, &registry, &buf)

	if result.CommandID != "test-cmd" {
		t.Errorf("Run() CommandID = %v, want test-cmd", result.CommandID)
	}
	if result.ExitCode != StatusOk {
		t.Errorf("Run() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}
	if result.Err != nil {
		t.Errorf("Run() Err = %v, want nil", result.Err)
	}
	if !strings.Contains(buf.String(), "Test command executed") {
		t.Errorf("Run() output = %v, want command output", buf.String())
	}
}

func TestExitCodeFromError(t *testing.T) {
	plainErr := errors.New("plain error")

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "nil error",
			err:  nil,
			want: StatusOk,
		},
		{
			name: "plain error",
			err:  plainErr,
			want: StatusErr,
		},
		{
			name: "custom exit code",
			err:  WithExitCode(plainErr, 7),
			want: 7,
		},
		{
			name: "zero custom exit code falls back to error status",
			err:  WithExitCode(plainErr, StatusOk),
			want: StatusErr,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := ExitCodeFromError(tt.err)
				if got != tt.want {
					t.Errorf("ExitCodeFromError() = %d, want %d", got, tt.want)
				}
			},
		)
	}
}

func TestWithExitCodeWrapsError(t *testing.T) {
	baseErr := errors.New("base error")
	err := WithExitCode(baseErr, 9)

	if !errors.Is(err, baseErr) {
		t.Errorf("WithExitCode() error = %v, want to wrap base error", err)
	}

	var exitCoder ExitCoder
	if !errors.As(err, &exitCoder) {
		t.Fatalf("WithExitCode() error does not implement ExitCoder")
	}
	if exitCoder.ExitCode() != 9 {
		t.Errorf("ExitCode() = %d, want 9", exitCoder.ExitCode())
	}
}

func TestRunUsesCustomExitCodeFromCommandError(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	_ = registry.Register(
		&MockCommand{
			id:          "custom-exit",
			description: "Custom exit command",
			execFunc: func(writer io.Writer) error {
				return WithExitCode(errors.New("custom failure"), 12)
			},
		},
	)

	var buf bytes.Buffer
	result := Run([]string{"custom-exit"}, &registry, &buf)

	if result.ExitCode != 12 {
		t.Errorf("Run() ExitCode = %v, want 12", result.ExitCode)
	}
	if result.Err == nil {
		t.Error("Run() Err = nil, want error")
	}
	if !strings.Contains(buf.String(), "custom failure") {
		t.Errorf("Run() output should contain custom failure, got %v", buf.String())
	}
}

func TestBootstrapUsesCustomExitCodeFromCommandError(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	_ = registry.Register(
		&MockCommand{
			id:          "custom-exit",
			description: "Custom exit command",
			execFunc: func(writer io.Writer) error {
				return WithExitCode(errors.New("custom failure"), 23)
			},
		},
	)

	var buf bytes.Buffer
	exitCode := -1
	Bootstrap(
		[]string{"custom-exit"},
		&registry,
		&buf,
		func(code int) { exitCode = code },
	)

	if exitCode != 23 {
		t.Errorf("Bootstrap() exitCode = %v, want 23", exitCode)
	}
}

func TestRunReturnsErrorResultForUnknownCommand(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}

	var buf bytes.Buffer
	result := Run([]string{"missing-cmd"}, &registry, &buf)

	if result.CommandID != "missing-cmd" {
		t.Errorf("Run() CommandID = %v, want missing-cmd", result.CommandID)
	}
	if result.ExitCode != StatusErr {
		t.Errorf("Run() ExitCode = %v, want %v", result.ExitCode, StatusErr)
	}
	if result.Err == nil {
		t.Error("Run() Err = nil, want error")
	}
	if !errors.Is(result.Err, ErrCommandNotFound) {
		t.Errorf("Run() Err = %v, want ErrCommandNotFound", result.Err)
	}
	if !strings.Contains(buf.String(), "does not exist") {
		t.Errorf("Run() output should contain 'does not exist', got %v", buf.String())
	}
}

func TestRunBuildsHelpWithCommandsOrderedById(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	_ = registry.Register(&MockCommand{id: "zebra", description: "Zebra command"})
	_ = registry.Register(&MockCommand{id: "alpha", description: "Alpha command"})

	var buf bytes.Buffer
	result := Run([]string{"help"}, &registry, &buf)

	if result.ExitCode != StatusOk {
		t.Fatalf("Run() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}

	output := buf.String()
	alphaIndex := strings.Index(output, "alpha")
	zebraIndex := strings.Index(output, "zebra")
	if alphaIndex < 0 {
		t.Fatalf("Run() help output does not contain alpha: %s", output)
	}
	if zebraIndex < 0 {
		t.Fatalf("Run() help output does not contain zebra: %s", output)
	}
	if alphaIndex > zebraIndex {
		t.Errorf("Run() help output listed zebra before alpha: %s", output)
	}
}

func TestRunDoesNotRegisterDefaultHelpCommandInRegistry(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	_ = registry.Register(&MockCommand{id: "test-cmd", description: "Test command"})

	var buf bytes.Buffer
	result := Run([]string{"help"}, &registry, &buf)

	if result.ExitCode != StatusOk {
		t.Fatalf("Run() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}

	if _, exists := registry.Command("help"); exists {
		t.Error("Run() registered the default help command in the caller registry")
	}
}

func TestRunBuildsDefaultHelpFromCurrentRegistryState(t *testing.T) {
	registry := CommandsRegistry{commands: make(map[string]Command)}
	_ = registry.Register(&MockCommand{id: "alpha", description: "Alpha command"})

	var buf bytes.Buffer
	result := Run([]string{"help"}, &registry, &buf)
	if result.ExitCode != StatusOk {
		t.Fatalf("Run() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}
	if !strings.Contains(buf.String(), "alpha") {
		t.Fatalf("Run() help output does not contain alpha: %s", buf.String())
	}

	_ = registry.Register(&MockCommand{id: "zebra", description: "Zebra command"})
	buf.Reset()
	result = Run([]string{"help"}, &registry, &buf)

	if result.ExitCode != StatusOk {
		t.Fatalf("Run() ExitCode = %v, want %v", result.ExitCode, StatusOk)
	}
	if !strings.Contains(buf.String(), "zebra") {
		t.Errorf("Run() help output does not contain command registered after first run: %s", buf.String())
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
