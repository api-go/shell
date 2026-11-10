# 测试报告 (shell)

## v1.5.8 验证
- 验证 `KillWithTimeout` 等待完整进程组并遵守自定义优雅期限。
- `go test -v ./...` 与 `go test -bench=. ./...` 通过。

## v1.5.7 验证
- 独立模块解析、全量测试与基准测试均通过。
- 验证 `Process.Kill` 并发调用只执行一次终止流程。
- 验证 `Kill` 返回前子进程已完成回收。
- 全量测试、竞态测试和基准测试均通过。

## v1.5.5 验证
- 独立模块解析、全量测试与基准测试均通过。

## 测试场景覆盖
- 颜色常量命名与样式渲染测试
- 命令行超时管理
- 环境变量注入
- 实时流式回调 (OnStdout)
- 标准错误流捕获
- 管道命令执行 (Pipeline)
- 完整命令字符串支持（包含 '|' 与 '&&' 逻辑）
- '&&' 短路逻辑验证（失败中断）
- 异步进程启动与管理 (Start / Process)
- 进程生命周期控制 (Kill / Read / Write / Check)
- 并发调用 Kill 时只执行一次终止流程，并在返回前完成子进程回收

## Benchmark 结果
- 目前暂未添加复杂基准测试，Shell 模块执行受限于 OS 系统调用开销。

## GitHub migration validation (pending release)

Module paths and internal dependency versions are migrated. Development verification uses GOWORK=off and explicit sibling replacements; remove these replacements before release. See migration validation logs for actual results. Existing unrelated working changes are preserved and require separate review.

API inventory verification: `GOWORK=off go doc` succeeded; output retained in the migration validation directory. Source exports were preserved by the path migration.

## GitHub release verification

GOWORK=off go test -v ./... and go test -bench=. ./... passed using published GitHub dependencies, without local replacements. go mod verify passed.
