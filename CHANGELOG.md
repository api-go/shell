# CHANGELOG - shell

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
