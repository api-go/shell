# CHANGELOG - shell

## v1.5.7 (2026-09-05)
- **依赖对齐**: 升级 `cast` 至 `v1.5.6`、`jsmod` 至 `v1.5.4`。

## v1.5.6 (2026-08-24)
- **进程生命周期**:
  - `Process.Kill` 改为并发安全，并在返回前等待子进程回收完成。
  - 优雅退出超时后升级为 `SIGKILL`，避免旧进程占用端口时启动替代进程。
  - 异步进程 I/O 改用 `os/exec` 原生管道，避免退出期间 `Wait` 与管道关闭互相等待。
- **依赖更新**: 对齐 `cast v1.5.5`。

## v1.5.5 (2026-07-18)
- **依赖校验修复**: 更新 `go.sum` 为当前可验证的依赖 checksum。

## v1.5.4 (2026-06-21)
- **新增 `Start` 方法和 `Process` 结构体**:
  - `Start(name, args, opts)` 异步启动命令，立即返回 `*Process`，不等待命令完成。
  - `Process` 提供 `Kill`/`Read`/`Write`/`Check` 方法，支持子进程生命周期管理和实时 I/O 交互。
  - stdout/stderr 默认流式输出到父终端，同时缓冲（上限 512KB）以供 `Read` 读取。
  - `CatchSignal` 选项支持信号自动转发到子进程组。
  - `Kill` 优雅终止：先 SIGTERM，200ms 后 SIGKILL；Windows 直接 Kill。

## v1.5.3 (2026-06-21)
- **重构与错误堆栈支持**:
  - 重构 `js_export.go`，将 `Run` 和 `RunCommand` 注册为具名函数，并动态使用 `jsmod.MakeError` 包裹错误。
  - 升级 `jsmod` 依赖至 v1.5.3，`cast` 依赖至 v1.5.3。

## v1.5.2 (2026-06-11)
- **版本对齐**: 基础设施全局对齐 v1.5.2。

## v1.5.1 (2026-06-08)
- **JS 对齐**: 将所有注册到 `jsmod` 的导出方法名统一为 PascalCase（如 `Run`, `RunCommand`），以消除 JS 与 Go 调用体感上的摩擦。

## [v1.3.2] - 2026-05-30
- **新增**: 注册到 `jsmod`，提供 `run` 和 `runCommand` 能力，并标记为高危方法（unsafeList）。

## [v1.0.4] - 2026-05-02
- **重构**: 优化颜色常量命名（如 TextBlack, BgBlack）以符合 Go 现代规范。
- **重构**: 迁移测试用例至 'shell_test' 包以强化 API 测试封装。
- **修复**: 修复 RunCommand 中 '&&' 逻辑的短路行为，确保在子命令失败时中断执行。
- **优化**: 全面升级 README 文档，增加核心设计哲学、API 深度指南及使用示例。
- **配置**: 对齐 go.mod 至 Go 1.25.0。
