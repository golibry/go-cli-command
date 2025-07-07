package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
)

const StatusOk = 0
const StatusErr = 1

// Command interface defines the methods that a command must implement
type Command interface {
	Id() string
	Description() string
	Exec(stdWriter io.Writer) error
	DefineFlags(flagSet *flag.FlagSet)
	ValidateFlags() error
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
func runCommand(cmd Command, args []string, outputWriter io.Writer) (cmdErr error) {
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
	if cmdErr = cmd.Exec(outputWriter); cmdErr != nil {
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

// Register adds a command to the registry
func (registry *CommandsRegistry) Register(cmd Command) error {
	if _, exists := registry.commands[cmd.Id()]; exists {
		return fmt.Errorf("command '%s' is already registered", cmd.Id())
	}
	registry.commands[cmd.Id()] = cmd
	return nil
}

// Commands returns a copy of all registered commands
func (registry *CommandsRegistry) Commands() map[string]Command {
	cmdCopy := make(map[string]Command, len(registry.commands))
	for name, cmd := range registry.commands {
		cmdCopy[name] = cmd
	}
	return cmdCopy
}

// Command returns a command by its ID
func (registry *CommandsRegistry) Command(id string) (Command, bool) {
	cmd, ok := registry.commands[id]
	return cmd, ok
}

// Bootstrap Will bootstrap everything needed for the user CLI request. Will process the
// user input and run the requested command. By default, will output to os.Stdout if
// nil is provided for the io.Writer argument.
func Bootstrap(
	args []string,
	availableCommands *CommandsRegistry,
	outputWriter io.Writer,
	processExit func(code int),
) {
	if outputWriter == nil {
		outputWriter = os.Stdout
	}

	if processExit == nil {
		processExit = os.Exit
	}

	_ = availableCommands.Register(
		&HelpCommand{
			CommandWithoutFlags{},
			slices.Collect(
				maps.Values(
					availableCommands.
						Commands(),
				),
			),
		},
	)

	cmdId, cmdArgs := parseCmdInput(args)
	if cmdId == "" {
		cmdId = (&HelpCommand{}).Id()
	}

	var cmdErr error
	cmd, exists := availableCommands.Command(cmdId)
	if !exists {
		cmdErr = fmt.Errorf("The command %s does not exist\n", cmdId)
	} else {
		cmdErr = runCommand(cmd, cmdArgs, outputWriter)
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
		processExit(StatusErr)
		return
	}

	processExit(StatusOk)
}
