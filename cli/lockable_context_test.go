package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/golibry/go-fs/filelock"
)

func runLockableContext(
	t *testing.T,
	ctx context.Context,
	cmd *FsLockableCommand,
	writer io.Writer,
) RunResult {
	t.Helper()
	var registry CommandsRegistry
	if err := registry.Register(cmd); err != nil {
		t.Fatal(err)
	}
	return RunContext(ctx, []string{cmd.Id()}, &registry, writer)
}

func TestLockableCommandRunContextDispatchAndUnlock(t *testing.T) {
	for _, outcome := range []string{"success", "error", "panic"} {
		t.Run(
			outcome, func(t *testing.T) {
				type contextKey struct{}
				ctx := context.WithValue(context.Background(), contextKey{}, "request-id")
				commandErr := errors.New("command failed")
				var output bytes.Buffer
				cmd := &MockContextCommand{MockCommand: MockCommand{id: "context-command"}}
				helper := NewLockableCommandWithOptions(
					cmd, LockOptions{
						LockFileDirPath: t.TempDir(), Timeout: time.Second,
					},
				)
				cmd.execContext = func(got context.Context, writer io.Writer) error {
					if got != ctx || writer != &output {
						t.Error("ExecContext did not receive the original context and writer")
					}
					if !helper.fileLock.IsLocked() {
						t.Error("command executed without its lock")
					}
					_, _ = io.WriteString(writer, "executed")
					switch outcome {
					case "error":
						return commandErr
					case "panic":
						panic(commandErr)
					}
					return nil
				}
				result := runLockableContext(t, ctx, helper, &output)
				if outcome == "success" {
					if result.Err != nil {
						t.Fatal(result.Err)
					}
				} else if !errors.Is(result.Err, commandErr) {
					t.Errorf("error = %v, want %v", result.Err, commandErr)
				}
				if cmd.ctx != ctx || cmd.execWasCalled || !bytes.HasPrefix(
					output.Bytes(),
					[]byte("executed"),
				) {
					t.Error("locked context command did not dispatch exclusively to ExecContext")
				}
				if err := helper.fileLock.Lock(); err != nil {
					t.Fatalf("lock was not released: %v", err)
				}
				_ = helper.Unlock()
			},
		)
	}
}

func TestLockableCommandOrdinaryExecPreserved(t *testing.T) {
	want := errors.New("ordinary execution")
	cmd := &MockContextCommand{
		MockCommand: MockCommand{
			id: "ordinary-command", execFunc: func(io.Writer) error { return want },
		},
	}
	helper := NewLockableCommand(cmd, t.TempDir())
	if err := helper.Exec(io.Discard); !errors.Is(err, want) {
		t.Fatalf("Exec error = %v, want %v", err, want)
	}
	if !cmd.execWasCalled || cmd.ctx != nil {
		t.Error("Exec must continue calling the wrapped command's Exec")
	}
	if helper.fileLock.IsLocked() {
		t.Error("Exec did not release its lock")
	}
}

func TestLockableCommandRunContextRegularCommand(t *testing.T) {
	cmd := &MockLockableCommand{id: "regular-command"}
	helper := NewLockableCommand(cmd, t.TempDir())
	result := runLockableContext(t, nil, helper, io.Discard)
	if result.Err != nil || cmd.executions != 1 {
		t.Fatalf("regular execution: result = %+v, executions = %d", result, cmd.executions)
	}
}

func TestLockableCommandExecContextNil(t *testing.T) {
	cmd := &MockContextCommand{MockCommand: MockCommand{id: "nil-context"}}
	helper := NewLockableCommand(cmd, t.TempDir())
	if err := helper.ExecContext(nil, io.Discard); err != nil {
		t.Fatal(err)
	}
	if cmd.ctx == nil || cmd.execWasCalled {
		t.Error("ExecContext(nil) must pass a non-nil context to the wrapped command")
	}
}

