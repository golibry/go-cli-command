# Examples

This folder contains runnable examples that demonstrate how to use each feature of `github.com/golibry/go-cli-command`.

Build or run from the repository root:

```
go run ./_examples -- help
```

You can replace `-- help` with any command below. The `--` ensures the program name is stripped (equivalent to passing `os.Args[1:]`).

## Available example commands

- help
  - Lists all available commands and their flags with nicely wrapped descriptions.
  - Run: `go run ./_examples -- help`

- say-hello
  - A basic command without flags. Demonstrates writing to an injected `io.Writer`.
  - Run: `go run ./_examples -- say-hello`
  - Expected output:
    ```
    Hello there!
    ```

- say-hello-dynamic (with file-based locking)
  - Demonstrates defining and validating flags, plus file-based locking via `FsLockableCommand`.
  - Flags:
    - `--name` (string): whom to greet
    - `--count-to` (int, default 1): number of greetings
    - `--count-delay` (duration, default 1s): delay between greetings
  - Run:
    - Single run: `go run ./_examples -- say-hello-dynamic --name Alice --count-to 2 --count-delay 500ms`
    - Concurrent lock demo: open two terminals and run the same command at the same time. One will execute; the other will skip with a “command is locked” error and exit non-zero.

- panic-demo
  - Demonstrates panic-safe execution: the library recovers from panics and returns a proper error.
  - Run: `go run ./_examples -- panic-demo`
  - Expected behavior: process exits non-zero and prints an error similar to:
    ```
    Failed to execute command panic-demo with error: this is an intentional panic for demo purposes
    ```

- custom-writer-demo
  - Shows how to pass a custom `io.Writer` to capture CLI output programmatically. Internally, it runs a small registry with the `help` command and prints the captured buffer.
  - Run: `go run ./_examples -- custom-writer-demo`
  - Expected output starts with:
    ```
    Captured output from custom writer demo:
    ```

## Notes

- The `help` command is automatically registered by `Bootstrap` and will list the other commands along with their flags.
- `FsLockableCommand` uses a lock file in your temporary directory (e.g., returned by `os.TempDir()`). This ensures cross-process safety for the locking example.
- Exit codes: successful runs exit with `0`; failures (including unknown commands, invalid flags, or panics) exit with `1` and print a helpful message.
