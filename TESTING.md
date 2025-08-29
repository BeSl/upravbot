# CupBot Testing Guide

This document describes the testing strategy and setup for the CupBot project.

## Test Structure

The project includes comprehensive testing across all core components:

### Unit Tests
- **Config Package** (`internal/config/config_test.go`)
  - Configuration loading and parsing
  - Environment variable handling
  - User ID validation
  - Admin user checking

- **Database Package** (`internal/database/database_test.go`)
  - User CRUD operations
  - Command history management
  - Session tracking
  - Statistics generation
  - Database cleanup operations

- **Auth Middleware** (`internal/auth/middleware_test.go`)
  - User authorization flows
  - Admin privilege checking
  - Command logging
  - User management operations
  - Safety checks for admin operations

- **System Service** (`internal/system/service_test.go`)
  - System information retrieval
  - Data structure validation
  - Utility function testing
  - Performance benchmarks

### Integration Tests
- **Bot Handlers** (`internal/bot/bot_test.go`)
  - Command handler logic
  - Admin vs regular user permissions
  - Input validation and parsing
  - Helper function testing

## Running Tests

### Manual Test Execution

#### Run All Tests
```bash
go test ./internal/...
```

#### Run Tests with Coverage
```bash
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html
```

#### Run Tests with Race Detection
```bash
go test -race ./internal/...
```

#### Run Benchmarks
```bash
go test -bench=. -benchmem ./internal/...
```

#### Run Specific Package Tests
```bash
go test -v ./internal/config
go test -v ./internal/database
go test -v ./internal/auth
go test -v ./internal/system
go test -v ./internal/bot
```

### Automated Test Runner

Use the provided test runner script for comprehensive testing:

```bash
run-tests.bat
```

This script will:
- Run all unit tests with coverage
- Generate HTML coverage reports
- Run benchmarks
- Check for race conditions
- Create a comprehensive test report

## Test Coverage Goals

The project aims for high test coverage across all packages:

- **Config**: 95%+ coverage
- **Database**: 90%+ coverage  
- **Auth**: 90%+ coverage
- **System**: 85%+ coverage (limited by OS dependencies)
- **Bot**: 80%+ coverage (limited by Telegram API dependencies)

## Continuous Integration

### GitHub Actions Workflow

The CI pipeline (`.github/workflows/ci.yml`) includes:

1. **Test Job**
   - Runs on Windows (target platform)
   - Executes all tests with race detection
   - Generates coverage reports
   - Uploads coverage to Codecov

2. **Build Job**
   - Builds the application
   - Tests the executable
   - Archives the binary

3. **Security Job**
   - Runs Gosec security scanner
   - Checks for common security issues

4. **Lint Job**
   - Runs golangci-lint
   - Ensures code quality standards

5. **Release Job**
   - Creates releases on tag push
   - Builds optimized binaries

### Code Quality

The project uses golangci-lint with configuration in `.golangci.yml`:

- Enables 30+ linters
- Enforces Go best practices
- Checks for security issues
- Validates code formatting

## Test Data Management

### Database Tests
- Use temporary SQLite files
- Clean up after each test
- Create fresh test data for each test

### Mock Dependencies
- Bot tests avoid real Telegram API calls
- System tests work with actual OS data
- Auth tests use in-memory configurations

## Test Conventions

### Naming
- Test files: `*_test.go`
- Test functions: `TestFunctionName`
- Benchmark functions: `BenchmarkFunctionName`
- Helper functions: `setupTest*`, `teardownTest*`

### Structure
```go
func TestFunctionName(t *testing.T) {
    // Setup
    // Execute
    // Assert
    // Cleanup
}
```

### Error Handling
- Use `t.Error()` for non-fatal assertions
- Use `t.Fatal()` for fatal errors that stop test execution
- Provide descriptive error messages

## Performance Testing

### Benchmarks
Key performance metrics are tracked through benchmarks:

- Database operations (CRUD)
- System information retrieval
- Data formatting functions

### Race Conditions
All tests are regularly run with `-race` flag to detect:
- Concurrent access issues
- Data races
- Synchronization problems

## Debugging Tests

### Verbose Output
```bash
go test -v ./internal/...
```

### Individual Test
```bash
go test -v -run TestSpecificFunction ./internal/package
```

### Test with Debug Information
```bash
go test -v -tags debug ./internal/...
```

## Test Environment Setup

### Requirements
- Go 1.21+
- Windows OS (for full integration testing)
- SQLite support
- Network access (for system information tests)

### Dependencies
All test dependencies are managed through `go.mod`:
- Standard library testing package
- SQLite driver for database tests
- gopsutil for system tests

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD approach)
2. **Maintain high coverage** (aim for 90%+)
3. **Test error conditions** as well as success paths
4. **Add benchmarks** for performance-critical code
5. **Update documentation** for new test patterns

### Test Review Checklist

- [ ] Tests cover all public functions
- [ ] Error cases are tested
- [ ] Edge cases are covered
- [ ] Tests are deterministic
- [ ] Cleanup is properly handled
- [ ] Test names are descriptive
- [ ] No hardcoded values (use constants/variables)

## Troubleshooting

### Common Issues

1. **Database locks**: Ensure proper cleanup in `teardown` functions
2. **File permissions**: Run tests with appropriate permissions
3. **Network issues**: Some system tests require network access
4. **Race conditions**: Use proper synchronization in concurrent code

### Debug Output
Set environment variable for verbose test output:
```bash
set CUPBOT_TEST_DEBUG=1
go test -v ./internal/...
```