func TestLockableCommandExecContextAlreadyLocked(t *testing.T) {
	for _, behavior := range []LockedBehavior{FailWhenLocked, SkipWhenLocked} {
		cmd := &MockContextCommand{MockCommand: MockCommand{id: "already-locked"}}
		helper := NewLockableCommandWithOptions(
			cmd, LockOptions{
				LockFileDirPath: t.TempDir(), Timeout: time.Second, WhenLocked: behavior,
			},
		)
		if locked, err := helper.Lock(); err != nil || !locked {
			t.Fatalf("Lock() = %v, %v", locked, err)
		}
		defer func() { _ = helper.Unlock() }()
		for _, exec := range []func() error{
			func() error { return helper.Exec(io.Discard) },
			func() error { return helper.ExecContext(context.Background(), io.Discard) },
		} {
			if err := exec(); !errors.Is(err, filelock.ErrAlreadyLocked) {
				t.Errorf("same-instance execution = %v, want ErrAlreadyLocked", err)
			}
		}
		if !helper.fileLock.IsLocked() || cmd.ctx != nil || cmd.execWasCalled {
			t.Error("same-instance execution must leave the existing lock untouched")
		}
	}
}

func TestLockableCommandRunContextWhenLocked(t *testing.T) {
	for _, behavior := range []LockedBehavior{FailWhenLocked, SkipWhenLocked} {
		for _, mode := range []string{
			"nonblocking",
			"lock-timeout",
			"parent-deadline",
			"parent-cancel",
		} {
			name := mode + "/fail"
			if behavior == SkipWhenLocked {
				name = mode + "/skip"
			}
			t.Run(
				name, func(t *testing.T) {
					cmd := &MockContextCommand{MockCommand: MockCommand{id: "locked-command"}}
					dir := t.TempDir()
					holder := NewLockableCommand(cmd, dir)
					if locked, err := holder.Lock(); err != nil || !locked {
						t.Fatalf("holder.Lock() = %v, %v", locked, err)
					}
					defer func() { _ = holder.Unlock() }()
					ctx := context.Background()
					timeout := time.Duration(0)
					var want error
					if behavior == FailWhenLocked {
						want = CommandLocked
					}
					switch mode {
					case "lock-timeout":
						timeout = 40 * time.Millisecond
					case "parent-deadline":
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, 40*time.Millisecond)
						defer cancel()
						timeout, want = 3*time.Second, context.DeadlineExceeded
					case "parent-cancel":
						var cancel context.CancelFunc
						ctx, cancel = context.WithCancel(ctx)
						defer cancel()
						timer := time.AfterFunc(40*time.Millisecond, cancel)
						defer timer.Stop()
						timeout, want = 3*time.Second, context.Canceled
					}
					helper := NewLockableCommandWithOptions(
						cmd, LockOptions{
							LockFileDirPath: dir, Timeout: timeout, WhenLocked: behavior,
						},
					)
					started := time.Now()
					result := runLockableContext(t, ctx, helper, io.Discard)
					if !errors.Is(result.Err, want) {
						t.Errorf("error = %v, want %v", result.Err, want)
					}
					if time.Since(started) > time.Second {
						t.Error("lock acquisition ignored the caller's cancellation/deadline")
					}
					if cmd.ctx != nil || cmd.execWasCalled {
						t.Error("command ran while its lock was held")
					}
					if !holder.fileLock.IsLocked() {
						t.Error("failed acquisition released another instance's lock")
					}
				},
			)
		}
	}
}

func TestLockableCommandRunContextCanceledBeforeExecution(t *testing.T) {
	for _, behavior := range []LockedBehavior{FailWhenLocked, SkipWhenLocked} {
		cmd := &MockLockableCommand{id: "canceled-command"}
		helper := NewLockableCommandWithOptions(
			cmd, LockOptions{
				LockFileDirPath: t.TempDir(), WhenLocked: behavior,
			},
		)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result := runLockableContext(t, ctx, helper, io.Discard)
		if !errors.Is(result.Err, context.Canceled) || cmd.executed {
			t.Errorf("canceled invocation: result = %+v, executed = %v", result, cmd.executed)
		}
		if err := helper.fileLock.Lock(); err != nil {
			t.Fatal(err)
		}
		if err := helper.fileLock.Lock(); !errors.Is(err, filelock.ErrAlreadyLocked) {
			t.Errorf("same-instance held lock = %v, want ErrAlreadyLocked", err)
		}
		_ = helper.Unlock()
	}
}
