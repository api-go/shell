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
