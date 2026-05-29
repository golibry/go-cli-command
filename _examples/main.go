package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golibry/go-cli-command/cli"
)

type SayHello struct {
	cli.CommandWithoutFlags
}

func (s SayHello) Id() string {
	return "say-hello"
}

func (s SayHello) Description() string {
	return "A basic command that will greet the user."
}

func (s SayHello) Exec(stdWriter io.Writer) error {
	// Basic command without flags. Demonstrates writing to the provided io.Writer.
	_, _ = stdWriter.Write([]byte("Hello there!\n"))
	return nil
}

type SayHelloFlags struct {
	Name       string
	CountTo    int
	CountDelay time.Duration
}

type SayHelloDynamic struct {
	ParsedFlags *SayHelloFlags
}

func (s *SayHelloDynamic) Id() string {
	return "say-hello-dynamic"
}

func (s *SayHelloDynamic) Description() string {
	return "A basic command that will greet the user based on the given input."
}

func (s *SayHelloDynamic) Exec(stdWriter io.Writer) error {
	for i := 0; i < s.ParsedFlags.CountTo; i++ {
		_, _ = stdWriter.Write([]byte("Hello there " + s.ParsedFlags.Name + "\n"))
		time.Sleep(s.ParsedFlags.CountDelay)
	}
	return nil
}

func (s *SayHelloDynamic) DefineFlags(flagSet *flag.FlagSet) {
	flagSet.StringVar(&s.ParsedFlags.Name, "name", "", "Specify the user Name to greet.")
	flagSet.IntVar(&s.ParsedFlags.CountTo, "count-to", 1, "Specify the number of times to greet.")
	flagSet.DurationVar(
		&s.ParsedFlags.CountDelay, "count-delay", 1*time.Second,
		"Specify the delay between greet repeats.",
	)
}

func (s *SayHelloDynamic) ValidateFlags() error {
	if s.ParsedFlags.CountTo <= 0 || s.ParsedFlags.CountDelay <= 0 {
		return cli.WithExitCode(fmt.Errorf(
			"count-to and count-delay must be greater than 0, got %d, %d",
			s.ParsedFlags.CountTo,
			s.ParsedFlags.CountDelay,
		), 2)
	}

	return nil
}

// PanicDemo demonstrates the panic-safe runner. It will panic during Exec.
type PanicDemo struct{ cli.CommandWithoutFlags }

func (p *PanicDemo) Id() string { return "panic-demo" }
func (p *PanicDemo) Description() string {
	return "Demonstrates panic-safe execution by intentionally panicking."
}
func (p *PanicDemo) Exec(_ io.Writer) error {
	panic("this is an intentional panic for demo purposes")
}

// ExitCodeDemo demonstrates returning a command-specific process exit code.
type ExitCodeDemo struct{ cli.CommandWithoutFlags }

func (e *ExitCodeDemo) Id() string { return "exit-code-demo" }
func (e *ExitCodeDemo) Description() string {
	return "Demonstrates returning a custom exit code from a command error."
}
func (e *ExitCodeDemo) Exec(_ io.Writer) error {
	return cli.WithExitCode(fmt.Errorf("this command failed with a custom exit code"), 7)
}

// ContextDemo demonstrates the optional ContextCommand interface.
type ContextDemo struct{ cli.CommandWithoutFlags }

func (c *ContextDemo) Id() string { return "context-demo" }
func (c *ContextDemo) Description() string {
	return "Demonstrates ExecContext for commands that need cancellation or deadlines."
}
func (c *ContextDemo) Exec(stdWriter io.Writer) error {
	return c.ExecContext(context.Background(), stdWriter)
}
func (c *ContextDemo) ExecContext(ctx context.Context, stdWriter io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, _ = stdWriter.Write([]byte("Context command executed.\n"))
		return nil
	}
}

// CustomWriterDemo shows how to inject a custom io.Writer and capture output
// programmatically by invoking Run with a bytes.Buffer. This command
// runs a tiny, isolated registry in-memory and prints the captured output.
type CustomWriterDemo struct{ cli.CommandWithoutFlags }

func (c *CustomWriterDemo) Id() string { return "custom-writer-demo" }
func (c *CustomWriterDemo) Description() string {
	return "Shows how to run Run with a custom io.Writer and capture output."
}
func (c *CustomWriterDemo) Exec(stdWriter io.Writer) error {
	// Create a small registry with a single command whose output we capture
	reg := cli.NewCommandsRegistry()
	_ = reg.Register(&SayHello{})

	var buf bytes.Buffer
	// Simulate running the help command and capture its output in buf.
	result := cli.Run([]string{"help"}, reg, &buf)
	if result.Err != nil {
		return result.Err
	}

	// Print the captured output to the main example's writer
	_, _ = stdWriter.Write([]byte("Captured output from custom writer demo:\n"))
	_, _ = stdWriter.Write(buf.Bytes())
	return nil
}

func main() {
	registry := cli.NewCommandsRegistry()
	availableCommands := []cli.Command{
		&SayHello{},
		// Wrap a command with file-based locking to prevent concurrent execution.
		// Try running two processes concurrently to see the second one skip execution.
		cli.NewLockableCommandWithOptions(
			&SayHelloDynamic{ParsedFlags: &SayHelloFlags{}},
			cli.LockOptions{
				LockFileDirPath: os.TempDir(),
				LockName:        "say-hello-dynamic",
				WhenLocked:      cli.SkipWhenLocked,
			},
		),
		&PanicDemo{},
		&ExitCodeDemo{},
		&ContextDemo{},
		&CustomWriterDemo{},
	}

	for _, cmd := range availableCommands {
		err := registry.Register(cmd)
		if err != nil {
			panic(err)
		}
	}

	// os.Args[1:] is mandatory to remove the program Name from the args slice
	cli.Bootstrap(os.Args[1:], registry, os.Stdout, os.Exit)
}
