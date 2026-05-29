# go-cli-command

A lightweight, flexible framework for building command-line applications in Go. This package
provides a simple way to define, register, and execute CLI commands with support for flags,
help documentation, process-safe locking, custom exit codes, and context-aware execution.

Migrated from https://github.com/rsgcata/go-cli-command

## Features

- Simple API for defining CLI commands
- First-class support for command-line flags with validation
- Built-in `help` command with deterministic command ordering
- File-based command locking to prevent concurrent execution, with configurable lock behavior
- Panic-safe command runner with error reporting and non-zero exit codes on failure
- Optional context-aware command execution via `RunContext` and `ContextCommand`
- Custom command exit codes via errors that implement `ExitCoder`
- Flexible output handling via injectable `io.Writer`
- Minimal dependencies (uses `github.com/golibry/go-fs` for file locking)
- Small, test-covered core

## Installation

Use `go get github.com/golibry/go-cli-command` from your application module.

## Getting Started

Create command-line applications by implementing `Command`, registering commands in a
`CommandsRegistry`, and executing the registry through `Bootstrap`, `Run`, or `RunContext`.

For runnable code and usage variants, start with:

- [_examples/README.md](_examples/README.md)
- [_examples/main.go](_examples/main.go)

## Core Components

### Command

The required interface for commands. It provides the command ID, help description, execution
method, flag definitions, and flag validation.

For commands without flags, embed `CommandWithoutFlags` to avoid boilerplate.

### ContextCommand

An optional interface for commands that need cancellation or deadlines. When a command implements
`ContextCommand`, the runner calls `ExecContext` instead of `Exec`.

Use it for commands that call databases, APIs, queues, or other resources that should respect
context cancellation.

### CommandsRegistry

Stores and resolves commands by ID. Registration validates commands and rejects nil commands,
empty command IDs, whitespace-padded IDs, and duplicate IDs.

It also exposes ordered command retrieval for deterministic help output.

### Help Command

The default `help` command is provided at runtime without mutating the caller's registry. Commands
are listed in deterministic order by command ID.

You may register your own `help` command to override the default behavior.

### Run, RunContext, and Bootstrap

- `Run` executes a command without exiting the process.
- `RunContext` executes a command without exiting and passes a context to context-aware commands.
- `Bootstrap` is the main entrypoint helper: it runs the requested command and calls the provided
  process exit function.

### Exit Codes

Plain command errors use `StatusErr`. Commands can opt into specific exit codes with errors that
implement `ExitCoder`, including errors created with `WithExitCode`.

### Locking

`FsLockableCommand` wraps any command with an exclusive file lock. Use it to prevent concurrent
execution across processes.

The basic constructors fail when the lock is held. The options-based constructor can also skip
successfully when another process already holds the lock, or wait for a timeout.

## Examples

The examples folder is the source of truth for practical usage:

- [_examples/README.md](_examples/README.md)
- [_examples/main.go](_examples/main.go)

It covers basic commands, flags, validation, `Bootstrap`, `Run`, `RunContext`, custom exit codes,
locking variants, panic recovery, and custom writer capture.

## License

This project is licensed under the terms found in the LICENSE file.
