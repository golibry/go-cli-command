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
	_, _ = fmt.Fprintln(writer, "\t")
	_, _ = fmt.Fprintln(writer, c.Id()+"\t"+c.Description())
	_, _ = fmt.Fprintln(writer, "\t")

	for _, command := range c.availableCommands {
		_, _ = fmt.Fprintln(writer, "\t")

		descChunks := chunkDescription(command.Description(), 80)
		_, _ = fmt.Fprintln(writer, command.Id()+"\t"+descChunks[0])
		if len(descChunks) > 1 {
			for _, descChunk := range descChunks[1:] {
				_, _ = fmt.Fprintln(writer, "\t"+descChunk)
			}
		}

		cmdFlagSet := setupFlagSet(command, writer)
		if cmdFlagSet != nil {
			command.DefineFlags(cmdFlagSet)
			countFlags := 0
			flagsListOutput := ""

			cmdFlagSet.VisitAll(
				func(flag *flag.Flag) {
					if flag != nil {
						countFlags++
						flagsListOutput += fmt.Sprintf(
							"\t--%s (default %s)\n",
							flag.Name,
							flag.DefValue,
						)
						usageChunks := chunkDescription(strings.Trim(flag.Usage, "\n "), 80)
						if len(usageChunks) > 0 {
							for _, usageChunk := range usageChunks {
								flagsListOutput += fmt.Sprintf("\t%s\n", usageChunk)
							}
						}
					}
				},
			)

			if countFlags > 0 {
				_, _ = fmt.Fprintln(writer, "\tFlags:")
				_, _ = fmt.Fprint(writer, flagsListOutput)
			} else {
				_, _ = fmt.Fprintln(writer, "\tFlags: none")
			}
		}

		_, _ = fmt.Fprintln(writer, "\t")
	}
	_ = writer.Flush()

	return nil
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
