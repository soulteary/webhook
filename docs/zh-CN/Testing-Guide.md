# 测试指南

本文档介绍如何运行测试、生成测试覆盖率报告，以及关键测试场景。

## 快速开始

### 运行所有测试

```bash
go test ./...
```

### 运行特定包的测试

```bash
# 运行 server 包测试
go test ./internal/server/...

# 运行 hook 包测试
go test ./internal/hook/...
```

### 运行关键场景测试

```bash
# 运行并发测试
go test -v ./internal/server -run TestConcurrent

# 运行安全测试
go test -v ./internal/server -run TestCommand|TestPath

# 运行性能测试
go test -v ./internal/server -run TestStress|TestLoad
```

## 测试覆盖率

### 使用测试覆盖率脚本

我们提供了一个便捷的脚本 `test-coverage.sh` 来生成测试覆盖率报告：

```bash
# 运行所有测试并生成覆盖率报告
./test-coverage.sh all

# 只运行 server 包测试
./test-coverage.sh server

# 只运行关键场景测试
./test-coverage.sh critical

# 生成 HTML 覆盖率报告
./test-coverage.sh html

# 查看函数级别覆盖率
./test-coverage.sh func

# 清理覆盖率文件
./test-coverage.sh clean
```

### 手动生成覆盖率报告

```bash
# 1. 运行测试并生成覆盖率文件
go test -coverprofile=coverage.out -covermode=atomic ./...

# 2. 查看覆盖率统计
go tool cover -func=coverage.out

# 3. 生成 HTML 报告
go tool cover -html=coverage.out -o coverage.html
```

## 测试场景分类

### 1. 错误路径测试

测试各种错误情况，确保系统能够优雅地处理错误：

- **文件不存在**: `TestMakeSureCallable_FileNotExists`
- **权限错误**: `TestMakeSureCallable_PermissionDenied`
- **工作目录不存在**: `TestMakeSureCallable_WorkingDirectoryNotExists`
- **命令超时**: `TestHandleHook_CommandTimeout`
- **配置文件错误**: `TestCreateHookHandler_ConfigFileError`

### 2. 并发场景测试

测试系统在高并发情况下的稳定性和正确性：

- **同一 Hook 并发执行**: `TestConcurrentHookExecution_SameHook`
- **并发文件操作**: `TestConcurrentHookExecution_FileOperations`
- **资源竞争**: `TestConcurrentHookExecution_ResourceContention`

### 3. 安全相关测试

测试系统的安全防护能力：

- **命令注入防护**: `TestCommandInjection_Prevention`
- **路径遍历防护**: `TestPathTraversal_Prevention`
- **严格模式验证**: `TestCommandValidator_StrictMode`
- **路径白名单**: `TestCommandValidator_PathWhitelist`
- **参数长度限制**: `TestCommandValidator_ArgLengthLimits`
- **特殊字符处理**: `TestSpecialCharacters_Handling`

### 4. 性能测试

测试系统的性能和可扩展性：

- **基准测试**: `BenchmarkHookExecution`, `BenchmarkConcurrentHookExecution`
- **负载测试**: `TestLoadTest_MultipleHooks`
- **压力测试**: `TestStressTest_HighConcurrency`
- **内存泄漏测试**: `TestMemoryLeak_RepeatedExecutions`

## 运行特定测试

### 按名称运行

```bash
# 运行单个测试
go test -v ./internal/server -run TestConcurrentHookExecution_SameHook

# 运行匹配模式的测试
go test -v ./internal/server -run TestCommand
```

### 运行基准测试

```bash
# 运行所有基准测试
go test -bench=. ./internal/server

# 运行特定基准测试
go test -bench=BenchmarkHookExecution ./internal/server

# 生成 CPU profile
go test -bench=. -cpuprofile=cpu.prof ./internal/server
```

## 测试覆盖率目标

我们建议保持以下测试覆盖率目标：

- **整体覆盖率**: ≥ 80%
- **关键路径覆盖率**: ≥ 90%
- **安全相关代码覆盖率**: ≥ 95%

### 查看覆盖率详情

```bash
# 生成 HTML 报告并打开
./test-coverage.sh html
open coverage.html  # macOS
# 或
xdg-open coverage.html  # Linux
```

## 持续集成

在 CI/CD 流程中，可以使用以下命令：

```bash
# 运行测试并检查覆盖率
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}'

# 如果覆盖率低于阈值，退出码不为 0
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
  echo "覆盖率 $COVERAGE% 低于 80%"
  exit 1
fi
```

## 测试最佳实践

1. **测试隔离**: 每个测试应该是独立的，不依赖其他测试的执行顺序
2. **清理资源**: 使用 `t.TempDir()` 创建临时目录，测试结束后自动清理
3. **错误处理**: 测试应该验证错误情况，而不仅仅是成功路径
4. **并发安全**: 并发测试应该验证数据竞争和死锁
5. **性能基准**: 定期运行基准测试，监控性能回归

## 故障排查

### 测试失败

如果测试失败，可以：

1. 使用 `-v` 标志查看详细输出
2. 使用 `-run` 运行特定测试进行调试
3. 检查测试日志中的错误信息

### 覆盖率不准确

如果覆盖率报告不准确：

1. 确保使用 `-covermode=atomic` 进行并发测试
2. 检查是否有未测试的代码分支
3. 验证测试是否真正执行了目标代码

## 相关文档

- [API 参考](./API-Reference.md)
- [安全最佳实践](./Security-Best-Practices.md)
- [性能调优](./Performance-Tuning.md)

