# go-cli-command

A lightweight, flexible framework for building command-line applications in Go. This package 
provides a simple way to define, register, and execute CLI commands with support for flags and 
help documentation.  
Migrated from https://github.com/rsgcata/go-cli-command

## Features

- Simple and intuitive API for defining CLI commands
- Support for command-line flags with validation
- Built-in help command that displays available commands and their flags
- Command locking mechanism to prevent concurrent execution
- Flexible output handling
- Minimal dependencies

## Installation

```bash
go get github.com/golibry/go-cli-command
```

## Usage

The package provides a straightforward way to create command-line applications:

1. Define your commands by implementing the `Command` interface
2. Register your commands with the `CommandsRegistry`
3. Bootstrap your application with the provided arguments

For commands without flags, you can embed the `CommandWithoutFlags` struct to avoid implementing empty methods.

## Documentation

### Core Components

#### Command Interface

The `Command` interface defines the methods that a command must implement:

- `Id() string`: Unique identifier for the command
- `Description() string`: Description shown in help
- `Exec(stdWriter io.Writer) error`: Execute the command
- `DefineFlags(flagSet *flag.FlagSet)`: Define command-specific flags
- `ValidateFlags() error`: Validate the parsed flags

#### FsLockableCommand

A helper struct that implements the `Command` interface and provides file-based locking to prevent concurrent execution of commands.

Example usage:

1. Create a new FsLockableCommand for a command with a directory for the lock file:
   ```
   lockableCmd := cli.NewLockableCommand(myCommand, os.TempDir())
   ```

2. Or with a custom lock name:
   ```
   lockableCmd := cli.NewLockableCommandWithLockName(myCommand, os.TempDir(), "custom-lock-name")
   ```

3. Register the helper instead of the original command:
   ```
   registry.Register(lockableCmd)
   ```

The helper uses file locks to ensure that only one instance of the command can run at a time, even across different processes. When a command is locked, the `Exec` method will return a `CommandLocked` error.

#### CommandWithoutFlags

For commands that don't need flags, you can embed this struct to avoid implementing empty methods.

#### CommandsRegistry

Manages the registration and retrieval of commands. Use `NewCommandsRegistry()` to create a new registry and `Register()` to add commands.

#### Bootstrap Function

The main entry point for your CLI application, which processes arguments, runs commands, and handles output.

## Examples

For complete examples of how to use this package, please see the [_examples](/_examples) directory in this repository.

## License

This project is licensed under the terms found in the LICENSE file.
