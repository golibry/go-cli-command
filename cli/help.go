package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type HelpCommand struct {
	CommandWithoutFlags
	availableCommands []Command
}

func NewHelpCommand(availableCommands []Command) *HelpCommand {
	return &HelpCommand{availableCommands: availableCommands}
}

func (c *HelpCommand) Id() string {
	return "help"
}

func (c *HelpCommand) Description() string {
	return "Lists all available commands"
}

func (c *HelpCommand) Exec(baseWriter io.Writer) error {
	writer := tabwriter.NewWriter(baseWriter, 0, 0, 4, ' ', 0)
	_, _ = fmt.Fprintln(writer, "Usage:")
	_, _ = fmt.Fprintln(writer, "  <command> [flags]")
	_, _ = fmt.Fprintln(writer)
	_, _ = fmt.Fprintln(writer, "Available commands:")

	writeCommandHelp(writer, c)
	for _, command := range c.availableCommands {
		if command.Id() == c.Id() {
			continue
		}

		writeCommandHelp(writer, command)
	}
	_ = writer.Flush()

	return nil
}

func writeCommandHelp(writer *tabwriter.Writer, command Command) {
	_, _ = fmt.Fprintln(writer)

	descChunks := chunkDescription(command.Description(), 80)
	_, _ = fmt.Fprintf(writer, "  %s\t%s\n", command.Id(), descChunks[0])
	if len(descChunks) > 1 {
		for _, descChunk := range descChunks[1:] {
			_, _ = fmt.Fprintf(writer, "  \t%s\n", descChunk)
		}
	}

	writeFlagsHelp(writer, command)
}

func writeFlagsHelp(writer *tabwriter.Writer, command Command) {
	cmdFlagSet := setupFlagSet(command, writer)
	if cmdFlagSet == nil {
		return
	}

	command.DefineFlags(cmdFlagSet)
	countFlags := 0
	var flagsListOutput strings.Builder

	cmdFlagSet.VisitAll(
		func(flag *flag.Flag) {
			if flag == nil {
				return
			}

			countFlags++
			flagsListOutput.WriteString(
				fmt.Sprintf("    --%s\t(default %q)\n", flag.Name, flag.DefValue),
			)

			usageChunks := chunkDescription(strings.Trim(flag.Usage, "\n "), 80)
			for _, usageChunk := range usageChunks {
				flagsListOutput.WriteString(fmt.Sprintf("    \t%s\n", usageChunk))
			}
		},
	)

	if countFlags > 0 {
		_, _ = fmt.Fprintln(writer, "    Flags:")
		_, _ = fmt.Fprint(writer, flagsListOutput.String())
	} else {
		_, _ = fmt.Fprintln(writer, "    Flags: none")
	}
}

func chunkDescription(description string, size int) []string {
	if len(description) == 0 {
		return []string{""}
	}

	var chunks []string
	accumulator := ""
	for _, char := range description {
		accumulator += string(char)
		if (len(accumulator) >= size && string(char) == " ") || string(char) == "\n" {
			chunks = append(chunks, strings.TrimSpace(accumulator))
			accumulator = ""
		}
	}

	if len(accumulator) > 0 {
		chunks = append(chunks, accumulator)
	}

	return chunks
}
