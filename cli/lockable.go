package cli

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"time"

	"github.com/golibry/go-fs"
	"github.com/golibry/go-fs/filelock"
)

var CommandLocked = errors.New("command is locked, skipping execution")

type LockedBehavior int

const (
	FailWhenLocked LockedBehavior = iota
	SkipWhenLocked
)

type LockOptions struct {
	LockFileDirPath string
	LockName        string
	Timeout         time.Duration
	WhenLocked      LockedBehavior
}

func normalizeCommandId(id string) string {
	var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return nonAlphanumericRegex.ReplaceAllString(id, "-")
}

func lockFilePath(lockFileDirPath string, lockName string) string {
	idHash := md5.Sum([]byte(lockName))
	return filepath.Join(
		lockFileDirPath,
		fmt.Sprintf(
			"go-cli-command-%s-%s.lock",
			normalizeCommandId(lockName),
			hex.EncodeToString(idHash[:]),
		),
	)
}

// FsLockableCommand is a helper struct that implements the locking mechanism
// for commands that need to run exclusively (preventing concurrent execution).
type FsLockableCommand struct {
	// The command that needs to be locked
	Command Command

	// The lock file
	fileLock filelock.FileLock

	lockTimeout time.Duration
	whenLocked  LockedBehavior
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
	return NewLockableCommandWithOptions(
		cmd,
		LockOptions{
			LockFileDirPath: lockFileDirPath,
			LockName:        lockName,
			WhenLocked:      FailWhenLocked,
		},
	)
}

// NewLockableCommandWithOptions creates a new FsLockableCommand for the given command,
// with configurable lock name, timeout, and behavior when the lock is already held.
func NewLockableCommandWithOptions(
	cmd Command,
	options LockOptions,
) *FsLockableCommand {
	lockName := options.LockName
	if lockName == "" {
		lockName = cmd.Id()
	}

	return &FsLockableCommand{
		Command:     cmd,
		fileLock:    fs.New(lockFilePath(options.LockFileDirPath, lockName)),
		lockTimeout: options.Timeout,
		whenLocked:  options.WhenLocked,
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
	return l.exec(context.Background(), func() error {
		return l.Command.Exec(stdWriter)
	})
}

// ExecContext uses ctx while acquiring the lock and passes it to context-aware commands.
// A non-positive lock timeout still performs a non-blocking acquisition.
func (l *FsLockableCommand) ExecContext(ctx context.Context, stdWriter io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return l.exec(ctx, func() error {
		if cmd, ok := l.Command.(ContextCommand); ok {
			return cmd.ExecContext(ctx, stdWriter)
		}
		return l.Command.Exec(stdWriter)
	})
}

func (l *FsLockableCommand) exec(ctx context.Context, run func() error) error {
	locked, err := l.lock(ctx)
	if err != nil {
		return err
	}

	if locked {
		// Ensure the lock is released when the function returns
		defer func(l *FsLockableCommand) {
			_ = l.Unlock()
		}(l)

		if err := ctx.Err(); err != nil {
			return err
		}
		return run()
	}

	if l.whenLocked == SkipWhenLocked {
		return nil
	}

	return CommandLocked
}

// Lock acquires both the in-memory mutex and the file lock.
// If the lock cannot be acquired, it returns an error.
func (l *FsLockableCommand) Lock() (bool, error) {
	return l.lock(context.Background())
}

func (l *FsLockableCommand) lock(ctx context.Context) (bool, error) {
	err := ctx.Err()
	if err == nil {
		if l.lockTimeout > 0 {
			lockCtx, cancel := context.WithTimeout(ctx, l.lockTimeout)
			defer cancel()
			err = l.fileLock.LockContext(lockCtx)
			// Only the configured lock timeout follows the skip/fail policy.
			if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
				err = filelock.ErrTimeout
			}
		} else {
			err = l.fileLock.Lock()
		}
	}

	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
		}
		if errors.Is(err, filelock.ErrLockHeld) || errors.Is(err, filelock.ErrTimeout) {
			return false, nil
		}

		return false, fmt.Errorf(
			"failed to acquire lock for command %s: %w",
			l.Id(),
			err,
		)
	}

	return true, nil
}

// Unlock releases both the in-memory mutex and the file lock.
func (l *FsLockableCommand) Unlock() error {
	return l.fileLock.Unlock()
}
