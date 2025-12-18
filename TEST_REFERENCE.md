# Phase 26 Test Reference

## Quick Test Commands

### Run All CLI Tests
```bash
cd /workspace/project/kubestack-ai
go test -v ./test/e2e/cli_*.go
```

### Run Individual Test Files
```bash
# Command structure tests
go test -v ./test/e2e/cli_commands_comprehensive_test.go

# Plugin coverage tests
go test -v ./test/e2e/cli_plugins_full_coverage_test.go

# Configuration validation tests
go test -v ./test/e2e/cli_config_validation_test.go

# Output format tests
go test -v ./test/e2e/cli_output_formats_test.go
```

### Run Smoke Tests
```bash
./scripts/cli_smoke_test.sh
```

### Run with Coverage
```bash
go test -v -cover ./test/e2e/cli_*.go
go test -coverprofile=coverage.out ./test/e2e/...
go tool cover -html=coverage.out -o coverage.html
```

## Test Files Overview

### 1. cli_commands_comprehensive_test.go
**Purpose**: Validate command structure and completeness
**Tests**:
- TestAllCommandsHaveHelp
- TestAllCommandsAreRegistered
- TestGlobalFlagsWork
- TestCommandTreeValidation
- TestDiagnoseCommandFlags
- TestAskCommandFlags
- TestFixCommandFlags
- TestServerCommandFlags
- TestVersionCommand
- TestHelpTextQuality
- TestCommandExamples

### 2. cli_plugins_full_coverage_test.go
**Purpose**: Test all middleware plugin capabilities
**Tests**:
- TestPluginCoverageMatrix
- TestRedisPlugin_AllCapabilities
- TestMySQLPlugin_AllCapabilities
- TestKafkaPlugin_AllCapabilities
- TestElasticsearchPlugin_AllCapabilities
- TestPostgreSQLPlugin_AllCapabilities
- TestPluginLoadingPerformance
- TestPluginCompatibility

### 3. cli_config_validation_test.go
**Purpose**: Validate configuration loading and validation
**Tests**:
- TestDefaultConfigLoads
- TestCustomConfigPath
- TestConfigValidation
- TestEnvironmentVariableOverride
- TestMiddlewareConfigs
- TestConfigSecrets
- TestConfigSchema
- TestConfigDefaults

### 4. cli_output_formats_test.go
**Purpose**: Test output format rendering and validation
**Tests**:
- TestOutputFormat_JSON
- TestOutputFormat_YAML
- TestOutputFormat_Text
- TestJSONSchemaCompliance
- TestOutputFormatRoundTrip
- TestOutputFormatErrors
- TestOutputFormatterFactory
- TestOutputFormatPerformance

### 5. cli_smoke_test.sh
**Purpose**: Automated smoke testing for CI/CD
**Checks** (15 total):
1. Version command
2. Help text availability
3. Diagnose command help
4. Ask command help
5. Fix command help
6. Server command help
7. Config file validation
8. JSON output format
9. YAML output format
10. Invalid command handling
11. Missing required flag
12. Log level functionality
13. Global flags persistence
14. Diagnose dry-run
15. Binary size check

## Test Data

### Config Files
- `configs/test/cli_test_config.yaml` - Test configuration
- `configs/config.yaml` - Default configuration

### Middleware Templates
- `configs/middleware/mongodb.yaml` (TODO)
- `configs/middleware/rabbitmq.yaml` (TODO)

## Validator Functions

Located in `internal/cli/validator/cli_validator.go`:

1. **ValidateCommand()** - Individual command validation
2. **ValidateCommandTree()** - Recursive tree validation
3. **ValidateMiddlewareType()** - Middleware type checking
4. **ValidateOutputFormat()** - Format validation
5. **ValidateConnectionString()** - Connection string format
6. **ValidateFlagsCompatibility()** - Flag combination checking
7. **ValidateInstanceName()** - Instance naming validation
8. **ValidateConfig()** - Configuration validation

## Expected Test Results

### Passing Tests
All tests should pass or skip gracefully with appropriate messages.

### Skipped Tests
Some tests may skip if:
- Middleware instances not available
- Test dependencies not installed
- Environment not configured

### Coverage Target
- Overall: 80%+
- CLI Package: 90%+
- Validator: 95%+

## Continuous Integration

### GitHub Actions (Recommended)
```yaml
- name: Run CLI Tests
  run: |
    go test -v ./test/e2e/cli_*.go
    ./scripts/cli_smoke_test.sh
```

### Pre-commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit
echo "Running CLI tests..."
go test ./test/e2e/cli_*.go || exit 1
./scripts/cli_smoke_test.sh || exit 1
```

## Troubleshooting

### Test Failures
1. Check Go version (1.21+ required)
2. Verify dependencies: `go mod tidy`
3. Check config files exist
4. Review test logs for details

### Smoke Test Issues
1. Verify binary exists: `ls -lh ksa`
2. Check execute permissions: `chmod +x scripts/cli_smoke_test.sh`
3. Ensure config files in place
4. Review colored output for specific failures

## Documentation

- [Design Document](docs/round6/phase26/design-cli-validation.md)
- [Test Plan](docs/round6/phase26/test-plan-cli-comprehensive.md)
- [CLI Reference](docs/round6/phase26/cli-command-reference.md)
- [Completion Report](docs/round6/phase26/PHASE26_COMPLETION_REPORT.md)

## Quick Reference

### Key Files
```
internal/cli/validator/cli_validator.go          - Validation logic
test/e2e/cli_commands_comprehensive_test.go      - Command tests
test/e2e/cli_plugins_full_coverage_test.go       - Plugin tests
test/e2e/cli_config_validation_test.go           - Config tests
test/e2e/cli_output_formats_test.go              - Format tests
scripts/cli_smoke_test.sh                        - Smoke tests
```

### Commands
```bash
# Build binary
make build

# Run tests
make test

# Run E2E tests
make e2e-test

# Smoke tests
./scripts/cli_smoke_test.sh
```

---

**Last Updated**: 2024-12-18  
**Phase**: P26 - CLI Full-Scenario Validation & Test Coverage  
**Status**: Complete ✅
