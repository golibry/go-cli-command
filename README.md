# go-cli-command

A lightweight, flexible framework for building command-line applications in Go. This package 
provides a simple way to define, register, and execute CLI commands with support for flags and 
help documentation.  
Migrated from https://github.com/rsgcata/go-cli-command

## Features

- Simple, intuitive API for defining CLI commands
- First-class support for command-line flags with validation
- Built-in `help` command that lists commands and their flags with nicely wrapped descriptions
- File-based command locking to prevent concurrent execution (cross-process safe)
- Panic-safe command runner with error reporting and non-zero exit codes on failure
- Flexible output handling via injectable `io.Writer`
- Minimal dependencies (uses `github.com/golibry/go-fs` for file locking)
- Small, test-covered core

## Installation

```bash
go get github.com/golibry/go-cli-command
```

## Usage

Create command-line applications by:

1. Implementing the `Command` interface for each command
2. Registering your commands in a `CommandsRegistry`
3. Bootstrapping the application with the provided arguments

For commands without flags, embed `CommandWithoutFlags` to avoid boilerplate.

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

A helper that wraps any `Command` to enforce exclusive execution using a file lock. This ensures only one instance runs at a time, even across processes. When a lock is already held, `Exec` returns the sentinel `CommandLocked` error.

#### CommandWithoutFlags

For commands that don't need flags, you can embed this struct to avoid implementing empty methods.

#### CommandsRegistry

Manages the registration and retrieval of commands. Use `NewCommandsRegistry()` to create a new registry and `Register()` to add commands.

#### Bootstrap Function

The main entry point for your CLI application, which processes arguments, runs commands, and handles output.

## Examples

For complete, runnable examples (including command implementation, registration, bootstrapping, and locking), see the [_examples](/_examples) directory in this repository.

## License

This project is licensed under the terms found in the LICENSE file.
