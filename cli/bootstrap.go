package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
)

const StatusOk = 0
const StatusErr = 1

var (
	ErrCommandAlreadyRegistered = errors.New("command already registered")
	ErrCommandNotFound          = errors.New("command not found")
	ErrInvalidCommand           = errors.New("invalid command")
	ErrInvalidCommandsRegistry  = errors.New("invalid commands registry")
)

// ExitCoder can be implemented by errors that should control the process exit code.
type ExitCoder interface {
	ExitCode() int
}

// ErrorWithExitCode wraps an error with a custom process exit code.
type ErrorWithExitCode struct {
	Err  error
	Code int
}

func (e *ErrorWithExitCode) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ErrorWithExitCode) Unwrap() error {
	return e.Err
}

func (e *ErrorWithExitCode) ExitCode() int {
	return e.Code
}

// WithExitCode wraps err with a custom process exit code.
func WithExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &ErrorWithExitCode{Err: err, Code: code}
}

// ExitCodeFromError returns StatusOk for nil, a custom error exit code when provided,
// or StatusErr for ordinary errors.
func ExitCodeFromError(err error) int {
	if err == nil {
		return StatusOk
	}

	var exitCoder ExitCoder
	if errors.As(err, &exitCoder) && exitCoder.ExitCode() > StatusOk {
		return exitCoder.ExitCode()
	}

	return StatusErr
}

// RunResult describes the outcome of processing one CLI invocation.
type RunResult struct {
	CommandID string
	ExitCode  int
	Err       error
}

// Command interface defines the methods that a command must implement
type Command interface {
	Id() string
	Description() string
	Exec(stdWriter io.Writer) error
	DefineFlags(flagSet *flag.FlagSet)
	ValidateFlags() error
}

// ContextCommand can be implemented by commands that need cancellation or deadlines.
type ContextCommand interface {
	Command
	ExecContext(ctx context.Context, stdWriter io.Writer) error
}

type LockableCommand interface {
	Command
	Lock() (bool, error)
	Unlock() error
}

type CommandWithoutFlags struct{}

func (*CommandWithoutFlags) DefineFlags(*flag.FlagSet) {}
func (*CommandWithoutFlags) ValidateFlags() error {
	return nil
}

// setupFlagSet creates and configures a flag.FlagSet for the given command
func setupFlagSet(cmd Command, outputWriter io.Writer) *flag.FlagSet {
	flagSet := flag.NewFlagSet(cmd.Id(), flag.ContinueOnError)
	flagSet.Usage = func() {
		_, _ = fmt.Fprintf(outputWriter, "Usage of %s:\n", cmd.Id())
		flagSet.PrintDefaults()
	}

	return flagSet
}

// runCommand runs the given command with the provided arguments
func runCommand(
	ctx context.Context,
	cmd Command,
	args []string,
	outputWriter io.Writer,
) (cmdErr error) {
	defer func() {
		if err := recover(); err != nil {
			switch v := err.(type) {
			case string:
				err = errors.New(v)
			case error:
				err = v
			default:
				err = errors.New(fmt.Sprint(v))
			}
			cmdErr = err.(error)
		}
	}()

	// Setup flag set for the command
	flagSet := setupFlagSet(cmd, outputWriter)
	flagSet.SetOutput(outputWriter)
	cmd.DefineFlags(flagSet)

	// Parse flagSet
	if !flagSet.Parsed() {
		if cmdErr = flagSet.Parse(args); cmdErr != nil {
			return cmdErr
		}
	}

	cmdErr = cmd.ValidateFlags()
	if cmdErr != nil {
		return cmdErr
	}

	// Execute the command
	if contextCmd, ok := cmd.(ContextCommand); ok {
		cmdErr = contextCmd.ExecContext(ctx, outputWriter)
	} else {
		cmdErr = cmd.Exec(outputWriter)
	}
	if cmdErr != nil {
		return cmdErr
	}

	return cmdErr
}

// parseCmdInput parses the command name and arguments from the input args
func parseCmdInput(args []string) (cmdName string, cmdArgs []string) {
	if len(args) == 0 {
		return
	} else if args[0] == "--" {
		args = args[1:]
	}

	if len(args) != 0 {
		cmdName = strings.TrimSpace(args[0])
		cmdArgs = args[1:]
	}

	return
}

// CommandsRegistry holds all registered commands
type CommandsRegistry struct {
	commands map[string]Command
}

func NewCommandsRegistry() *CommandsRegistry {
	return &CommandsRegistry{make(map[string]Command)}
}

func isNilCommand(cmd Command) bool {
	if cmd == nil {
		return true
	}

	cmdValue := reflect.ValueOf(cmd)
	switch cmdValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return cmdValue.IsNil()
	default:
		return false
	}
}

