package shell
import (
	"apigo.cc/go/cast"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// CommandResult 封装命令执行后的输出与错误状态
type CommandResult struct {
	Stdout     []byte
	Stderr     []byte
	ExitCode   int
	ProcessErr error
}

// Options 定义执行时的配置，零值即默认行为
type Options struct {
	Env         []string
	Stdin       io.Reader
	CatchSignal bool
	Timeout     time.Duration
	Dir         string
	Verbose     bool
	OnStdout    func(data []byte)
	OnStderr    func(data []byte)
}

// Run 执行命令并返回包含 stdout/stderr 的结果结构体
func Run(name string, args []string, opts *Options) (*CommandResult, error) {
	var ctx context.Context
	var cancel context.CancelFunc

	if opts != nil && opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	// 处理流式实时回调
	var stdoutW io.Writer = &stdout
	var stderrW io.Writer = &stderr
	if opts != nil {
		if opts.OnStdout != nil {
			stdoutW = io.MultiWriter(stdoutW, writerFunc(opts.OnStdout))
		}
		if opts.OnStderr != nil {
			stderrW = io.MultiWriter(stderrW, writerFunc(opts.OnStderr))
		}
	}
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	if opts != nil {
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if opts.Verbose {
			fmt.Println(Style(TextCyan, "Exec:", name, strings.Join(args, " ")))
		}
		if opts.Env != nil {
			cmd.Env = append(os.Environ(), opts.Env...)
		}
		if opts.Stdin != nil {
			cmd.Stdin = opts.Stdin
		}
		if opts.CatchSignal {
			setupSignalHandling(cmd)
		}
	}

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	err := cmd.Run()

	res := &CommandResult{
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
		ProcessErr: err,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			res.ExitCode = -1
			return res, fmt.Errorf("command timed out: %w", err)
		} else if exitError, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitError.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, err
	}

	return res, nil
}

// InteractiveRun 直接将标准输入、输出和错误流绑定到进程
func InteractiveRun(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type writerFunc func([]byte)
func (w writerFunc) Write(p []byte) (n int, err error) {
	w(p)
	return len(p), nil
}

// Pipeline 执行一组命令，每个命令由 name 和 args 组成
func Pipeline(commands [][]string, opts *Options) (*CommandResult, error) {
	if len(commands) == 0 {
		return nil, fmt.Errorf("no commands provided")
	}

	var cmds []*exec.Cmd
	for _, cmdParts := range commands {
		c := exec.Command(cmdParts[0], cmdParts[1:]...)
		if opts != nil && opts.Dir != "" {
			c.Dir = opts.Dir
		}
		cmds = append(cmds, c)
	}

	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			return nil, err
		}
		cmds[i+1].Stdin = stdout
	}

	var lastStdout bytes.Buffer
	cmds[len(cmds)-1].Stdout = &lastStdout

	for _, c := range cmds {
		if err := c.Start(); err != nil {
			return nil, err
		}
	}

	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			return &CommandResult{ExitCode: -1}, err
		}
	}

	return &CommandResult{Stdout: lastStdout.Bytes()}, nil
}

// PipelineCommands 支持命令字符串切片
func PipelineCommands(commands []string, opts *Options) (*CommandResult, error) {
	var cmds [][]string
	for _, cmdStr := range commands {
		cmds = append(cmds, cast.SplitArgs(cmdStr))
	}
	return Pipeline(cmds, opts)
}

// RunCommand 支持完整命令字符串（如 "ls -la | grep go" 或 "cmd1 && cmd2"），支持 pipe 和 AND 逻辑
func RunCommand(command string, opts *Options) (*CommandResult, error) {
	// 处理 && 逻辑
	if strings.Contains(command, "&&") {
		parts := strings.Split(command, "&&")
		var lastResult *CommandResult
		var fullStdout []byte
		for _, p := range parts {
			res, err := RunCommand(strings.TrimSpace(p), opts)
			if err != nil {
				return res, err
			}
			lastResult = res
			fullStdout = append(fullStdout, res.Stdout...)
		}
		if lastResult != nil {
			lastResult.Stdout = fullStdout
		}
		return lastResult, nil
	}

	// 处理 | 逻辑
	if strings.Contains(command, "|") {
		parts := strings.Split(command, "|")
		var cmds []string
		for _, p := range parts {
			cmds = append(cmds, strings.TrimSpace(p))
		}
		return PipelineCommands(cmds, opts)
	}

	args := cast.SplitArgs(command)
	if len(args) == 0 {
		return &CommandResult{ExitCode: -1}, fmt.Errorf("empty command")
	}
	return Run(args[0], args[1:], opts)
}

