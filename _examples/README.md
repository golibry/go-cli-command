# Examples

This folder contains runnable examples that demonstrate how to use
`github.com/golibry/go-cli-command`.

Run from the repository root:

```bash
go run ./_examples -- help
```

You can replace `help` with any command below. The `--` ensures the program name is stripped,
which is equivalent to passing `os.Args[1:]`.

## Usage Variants

### Main Entrypoint With Bootstrap

Use `Bootstrap` when building a normal CLI `main.go`. It runs the selected command and delegates
the final process exit decision to the provided exit function.

```go
registry := cli.NewCommandsRegistry()
_ = registry.Register(&SayHello{})

cli.Bootstrap(os.Args[1:], registry, os.Stdout, os.Exit)
```

### Embedded Or Testable Execution With Run

Use `Run` when you want to execute commands without exiting the current process. This is useful in
tests, nested command runners, or libraries that embed a CLI registry.

```go
var buf bytes.Buffer
result := cli.Run([]string{"help"}, registry, &buf)
if result.Err != nil {
    return result.Err
}
fmt.Print(buf.String())
```

### Context-Aware Execution With RunContext

Use `RunContext` with commands that implement `ContextCommand`. The runner calls `ExecContext`
instead of `Exec` when the command supports it.

```go
type ImportCommand struct {
    cli.CommandWithoutFlags
}

func (c *ImportCommand) Id() string          { return "import" }
func (c *ImportCommand) Description() string { return "Imports data." }
func (c *ImportCommand) Exec(w io.Writer) error {
    return c.ExecContext(context.Background(), w)
}
func (c *ImportCommand) ExecContext(ctx context.Context, w io.Writer) error {
    return importer.Import(ctx)
}

result := cli.RunContext(ctx, []string{"import"}, registry, os.Stdout)
```

### Commands With Flags And Validation

Define flags in `DefineFlags`, then validate parsed values in `ValidateFlags`.

```go
func (c *CommandWithFlags) DefineFlags(flagSet *flag.FlagSet) {
    flagSet.StringVar(&c.name, "name", "", "Name to greet.")
}

func (c *CommandWithFlags) ValidateFlags() error {
    if c.name == "" {
        return cli.WithExitCode(errors.New("name is required"), 2)
    }
    return nil
}
```

### Custom Exit Codes

Plain errors use `StatusErr`. Use `WithExitCode` when a command needs a specific process exit code.

```go
func (c *ValidateCommand) Exec(w io.Writer) error {
    return cli.WithExitCode(errors.New("invalid input"), 2)
}
```

### File Locking

Use the basic lock wrapper when concurrent execution should fail.

```go
locked := cli.NewLockableCommand(command, os.TempDir())
```

Use options when concurrent execution should skip successfully, or when the command should wait for
the lock for a limited time.

```go
locked := cli.NewLockableCommandWithOptions(command, cli.LockOptions{
    LockFileDirPath: os.TempDir(),
    LockName:        "nightly-import",
    Timeout:         5 * time.Second,
    WhenLocked:      cli.SkipWhenLocked,
})
```

### Custom Help

The default `help` command is virtual: it is available at runtime without being registered in your
registry. Register a command with ID `help` if you want to replace it.

```go
_ = registry.Register(&MyCustomHelpCommand{})
```

## Runnable Commands

### help

Lists available commands and their flags.

```bash
go run ./_examples -- help
```

### say-hello

A basic command without flags. Demonstrates writing to an injected `io.Writer`.

```bash
go run ./_examples -- say-hello
```

Expected output:

```text
Hello there!
```

### say-hello-dynamic

Demonstrates flags, validation, custom validation exit codes, and file-based locking.

Flags:

- `--name`: whom to greet
- `--count-to`: number of greetings, default `1`
- `--count-delay`: delay between greetings, default `1s`

```bash
go run ./_examples -- say-hello-dynamic --name Alice --count-to 2 --count-delay 500ms
```

Try running it in two terminals at the same time. The command uses `SkipWhenLocked`, so the second
process exits successfully without running the command body while the first process holds the lock.

Invalid flags use `cli.WithExitCode(..., 2)`:

```bash
go run ./_examples -- say-hello-dynamic --count-to 0
```

### context-demo

Demonstrates the optional `ContextCommand` interface. The example binary calls `Bootstrap`, which
uses a background context internally, but library consumers can call `RunContext` with their own
context.

```bash
go run ./_examples -- context-demo
```

### exit-code-demo

Demonstrates returning a command-specific process exit code.

```bash
go run ./_examples -- exit-code-demo
```

Expected behavior: the command fails and exits with code `7`. When running through `go run`, the Go
tool reports this as `exit status 7`.

### panic-demo

Demonstrates panic-safe execution. The library recovers from panics and returns a proper error.

```bash
go run ./_examples -- panic-demo
```

Expected behavior: process exits non-zero and prints an error similar to:

```text
Failed to execute command panic-demo with error: this is an intentional panic for demo purposes
```

### custom-writer-demo

Shows how to call `cli.Run` with a custom `io.Writer` to capture CLI output programmatically.
Internally, it runs a small registry with the `help` command and prints the captured buffer.

```bash
go run ./_examples -- custom-writer-demo
```

Expected output starts with:

```text
Captured output from custom writer demo:
```

## Notes

- The default `help` command is available at runtime without being registered in your command registry.
- `FsLockableCommand` uses a lock file in the configured directory.
- Successful runs exit with `0`.
- Plain failures exit with `1`.
- Commands can return specific exit codes with `cli.WithExitCode(err, code)`.