// Register adds a command to the registry
func (registry *CommandsRegistry) Register(cmd Command) error {
	if registry == nil {
		return fmt.Errorf("%w: registry cannot be nil", ErrInvalidCommandsRegistry)
	}

	if isNilCommand(cmd) {
		return fmt.Errorf("%w: command cannot be nil", ErrInvalidCommand)
	}

	cmdId := cmd.Id()
	trimmedCmdId := strings.TrimSpace(cmdId)
	if trimmedCmdId == "" {
		return fmt.Errorf("%w: command id cannot be empty", ErrInvalidCommand)
	}
	if trimmedCmdId != cmdId {
		return fmt.Errorf(
			"%w: command id %q must not contain leading or trailing whitespace",
			ErrInvalidCommand,
			cmdId,
		)
	}

	if registry.commands == nil {
		registry.commands = make(map[string]Command)
	}

	if _, exists := registry.commands[cmdId]; exists {
		return fmt.Errorf("%w: command %q", ErrCommandAlreadyRegistered, cmdId)
	}
	registry.commands[cmdId] = cmd
	return nil
}

// Commands return a copy of all registered commands
func (registry *CommandsRegistry) Commands() map[string]Command {
	if registry == nil {
		return map[string]Command{}
	}

	cmdCopy := make(map[string]Command, len(registry.commands))
	for name, cmd := range registry.commands {
		cmdCopy[name] = cmd
	}
	return cmdCopy
}

// OrderedCommands returns all registered commands ordered by command ID.
func (registry *CommandsRegistry) OrderedCommands() []Command {
	if registry == nil {
		return []Command{}
	}

	commands := make([]Command, 0, len(registry.commands))
	for _, cmd := range registry.commands {
		commands = append(commands, cmd)
	}

	sort.Slice(
		commands, func(i, j int) bool {
			return commands[i].Id() < commands[j].Id()
		},
	)

	return commands
}

// Command returns a command by its ID
func (registry *CommandsRegistry) Command(id string) (Command, bool) {
	if registry == nil {
		return nil, false
	}

	cmd, ok := registry.commands[id]
	return cmd, ok
}

func newDefaultHelpCommand(availableCommands *CommandsRegistry) *HelpCommand {
	return &HelpCommand{
		CommandWithoutFlags{},
		availableCommands.OrderedCommands(),
	}
}

func resolveCommand(id string, availableCommands *CommandsRegistry) (Command, bool) {
	cmd, exists := availableCommands.Command(id)
	if exists {
		return cmd, true
	}

	helpCommand := &HelpCommand{}
	if id == helpCommand.Id() {
		return newDefaultHelpCommand(availableCommands), true
	}

	return nil, false
}

// Run processes the user input and executes the requested command without exiting the process.
// By default, it writes to os.Stdout if nil is provided for the io.Writer argument.
func Run(
	args []string,
	availableCommands *CommandsRegistry,
	outputWriter io.Writer,
) RunResult {
	return RunContext(context.Background(), args, availableCommands, outputWriter)
}

// RunContext processes the user input and executes the requested command without exiting the process.
// By default, it writes to os.Stdout if nil is provided for the io.Writer argument.
func RunContext(
	ctx context.Context,
	args []string,
	availableCommands *CommandsRegistry,
	outputWriter io.Writer,
) RunResult {
	if ctx == nil {
		ctx = context.Background()
	}

	if outputWriter == nil {
		outputWriter = os.Stdout
	}

	cmdId, cmdArgs := parseCmdInput(args)
	if cmdId == "" {
		cmdId = (&HelpCommand{}).Id()
	}

	result := RunResult{CommandID: cmdId, ExitCode: StatusOk}
	var cmdErr error
	cmd, exists := resolveCommand(cmdId, availableCommands)
	if !exists {
		cmdErr = fmt.Errorf("%w: The command %s does not exist", ErrCommandNotFound, cmdId)
	} else {
		cmdErr = runCommand(ctx, cmd, cmdArgs, outputWriter)
	}

	if cmdErr != nil {
		_, outputErr := outputWriter.Write(
			[]byte(
				fmt.Sprintf(
					"Failed to execute command %s with error: %s\n",
					cmdId,
					cmdErr.Error(),
				),
			),
		)
		if outputErr != nil {
			fmt.Printf(
				"Error writing to the provided output writer %s\n",
				reflect.TypeOf(outputWriter),
			)
		}
		result.ExitCode = ExitCodeFromError(cmdErr)
		result.Err = cmdErr
		return result
	}

	return result
}

// Bootstrap Will bootstrap everything needed for the user CLI request. Will process the
// user input, run the requested command, and call processExit with the resulting status code.
// By default, it will output to os.Stdout if nil is provided for the io.Writer argument.
func Bootstrap(
	args []string,
	availableCommands *CommandsRegistry,
	outputWriter io.Writer,
	processExit func(code int),
) {
	if processExit == nil {
		processExit = os.Exit
	}

	result := Run(args, availableCommands, outputWriter)
	processExit(result.ExitCode)
}
