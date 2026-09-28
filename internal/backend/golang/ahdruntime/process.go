package ahdruntime

// The v2.5.0 Process standard module: run one executable with an explicit
// argument list, bounded in time and in output. Built only on the Go standard
// library (os/exec, context), so this file is emitted verbatim into native
// programs and called directly by the evaluator.
//
// Security model:
//   - there is no shell. The executable is started directly with its
//     arguments as separate strings; no /bin/sh -c, cmd.exe /C, or PowerShell
//     is ever involved, so shell metacharacters in arguments are literal text.
//     On Windows, .bat and .cmd files are refused because CreateProcess would
//     run them through cmd.exe;
//   - every run has a finite timeout and a finite combined stdout+stderr
//     budget. Exceeding either kills the child (its whole process group on
//     Unix), reaps it, and fails with a ProcessError; output is never silently
//     truncated;
//   - standard input is the null device; the environment is inherited
//     unchanged and never exposed or modified;
//   - the module decides nothing about which programs are safe to run. A
//     privileged worker must apply its own allowlist before calling it.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Process bounds. Defaults apply when a call omits timeoutSeconds or
// maxOutputBytes; explicit values must stay within the hard limits.
const (
	AhdProcessDefaultTimeoutSeconds = int64(30)
	AhdProcessHardMaxTimeoutSeconds = int64(3600)
	AhdProcessDefaultMaxOutputBytes = int64(1) << 20 // 1 MiB, stdout+stderr combined
	AhdProcessHardMaxOutputBytes    = int64(64) << 20
	// ahdProcessPipeGrace is how long Wait keeps reading output after the
	// child exited or was killed, so a descendant that inherited the pipes
	// cannot hold the caller forever.
	ahdProcessPipeGrace = 2 * time.Second
)

func ahdProcessFail(command, reason string) error {
	return errors.New("run " + strconv.Quote(command) + " failed: " + reason)
}

// ahdProcessBudget is the combined output allowance of one run.
type ahdProcessBudget struct {
	mutex    sync.Mutex
	left     int64
	exceeded bool
	onExceed func()
}

var errAhdProcessOutputLimit = errors.New("process output limit exceeded")

// ahdProcessStream collects one of stdout or stderr against the shared
// budget. Once the budget is exhausted every further write fails, which stops
// exec's copying goroutine, and the child is killed.
type ahdProcessStream struct {
	budget *ahdProcessBudget
	buffer bytes.Buffer
}

func (stream *ahdProcessStream) Write(data []byte) (int, error) {
	budget := stream.budget
	budget.mutex.Lock()
	defer budget.mutex.Unlock()
	if budget.exceeded || int64(len(data)) > budget.left {
		if !budget.exceeded {
			budget.exceeded = true
			budget.onExceed()
		}
		return 0, errAhdProcessOutputLimit
	}
	budget.left -= int64(len(data))
	stream.buffer.Write(data)
	return len(data), nil
}

func (stream *ahdProcessStream) text() string {
	return strings.ToValidUTF8(stream.buffer.String(), "�")
}

