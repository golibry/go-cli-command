package main

import (
	"flag"
	"fmt"
	"github.com/golibry/go-cli-command/cli"
	"io"
	"os"
	"time"
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
	panic("implement me")
	_, _ = stdWriter.Write([]byte("Hello there!"))
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

func (s SayHelloDynamic) Id() string {
	return "say-hello-dynamic"
}

func (s SayHelloDynamic) Description() string {
	return "A basic command that will greet the user based on the given input."
}

func (s SayHelloDynamic) Exec(stdWriter io.Writer) error {
	for i := 0; i < s.ParsedFlags.CountTo; i++ {
		_, _ = stdWriter.Write([]byte("Hello there " + s.ParsedFlags.Name + "\n"))
		time.Sleep(s.ParsedFlags.CountDelay)
	}
	return nil
}

func (s SayHelloDynamic) DefineFlags(flagSet *flag.FlagSet) {
	flagSet.StringVar(&s.ParsedFlags.Name, "name", "", "Specify the user Name to greet.")
	flagSet.IntVar(&s.ParsedFlags.CountTo, "count-to", 1, "Specify the number of times to greet.")
	flagSet.DurationVar(
		&s.ParsedFlags.CountDelay, "count-delay", 1*time.Second,
		"Specify the delay between greet repeats.",
	)
}

func (s SayHelloDynamic) ValidateFlags() error {
	if s.ParsedFlags.CountTo <= 0 || s.ParsedFlags.CountDelay <= 0 {
		return fmt.Errorf(
			"count-to and count-delay must be greater than 0, got %d, %d",
			s.ParsedFlags.CountTo,
			s.ParsedFlags.CountDelay,
		)
	}

	return nil
}

func main() {
	registry := cli.NewCommandsRegistry()
	availableCommands := []cli.Command{
		&SayHello{},
		cli.NewLockableCommand(
			&SayHelloDynamic{ParsedFlags: &SayHelloFlags{}},
			os.TempDir(),
		),
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