func setupSignalHandling(cmd *exec.Cmd) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		sig := <-sigCh
		if cmd.Process != nil {
			if runtime.GOOS == "windows" {
				_ = cmd.Process.Kill()
			} else {
				pgid, err := syscall.Getpgid(cmd.Process.Pid)
				if err == nil {
					_ = syscall.Kill(-pgid, sig.(syscall.Signal))
				}
			}
		}
	}()
}

// Process represents a running command started by Start.
// It provides methods to manage the subprocess lifecycle and I/O.
type Process struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdoutBuf bytes.Buffer
	stderrBuf bytes.Buffer
	mu        sync.Mutex
	done      chan struct{}
}

// Start starts a command and returns immediately, unlike Run which waits for completion.
// The returned Process provides Kill/Read/Write/Check for lifecycle management.
// If CatchSignal is set in opts, OS signals are automatically forwarded to the child.
// By default, stdout/stderr are streamed to the parent's terminal in addition to
// being buffered for Read calls.
func Start(name string, args []string, opts *Options) (*Process, error) {
	cmd := exec.Command(name, args...)

	if opts != nil {
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if opts.Verbose {
			fmt.Println(Style(TextCyan, "Start:", name, strings.Join(args, " ")))
		}
		if opts.Env != nil {
			cmd.Env = append(os.Environ(), opts.Env...)
		}
	}

	stdinR, stdinW := io.Pipe()
	cmd.Stdin = stdinR

	stdoutR, stdoutW := io.Pipe()
	cmd.Stdout = stdoutW

	stderrR, stderrW := io.Pipe()
	cmd.Stderr = stderrW

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		stdinW.Close()
		stdoutW.Close()
		stderrW.Close()
		return nil, err
	}

	p := &Process{
		cmd:   cmd,
		stdin: stdinW,
		done:  make(chan struct{}),
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				p.mu.Lock()
				if p.stdoutBuf.Len() > 512*1024 {
					p.stdoutBuf.Next(p.stdoutBuf.Len() / 2)
				}
				p.stdoutBuf.Write(buf[:n])
				p.mu.Unlock()
				if opts != nil && opts.OnStdout != nil {
					opts.OnStdout(buf[:n])
				} else {
					os.Stdout.Write(buf[:n])
				}
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrR.Read(buf)
			if n > 0 {
				p.mu.Lock()
				if p.stderrBuf.Len() > 512*1024 {
					p.stderrBuf.Next(p.stderrBuf.Len() / 2)
				}
				p.stderrBuf.Write(buf[:n])
				p.mu.Unlock()
				if opts != nil && opts.OnStderr != nil {
					opts.OnStderr(buf[:n])
				} else {
					os.Stderr.Write(buf[:n])
				}
			}
			if err != nil {
				break
			}
		}
	}()

	if opts != nil && opts.CatchSignal {
		setupSignalHandling(cmd)
	}

	go func() {
		cmd.Wait()
		stdinW.Close()
		stdoutW.Close()
		stderrW.Close()
		close(p.done)
	}()

	return p, nil
}

// Kill terminates the process group. SIGTERM first for graceful shutdown,
// then SIGKILL after 200ms. On Windows, uses os.Process.Kill directly.
func (p *Process) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return p.cmd.Process.Kill()
	}
	pgid, err := syscall.Getpgid(p.cmd.Process.Pid)
	if err != nil {
		return p.cmd.Process.Kill()
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	time.Sleep(200 * time.Millisecond)
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// Read reads buffered stdout data from the process. The buffer retains up to
// 512KB of recent output; older data is discarded when the limit is exceeded.
func (p *Process) Read(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stdoutBuf.Read(data)
}

// Write sends data to the process's stdin.
func (p *Process) Write(data []byte) (int, error) {
	return p.stdin.Write(data)
}

// Check returns nil if the process is still running.
// If the process has exited, it returns an error with exit details.
func (p *Process) Check() error {
	select {
	case <-p.done:
		if p.cmd.ProcessState != nil {
			if code := p.cmd.ProcessState.ExitCode(); code != 0 {
				return fmt.Errorf("process exited with code %d", code)
			}
		}
		return fmt.Errorf("process completed")
	default:
		return nil
	}
}
