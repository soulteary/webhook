# Testing Guide

This document explains how to run tests, generate test coverage reports, and covers key testing scenarios.

## Quick Start

### Run All Tests

```bash
go test ./...
```

### Run Tests for Specific Packages

```bash
# Run server package tests
go test ./internal/server/...

# Run hook package tests
go test ./internal/hook/...
```

### Run Critical Scenario Tests

```bash
# Run concurrency tests
go test -v ./internal/server -run TestConcurrent

# Run security tests
go test -v ./internal/server -run TestCommand|TestPath

# Run performance tests
go test -v ./internal/server -run TestStress|TestLoad
```

## Test Coverage

### Using the Test Coverage Script

We provide a convenient script `test-coverage.sh` to generate test coverage reports:

```bash
# Run all tests and generate coverage report
./test-coverage.sh all

# Run only server package tests
./test-coverage.sh server

# Run only critical scenario tests
./test-coverage.sh critical

# Generate HTML coverage report
./test-coverage.sh html

# View function-level coverage
./test-coverage.sh func

# Clean coverage files
./test-coverage.sh clean
```

### Manual Coverage Report Generation

```bash
# 1. Run tests and generate coverage file
go test -coverprofile=coverage.out -covermode=atomic ./...

# 2. View coverage statistics
go tool cover -func=coverage.out

# 3. Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

## Test Scenario Categories

### 1. Error Path Testing

Tests various error conditions to ensure the system handles errors gracefully:

- **File Not Found**: `TestMakeSureCallable_FileNotExists`
- **Permission Errors**: `TestMakeSureCallable_PermissionDenied`
- **Working Directory Not Found**: `TestMakeSureCallable_WorkingDirectoryNotExists`
- **Command Timeout**: `TestHandleHook_CommandTimeout`
- **Configuration File Errors**: `TestCreateHookHandler_ConfigFileError`

### 2. Concurrency Scenario Testing

Tests system stability and correctness under high concurrency:

- **Concurrent Execution of Same Hook**: `TestConcurrentHookExecution_SameHook`
- **Concurrent File Operations**: `TestConcurrentHookExecution_FileOperations`
- **Resource Contention**: `TestConcurrentHookExecution_ResourceContention`

### 3. Security-Related Testing

Tests system security protection capabilities:

- **Command Injection Prevention**: `TestCommandInjection_Prevention`
- **Path Traversal Prevention**: `TestPathTraversal_Prevention`
- **Strict Mode Validation**: `TestCommandValidator_StrictMode`
- **Path Whitelist**: `TestCommandValidator_PathWhitelist`
- **Argument Length Limits**: `TestCommandValidator_ArgLengthLimits`
- **Special Character Handling**: `TestSpecialCharacters_Handling`

### 4. Performance Testing

Tests system performance and scalability:

- **Benchmark Tests**: `BenchmarkHookExecution`, `BenchmarkConcurrentHookExecution`
- **Load Testing**: `TestLoadTest_MultipleHooks`
- **Stress Testing**: `TestStressTest_HighConcurrency`
- **Memory Leak Testing**: `TestMemoryLeak_RepeatedExecutions`

## Running Specific Tests

### Run by Name

```bash
# Run a single test
go test -v ./internal/server -run TestConcurrentHookExecution_SameHook

# Run tests matching a pattern
go test -v ./internal/server -run TestCommand
```

### Run Benchmark Tests

```bash
# Run all benchmark tests
go test -bench=. ./internal/server

# Run a specific benchmark test
go test -bench=BenchmarkHookExecution ./internal/server

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof ./internal/server
```

## Test Coverage Goals

We recommend maintaining the following test coverage targets:

- **Overall Coverage**: ≥ 80%
- **Critical Path Coverage**: ≥ 90%
- **Security-Related Code Coverage**: ≥ 95%

### View Coverage Details

```bash
# Generate HTML report and open
./test-coverage.sh html
open coverage.html  # macOS
# or
xdg-open coverage.html  # Linux
```

## Continuous Integration

In CI/CD pipelines, you can use the following commands:

```bash
# Run tests and check coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}'

# Exit with non-zero code if coverage is below threshold
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
  echo "Coverage $COVERAGE% is below 80%"
  exit 1
fi
```

## Testing Best Practices

1. **Test Isolation**: Each test should be independent and not depend on the execution order of other tests
2. **Resource Cleanup**: Use `t.TempDir()` to create temporary directories that are automatically cleaned up after tests
3. **Error Handling**: Tests should verify error conditions, not just success paths
4. **Concurrency Safety**: Concurrency tests should verify data races and deadlocks
5. **Performance Benchmarks**: Regularly run benchmark tests to monitor performance regressions

## Troubleshooting

### Test Failures

If tests fail, you can:

1. Use the `-v` flag to view detailed output
2. Use `-run` to run specific tests for debugging
3. Check error messages in test logs

### Inaccurate Coverage

If coverage reports are inaccurate:

1. Ensure you use `-covermode=atomic` for concurrent tests
2. Check for untested code branches
3. Verify that tests actually execute the target code

## Related Documentation

- [API Reference](./API-Reference.md)
- [Security Best Practices](./Security-Best-Practices.md)
- [Performance Tuning](./Performance-Tuning.md)

