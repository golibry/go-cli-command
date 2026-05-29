package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestItCanDisplayHelpfulInformationAboutAvailableCommands(t *testing.T) {
	// Create a mock command to be listed in help
	mockCmd := &MockCommand{
		id:          "test-cmd",
		description: "Test command description",
	}

	// Create a mock command with flagSet
	mockCmdWithFlags := &MockCommandWithFlags{
		id:          "flag-cmd",
		description: "Command with flagSet",
	}

	// Create help command with the mock commands
	helpCmd := &HelpCommand{
		availableCommands: []Command{mockCmd, mockCmdWithFlags},
	}

	// Execute the help command
	var buf bytes.Buffer
	err := helpCmd.Exec(&buf)
	if err != nil {
		t.Errorf("HelpCommand.Exec() error = %v, want nil", err)
	}

	// Check that the output contains expected information
	output := buf.String()

	if !strings.Contains(output, "Usage:") {
		t.Errorf("Help output doesn't contain usage section")
	}
	if !strings.Contains(output, "<command> [flags]") {
		t.Errorf("Help output doesn't contain command usage")
	}
	if !strings.Contains(output, "Available commands:") {
		t.Errorf("Help output doesn't contain available commands section")
	}

	// Check command IDs are in the output
	if !strings.Contains(output, "help") {
		t.Errorf("Help output doesn't contain the help command ID")
	}
	if !strings.Contains(output, "test-cmd") {
		t.Errorf("Help output doesn't contain the test command ID")
	}
	if !strings.Contains(output, "flag-cmd") {
		t.Errorf("Help output doesn't contain the flag command ID")
	}

	// Check descriptions are in the output
	if !strings.Contains(output, "Test command description") {
		t.Errorf("Help output doesn't contain the test command description")
	}
	if !strings.Contains(output, "Command with flagSet") {
		t.Errorf("Help output doesn't contain the flag command description")
	}

	// Check that flag information is included
	if !strings.Contains(output, "Flags:") {
		t.Errorf("Help output doesn't contain flag section")
	}
	if !strings.Contains(output, "--test-flag") {
		t.Errorf("Help output doesn't contain flag name")
	}
	if !strings.Contains(output, `(default "")`) {
		t.Errorf("Help output doesn't contain quoted flag default")
	}
}

func TestHelpCommandSkipsDuplicateHelpEntry(t *testing.T) {
	helpCmd := &HelpCommand{
		availableCommands: []Command{
			&HelpCommand{},
			&MockCommand{id: "test-cmd", description: "Test command"},
		},
	}

	var buf bytes.Buffer
	err := helpCmd.Exec(&buf)
	if err != nil {
		t.Errorf("HelpCommand.Exec() error = %v, want nil", err)
	}

	helpRows := countCommandRows(buf.String(), "help")
	if helpRows != 1 {
		t.Errorf("Help output contains %d help rows, want 1. Output:\n%s", helpRows, buf.String())
	}
}

func countCommandRows(output string, commandId string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == commandId {
			count++
		}
	}
	return count
}

func TestItCanChunkDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		size        int
		want        []string
	}{
		{
			name:        "empty description",
			description: "",
			size:        10,
			want:        []string{""},
		},
		{
			name:        "short description",
			description: "Short text",
			size:        20,
			want:        []string{"Short text"},
		},
		{
			name:        "long description",
			description: "This is a longer description that should be split into multiple chunks",
			size:        20,
			want: []string{
				"This is a longer description",
				"that should be split",
				"into multiple chunks",
			},
		},
		{
			name:        "description with newlines",
			description: "First line\nSecond line\nThird line",
			size:        20,
			want: []string{
				"First line",
				"Second line",
				"Third line",
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := chunkDescription(tt.description, tt.size)
				if len(got) != len(tt.want) {
					t.Errorf(
						"chunkDescription() returned %d chunks, want %d",
						len(got),
						len(tt.want),
					)
					return
				}
				for i, chunk := range got {
					if chunk != tt.want[i] {
						t.Errorf("chunkDescription() chunk[%d] = %q, want %q", i, chunk, tt.want[i])
					}
				}
			},
		)
	}
}
