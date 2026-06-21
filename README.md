# Shell Utilities

本模块旨在提供一套极致精简、高性能且符合 POSIX 语义的终端与 Shell 基础设施。它不仅是简单的命令执行包装，更提供了流式实时回调、上下文管理及命令链式执行能力。

## 核心设计哲学

- **语义对齐**: 针对 Shell 交互行为（如 `|` 管道，`&&` 短路逻辑）进行语义一致性实现。
- **安全优先**: 默认支持进程组管理 (`pgid`)，确保中断信号能精准传播至整个命令链条，避免产生僵尸进程。
- **内存透明**: 通过 `bytes.Buffer` 和流式 IO 接口管理大规模输出，避免内存溢出风险。

## API 指南

### 终端渲染 (Color)

用于生成符合终端 ANSI 标准的样式字符串。

- **`Style(a ...any) string`**: 核心渲染引擎。自动合并样式常量与内容，并处理 Windows 系统下的兼容性（剥离样式）。
- **颜色定义**:
  - 前景色: `TextBlack`, `TextRed`, `TextGreen`, `TextYellow`, `TextBlue`, `TextMagenta`, `TextCyan`, `TextWhite`
  - 背景色: `BgBlack`, `BgRed`, `BgGreen`, `BgYellow`, `BgBlue`, `BgMagenta`, `BgCyan`, `BgWhite`
  - 属性: `Reset`, `Bold`, `Dim`, `Italic`, `Underline`, `Blink`, `Reverse`, `Hidden`
- **高阶快捷函数**: 
  - 颜色快捷调用: `Red("error")`, `Green("success")` 等。
  - 背景色快捷调用（B开头）: `BRed("fatal")` (带白色前景色)。

### 命令执行 (Run)

#### 数据结构
- **`Options`**: 配置执行行为
  - `Env []string`: 环境变量增量
  - `Stdin io.Reader`: 自定义输入源
  - `CatchSignal bool`: 是否开启信号捕获，确保进程树生命周期一致
  - `Timeout time.Duration`: 执行超时限制
  - `Dir string`: 设置工作目录
  - `Verbose bool`: 是否开启详细执行日志（终端高亮输出）
  - `OnStdout func([]byte)`: 实时 stdout 捕获回调
  - `OnStderr func([]byte)`: 实时 stderr 捕获回调

- **`CommandResult`**: 命令执行反馈
  - `Stdout []byte`: 标准输出内容
  - `Stderr []byte`: 标准错误输出内容
  - `ExitCode int`: 进程退出码（0 成功，非 0 失败）
  - `ProcessErr error`: 执行层面的错误信息

#### 函数接口
- **`Run(name string, args []string, opts *Options) (*CommandResult, error)`**: 基础执行。
- **`RunCommand(command string, opts *Options) (*CommandResult, error)`**: 支持解析 `|` 管道与 `&&` 短路逻辑的复合命令执行。
- **`Pipeline(commands [][]string, opts *Options) (*CommandResult, error)`**: 执行多命令并行管道，高性能 IO 传输。

### 异步进程管理 (Start)

#### 数据结构
- **`Process`**: 代表由 `Start` 启动的运行中命令
  - `Kill() error`: 终止进程组（SIGTERM → 200ms → SIGKILL），Windows 直接 Kill
  - `Read(data []byte) (int, error)`: 读取缓冲的 stdout 数据（上限 512KB，超出丢弃旧数据）
  - `Write(data []byte) (int, error)`: 向进程 stdin 写入数据
  - `Check() error`: 检查进程是否仍在运行，已退出则返回错误

#### 函数接口
- **`Start(name string, args []string, opts *Options) (*Process, error)`**: 异步启动命令，立即返回 `*Process`。stdout/stderr 默认流式输出到终端，可通过 `opts.OnStdout`/`opts.OnStderr` 自定义回调。

## 使用示例

```go
// 1. 带超时和实时回调的执行
res, err := shell.Run("grep", []string{"pattern", "file.log"}, &shell.Options{
    Timeout: 5 * time.Second,
    OnStdout: func(data []byte) { fmt.Printf("Output: %s", data) },
})

// 2. 管道与短路逻辑组合
res, err := shell.RunCommand("cat file.txt | grep 'foo' && echo 'found'", nil)

// 3. 异步启动长期运行的服务
proc, err := shell.Start("go", []string{"run", "server.go"}, &shell.Options{
    Dir: "./app",
    OnStdout: func(data []byte) { log.Print(string(data)) },
})
if err != nil { panic(err) }
defer proc.Kill()
// 写入 stdin
proc.Write([]byte("input\n"))
// 检查运行状态
if err := proc.Check(); err != nil {
    fmt.Println("process stopped:", err)
}
```

## 测试与质量保证
请参考 [TEST.md](./TEST.md) 获取覆盖率与 Benchmark 数据。
