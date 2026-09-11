package shell_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"apigo.cc/go/shell"
)

func TestRunTimeout(t *testing.T) {
	// 测试超时功能
	_, err := shell.Run("sleep", []string{"2"}, &shell.Options{
		Timeout: 100 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestRunEnv(t *testing.T) {
	// 测试环境变量传递
	// 使用 printenv 检查是否设置了自定义变量
	res, err := shell.Run("printenv", []string{"TEST_VAR"}, &shell.Options{
		Env: []string{"TEST_VAR=hello_env"},
	})
	if err != nil {
		t.Errorf("Run with env failed: %v", err)
	}
	if strings.TrimSpace(string(res.Stdout)) != "hello_env" {
		t.Errorf("Expected 'hello_env', got: %s", res.Stdout)
	}
}

func TestRunCallbacks(t *testing.T) {
	// 测试 OnStdout 回调
	var stdoutCaptured bytes.Buffer
	_, err := shell.Run("echo", []string{"test_callback"}, &shell.Options{
		OnStdout: func(data []byte) {
			stdoutCaptured.Write(data)
		},
	})
	if err != nil {
		t.Errorf("Run with callback failed: %v", err)
	}
	if strings.TrimSpace(stdoutCaptured.String()) != "test_callback" {
		t.Errorf("Callback didn't capture expected output, got: %s", stdoutCaptured.String())
	}
}

func TestRunStderr(t *testing.T) {
	// 测试标准错误捕获
	// 使用 sh -c 显式输出到 stderr
	res, _ := shell.Run("sh", []string{"-c", "echo error_msg >&2; exit 1"}, nil)
	// 因为返回错误，所以检查 res
	if res.ProcessErr == nil {
		t.Error("Expected error, but got nil")
	}
	if strings.TrimSpace(string(res.Stderr)) != "error_msg" {
		t.Errorf("Expected 'error_msg', got: %s", res.Stderr)
	}
}

func TestPipeline(t *testing.T) {
	res, err := shell.Pipeline([][]string{{"echo", "hello", "world"}, {"grep", "hello"}}, nil)
	if err != nil {
		t.Errorf("Pipeline failed: %v", err)
	}
	if strings.TrimSpace(string(res.Stdout)) != "hello world" {
		t.Errorf("Expected 'hello world', got %s", res.Stdout)
	}
}

func TestRun(t *testing.T) {
	// 测试直接 Run
	res, err := shell.Run("echo", []string{"hello"}, nil)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
	if strings.TrimSpace(string(res.Stdout)) != "hello" {
		t.Errorf("Expected 'hello', got %s", res.Stdout)
	}
}

func TestRunAdvanced(t *testing.T) {
	// 测试 Timeout, Dir, Verbose
	_, err := shell.Run("sleep", []string{"0.1"}, &shell.Options{
		Timeout: 1 * time.Second,
		Verbose: true,
	})
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
}

func TestRunCommand(t *testing.T) {
	// 测试完整命令字符串 RunCommand
	res, err := shell.RunCommand("echo hello world", nil)
	if err != nil {
		t.Errorf("RunCommand failed: %v", err)
	}
	if strings.TrimSpace(string(res.Stdout)) != "hello world" {
		t.Errorf("Expected 'hello world', got %s", res.Stdout)
	}
}

func TestRunCommandWithPipe(t *testing.T) {
	// 测试包含管道的 RunCommand
	res, err := shell.RunCommand("echo hello world | grep hello", nil)
	if err != nil {
		t.Errorf("RunCommand with pipe failed: %v", err)
	}
	if strings.TrimSpace(string(res.Stdout)) != "hello world" {
		t.Errorf("Expected 'hello world', got %s", res.Stdout)
	}
}

func TestRunCommandMixedPipeAndChain(t *testing.T) {
	// 测试 | 和 && 混合使用
	// 期望：(echo hello | grep h) && (echo world | grep w)
	res, err := shell.RunCommand("echo hello | grep h && echo world | grep w", nil)
	if err != nil {
		t.Errorf("RunCommand with mixed operators failed: %v", err)
	}
	output := strings.TrimSpace(string(res.Stdout))
	if !strings.Contains(output, "hello") || !strings.Contains(output, "world") {
		t.Errorf("Expected 'hello' and 'world' in output, got: %s", output)
	}
}

func TestProcessKillWaitsAndIsConcurrentSafe(t *testing.T) {
	proc, err := shell.Start("sh", []string{"-c", "while :; do read line || :; done"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- proc.Kill()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("Kill failed: %v", err)
		}
	}
	if err := proc.Check(); err == nil {
		t.Fatal("process should be reaped before Kill returns")
	}
}

func TestProcessKillWaitsForEntireProcessGroup(t *testing.T) {
	ready := make(chan struct{}, 1)
	proc, err := shell.Start("sh", []string{"-c", "trap 'exit 0' TERM; sh -c 'trap \"\" TERM; echo ready; exec sleep 30' & wait"}, &shell.Options{OnStdout: func([]byte) {
		select {
		case ready <- struct{}{}:
		default:
		}
	}})
	if err != nil {
		t.Fatal(err)
	}
	<-ready

	grace := 100 * time.Millisecond
	started := time.Now()
	if err = proc.KillWithTimeout(grace); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < grace {
		t.Fatalf("KillWithTimeout returned after %v, before the process-group grace period %v", elapsed, grace)
	}
}
