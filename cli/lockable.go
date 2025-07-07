package cli

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/golibry/go-fs"
	"github.com/golibry/go-fs/filelock"
	"io"
	"path/filepath"
	"regexp"
)

var CommandLocked = errors.New("command is locked, skipping execution")

func normalizeCommandId(id string) string {
	var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return nonAlphanumericRegex.ReplaceAllString(id, "-")
}

// FsLockableCommand is a helper struct that implements the locking mechanism
// for commands that need to run exclusively (preventing concurrent execution).
type FsLockableCommand struct {
	// The command that needs to be locked
	Command Command

	// The lock file
	fileLock filelock.FileLock
}

// NewLockableCommand creates a new FsLockableCommand for the given command.
// The lock file will be created with the Command.Id() in its name.
func NewLockableCommand(
	cmd Command,
	lockFileDirPath string,
) *FsLockableCommand {
	return NewLockableCommandWithLockName(cmd, lockFileDirPath, cmd.Id())
}

// NewLockableCommandWithLockName creates a new FsLockableCommand for the given command.
// The lock file will be created with the given lockName in its name.
func NewLockableCommandWithLockName(
	cmd Command,
	lockFileDirPath string,
	lockName string,
) *FsLockableCommand {
	idHash := md5.Sum([]byte(lockName))
	lockFilePath := filepath.Join(
		lockFileDirPath,
		fmt.Sprintf(
			"go-cli-command-%s-%s.lock",
			normalizeCommandId(lockName),
			hex.EncodeToString(idHash[:]),
		),
	)
	return &FsLockableCommand{
		Command:  cmd,
		fileLock: fs.New(lockFilePath),
	}
}

// Id returns the ID of the wrapped command.
func (l *FsLockableCommand) Id() string {
	return l.Command.Id()
}

// Description returns the description of the wrapped command.
func (l *FsLockableCommand) Description() string {
	return l.Command.Description()
}

// DefineFlags delegates to the wrapped command.
func (l *FsLockableCommand) DefineFlags(flagSet *flag.FlagSet) {
	l.Command.DefineFlags(flagSet)
}

// ValidateFlags delegates to the wrapped command.
func (l *FsLockableCommand) ValidateFlags() error {
	return l.Command.ValidateFlags()
}

// Exec acquires the lock, executes the wrapped command, and then releases the lock.
func (l *FsLockableCommand) Exec(stdWriter io.Writer) error {
	locked, err := l.Lock()
	if err != nil {
		return err
	}

	if locked {
		// Ensure the lock is released when the function returns
		defer func(l *FsLockableCommand) {
			_ = l.Unlock()
		}(l)

		// Execute the wrapped command
		return l.Command.Exec(stdWriter)
	} else {
		return CommandLocked
	}
}

// Lock acquires both the in-memory mutex and the file lock.
// If the lock cannot be acquired, it returns an error.
func (l *FsLockableCommand) Lock() (bool, error) {
	err := l.fileLock.Lock()
	if err != nil {
		if errors.Is(err, filelock.ErrLockHeld) {
			return false, nil
		} else {
			return false, fmt.Errorf(
				"failed to acquire lock for command %s: %w",
				l.Id(),
				err,
			)
		}
	}

	return true, nil
}

// Unlock releases both the in-memory mutex and the file lock.
func (l *FsLockableCommand) Unlock() error {
	return l.fileLock.Unlock()
}