// ahdProcessResultData is the whole public surface of one ProcessResult.
type ahdProcessResultData struct {
	ExitCode int64  `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// ProcessRun starts command with args and waits for it, returning the
// encoded ProcessResult. dir is the child's working directory ("" inherits
// the caller's); a relative command containing a path separator is resolved
// against it, and a bare name is looked up in PATH (never in the current
// directory). A non-zero exit status is an ordinary result, not an error.
func ProcessRun(command string, args []string, timeoutSeconds, maxOutputBytes int64, dir string) (string, error) {
	if command == "" {
		return "", ahdProcessFail(command, "the command is empty")
	}
	if strings.ContainsRune(command, 0) {
		return "", ahdProcessFail(command, "the command contains a NUL byte")
	}
	for index, argument := range args {
		if strings.ContainsRune(argument, 0) {
			return "", ahdProcessFail(command, fmt.Sprintf("argument %d contains a NUL byte", index+1))
		}
	}
	if timeoutSeconds < 1 || timeoutSeconds > AhdProcessHardMaxTimeoutSeconds {
		return "", ahdProcessFail(command, fmt.Sprintf("timeoutSeconds must be between 1 and %d; received %d", AhdProcessHardMaxTimeoutSeconds, timeoutSeconds))
	}
	if maxOutputBytes < 1 || maxOutputBytes > AhdProcessHardMaxOutputBytes {
		return "", ahdProcessFail(command, fmt.Sprintf("maxOutputBytes must be between 1 and %d; received %d", AhdProcessHardMaxOutputBytes, maxOutputBytes))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	if dir != "" && !filepath.IsAbs(command) && strings.ContainsAny(command, `/\`) {
		// exec resolved a relative path against the caller's directory;
		// resolve it against dir, which is where the child will run.
		cmd = exec.CommandContext(ctx, filepath.Join(dir, command), args...)
		cmd.Dir = dir
	}
	if cmd.Err != nil {
		return "", ahdProcessFail(command, ahdProcessStartReason(cmd.Err))
	}
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(cmd.Path)) {
		case ".bat", ".cmd":
			return "", ahdProcessFail(command, "batch files run through cmd.exe; Process.run never uses a shell")
		}
	}
	budget := &ahdProcessBudget{left: maxOutputBytes, onExceed: cancel}
	stdout := &ahdProcessStream{budget: budget}
	stderr := &ahdProcessStream{budget: budget}
	cmd.Stdin = nil // the null device
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = ahdProcessPipeGrace
	ahdProcessConfigure(cmd)

	if err := cmd.Start(); err != nil {
		return "", ahdProcessFail(command, ahdProcessStartReason(err))
	}
	waitErr := cmd.Wait() // always reaps the child

	budget.mutex.Lock()
	exceeded := budget.exceeded
	budget.mutex.Unlock()
	if exceeded {
		return "", ahdProcessFail(command, fmt.Sprintf("output limit exceeded: stdout and stderr together produced more than %d bytes (maxOutputBytes); the process was terminated", maxOutputBytes))
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", ahdProcessFail(command, fmt.Sprintf("timed out after %d seconds (timeoutSeconds); the process was terminated", timeoutSeconds))
	}
	exitCode := int64(0)
	if cmd.ProcessState != nil {
		exitCode = int64(cmd.ProcessState.ExitCode())
	}
	if waitErr != nil {
		var exitError *exec.ExitError
		if !errors.As(waitErr, &exitError) && !errors.Is(waitErr, exec.ErrWaitDelay) {
			return "", ahdProcessFail(command, "waiting for the process failed: "+ahdFSReason(waitErr))
		}
	}
	encoded, err := json.Marshal(ahdProcessResultData{ExitCode: exitCode, Stdout: stdout.text(), Stderr: stderr.text()})
	if err != nil {
		return "", ahdProcessFail(command, err.Error())
	}
	return string(encoded), nil
}

// ahdProcessStartReason phrases a launch failure without Go's error prefixes.
func ahdProcessStartReason(err error) string {
	switch {
	case errors.Is(err, exec.ErrNotFound):
		return "executable not found"
	case errors.Is(err, exec.ErrDot):
		return "executable not found (a program in the current directory is never run by bare name; use an explicit path)"
	case errors.Is(err, fs.ErrNotExist):
		return "executable or working directory not found"
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	}
	return ahdFSReason(err)
}

func ahdProcessDecode(data string) (ahdProcessResultData, error) {
	var result ahdProcessResultData
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return result, errors.New("ProcessResult storage is corrupted")
	}
	return result, nil
}

func ProcessResultExitCode(data string) (int64, error) {
	result, err := ahdProcessDecode(data)
	return result.ExitCode, err
}

func ProcessResultStdout(data string) (string, error) {
	result, err := ahdProcessDecode(data)
	return result.Stdout, err
}

func ProcessResultStderr(data string) (string, error) {
	result, err := ahdProcessDecode(data)
	return result.Stderr, err
}

// --- native wrappers: raise the program's ProcessError ---

func ahdProcessRaise(class *AhdClass, err error) {
	if err != nil {
		AhdRaiseClass(class, err.Error())
	}
}

func AhdProcessRun(class *AhdClass, command string, args *AhdList[string], timeoutSeconds, maxOutputBytes int64) string {
	var arguments []string
	if args != nil {
		arguments = args.Snapshot()
	}
	result, err := ProcessRun(command, arguments, timeoutSeconds, maxOutputBytes, "")
	ahdProcessRaise(class, err)
	return result
}

func AhdProcessResultExitCode(class *AhdClass, data string) int64 {
	value, err := ProcessResultExitCode(data)
	ahdProcessRaise(class, err)
	return value
}

func AhdProcessResultStdout(class *AhdClass, data string) string {
	value, err := ProcessResultStdout(data)
	ahdProcessRaise(class, err)
	return value
}

func AhdProcessResultStderr(class *AhdClass, data string) string {
	value, err := ProcessResultStderr(data)
	ahdProcessRaise(class, err)
	return value
}
