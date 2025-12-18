# CLI Comprehensive Test Plan

**Phase**: P26 - KSA CLI Full-Scenario Validation & Test Coverage  
**Version**: 1.0  
**Date**: 2024-12-18  
**Status**: Active

## 1. Executive Summary

This document outlines the comprehensive testing strategy for the KubeStack-AI CLI, ensuring all commands, plugins, configurations, and output formats are thoroughly validated.

**Test Coverage Target**: > 80%  
**E2E Test Cases**: 20+ scenarios  
**Supported Middleware**: 5 (Redis, MySQL, Kafka, Elasticsearch, PostgreSQL)  
**Output Formats**: 3 (text, JSON, YAML)

## 2. Test Scope

### 2.1 In Scope

- ✅ All CLI commands and subcommands
- ✅ Global and command-specific flags
- ✅ All middleware plugin capabilities
- ✅ Configuration file loading and validation
- ✅ Output format rendering (text, JSON, YAML)
- ✅ Error handling and user feedback
- ✅ Help text and documentation
- ✅ Binary compilation and execution

### 2.2 Out of Scope

- ❌ Web UI testing (covered in Phase 23-24)
- ❌ MCP server integration (covered separately)
- ❌ Performance benchmarking (separate phase)
- ❌ Security penetration testing (separate phase)

## 3. Test Levels

### 3.1 Unit Tests

**Purpose**: Validate individual functions and components

**Coverage**:
- Validator functions
- Output formatters
- Config parsers
- Utility functions

**Location**: Throughout codebase (`*_test.go` files)

### 3.2 Integration Tests

**Purpose**: Validate component interactions

**Coverage**:
- Plugin loading and execution
- Config file loading with various options
- Command flag parsing
- Output format conversion

**Location**: `test/integration/`

### 3.3 End-to-End Tests

**Purpose**: Validate complete workflows

**Coverage**:
- Full command execution flows
- Multi-step scenarios (diagnose → fix)
- Real or mock middleware interactions
- Output to different formats

**Location**: `test/e2e/`

### 3.4 Smoke Tests

**Purpose**: Quick sanity checks

**Coverage**:
- All commands execute without crashing
- Help text is available
- Basic functionality works
- Binary runs on target platforms

**Location**: `scripts/cli_smoke_test.sh`

## 4. Test Matrix

### 4.1 Command Coverage Matrix

| Command | Unit Tests | Integration | E2E | Smoke |
|---------|-----------|-------------|-----|-------|
| `ksa` (root) | ✅ | ✅ | ✅ | ✅ |
| `ksa diagnose` | ✅ | ✅ | ✅ | ✅ |
| `ksa ask` | ✅ | ✅ | ✅ | ✅ |
| `ksa fix` | ✅ | ✅ | ✅ | ✅ |
| `ksa server` | ✅ | ✅ | ✅ | ✅ |
| `ksa monitor` | ✅ | ✅ | ⚠️  | ✅ |
| `ksa alert` | ✅ | ✅ | ⚠️  | ✅ |
| `ksa version` | ✅ | ✅ | ✅ | ✅ |

Legend: ✅ Complete, ⚠️ Partial, ❌ Not Covered

### 4.2 Plugin Coverage Matrix

| Plugin | Health Check | Metrics | Diagnose | Execute | Config |
|--------|--------------|---------|----------|---------|--------|
| Redis | ✅ | ✅ | ✅ | ✅ | ✅ |
| MySQL | ✅ | ✅ | ✅ | ✅ | ✅ |
| Kafka | ✅ | ✅ | ✅ | ✅ | ✅ |
| Elasticsearch | ✅ | ✅ | ✅ | ✅ | ✅ |
| PostgreSQL | ✅ | ✅ | ✅ | ✅ | ✅ |
| MongoDB | 📝 | 📝 | 📝 | 📝 | 📝 |
| RabbitMQ | 📝 | 📝 | 📝 | 📝 | 📝 |

Legend: ✅ Implemented & Tested, 📝 TODO/Planned

### 4.3 Output Format Matrix

| Command | text | json | yaml | table |
|---------|------|------|------|-------|
| diagnose | ✅ | ✅ | ✅ | ⚠️ |
| ask | ✅ | ✅ | ✅ | ❌ |
| fix | ✅ | ✅ | ✅ | ❌ |
| version | ✅ | ⚠️ | ⚠️ | ❌ |
| monitor | ✅ | ✅ | ✅ | ✅ |

## 5. Test Cases

### 5.1 Command Structure Tests

#### TC-001: All Commands Have Help Text
**File**: `test/e2e/cli_commands_comprehensive_test.go::TestAllCommandsHaveHelp`

**Objective**: Verify all commands have complete help documentation

**Steps**:
1. Get root command
2. Iterate through all subcommands
3. Verify Use, Short, and Long fields are populated
4. Check that help text meets quality standards

**Expected**: All commands have complete help text

**Priority**: P0 (Critical)

#### TC-002: Command Tree Registration
**File**: `test/e2e/cli_commands_comprehensive_test.go::TestAllCommandsAreRegistered`

**Objective**: Verify all expected commands are registered

**Steps**:
1. Get registered commands
2. Compare with expected command list
3. Verify no commands are missing

**Expected**: All 7 commands registered (diagnose, ask, fix, server, monitor, alert, version)

**Priority**: P0 (Critical)

#### TC-003: Global Flags Work
**File**: `test/e2e/cli_commands_comprehensive_test.go::TestGlobalFlagsWork`

**Objective**: Verify global flags work with all commands

**Steps**:
1. Check --config flag exists
2. Check --log-level flag exists
3. Check --output flag exists
4. Verify flags have usage text

**Expected**: All global flags present and documented

**Priority**: P0 (Critical)

### 5.2 Plugin Tests

#### TC-101: Redis Plugin Full Coverage
**File**: `test/e2e/cli_plugins_full_coverage_test.go::TestRedisPlugin_AllCapabilities`

**Objective**: Test all Redis plugin capabilities

**Setup**: Redis instance at localhost:6379 (or mock)

**Steps**:
1. Load Redis plugin
2. Test health check
3. Test metrics collection
4. Test diagnosis
5. Test command execution
6. Test configuration

**Expected**: All capabilities work correctly

**Priority**: P0 (Critical)

#### TC-102: MySQL Plugin Full Coverage
**File**: `test/e2e/cli_plugins_full_coverage_test.go::TestMySQLPlugin_AllCapabilities`

**Objective**: Test all MySQL plugin capabilities

**Steps**: Similar to TC-101 for MySQL

**Priority**: P0 (Critical)

#### TC-103: Plugin Loading Performance
**File**: `test/e2e/cli_plugins_full_coverage_test.go::TestPluginLoadingPerformance`

**Objective**: Verify plugins load efficiently

**Steps**:
1. Measure plugin manager initialization time
2. Verify time < 2 seconds

**Expected**: Fast plugin loading

**Priority**: P1 (High)

### 5.3 Configuration Tests

#### TC-201: Default Config Loads
**File**: `test/e2e/cli_config_validation_test.go::TestDefaultConfigLoads`

**Objective**: Verify default configuration loads successfully

**Steps**:
1. Load configs/config.yaml
2. Verify all sections present
3. Validate structure

**Expected**: Config loads without errors

**Priority**: P0 (Critical)

#### TC-202: Custom Config Path
**File**: `test/e2e/cli_config_validation_test.go::TestCustomConfigPath`

**Objective**: Verify custom config paths work

**Steps**:
1. Create temporary config file
2. Load config from custom path
3. Verify config is used

**Expected**: Custom config loads correctly

**Priority**: P1 (High)

#### TC-203: Config Validation
**File**: `test/e2e/cli_config_validation_test.go::TestConfigValidation`

**Objective**: Verify configuration validation logic

**Steps**:
1. Test valid config passes validation
2. Test invalid port fails validation
3. Test missing required fields fail validation

**Expected**: Validation catches errors

**Priority**: P0 (Critical)

### 5.4 Output Format Tests

#### TC-301: JSON Output Format
**File**: `test/e2e/cli_output_formats_test.go::TestOutputFormat_JSON`

**Objective**: Verify JSON output is valid and complete

**Steps**:
1. Create sample diagnosis report
2. Render as JSON
3. Parse JSON
4. Verify all fields present
5. Validate against schema

**Expected**: Valid, complete JSON output

**Priority**: P0 (Critical)

#### TC-302: YAML Output Format
**File**: `test/e2e/cli_output_formats_test.go::TestOutputFormat_YAML`

**Objective**: Verify YAML output is valid and complete

**Steps**: Similar to TC-301 for YAML

**Priority**: P0 (Critical)

#### TC-303: Text Output Format
**File**: `test/e2e/cli_output_formats_test.go::TestOutputFormat_Text`

**Objective**: Verify text output is human-readable

**Steps**:
1. Render as text
2. Verify key information present
3. Check readability (multiple lines, sections)

**Expected**: Human-readable text output

**Priority**: P0 (Critical)

#### TC-304: Output Round-Trip
**File**: `test/e2e/cli_output_formats_test.go::TestOutputFormatRoundTrip`

**Objective**: Verify data integrity through format conversion

**Steps**:
1. Create original data
2. Convert to JSON, parse back
3. Convert to YAML, parse back
4. Verify data matches original

**Expected**: No data loss in conversion

**Priority**: P1 (High)

### 5.5 End-to-End Scenarios

#### TC-401: Diagnose-Fix Workflow
**File**: `test/e2e/scenarios/diagnosis_flow_test.go`

**Objective**: Test complete diagnose-fix workflow

**Steps**:
1. Run diagnose command
2. Capture diagnosis ID
3. Run fix command with ID
4. Verify fix applied

**Expected**: Complete workflow succeeds

**Priority**: P0 (Critical)

#### TC-402: Ask Command Streaming
**File**: Test to be created

**Objective**: Verify ask command provides streaming responses

**Steps**:
1. Execute ask command with question
2. Verify streaming output
3. Check for errors

**Expected**: Streaming response without errors

**Priority**: P1 (High)

#### TC-403: Multi-Format Output
**File**: Test to be created

**Objective**: Verify same data can be output in multiple formats

**Steps**:
1. Run same command with different -o flags
2. Verify all formats produce valid output
3. Compare data consistency

**Expected**: All formats work correctly

**Priority**: P1 (High)

## 6. Test Execution

### 6.1 Manual Test Execution

```bash
# Run all E2E tests
go test -v ./test/e2e/...

# Run specific test file
go test -v ./test/e2e/cli_commands_comprehensive_test.go

# Run smoke tests
./scripts/cli_smoke_test.sh

# Run with coverage
go test -v -cover ./test/e2e/...
```

### 6.2 Automated Test Execution

CI/CD pipeline runs:
1. Unit tests on every commit
2. Integration tests on every PR
3. E2E tests before merge
4. Smoke tests on builds

### 6.3 Test Environment

**Requirements**:
- Go 1.21+
- Access to test middleware instances (or mocks)
- Config files in place
- Built binary

**Setup**:
```bash
# Build binary
make build

# Set up test environment
export KSA_TEST_MODE=true
export KSA_CONFIG=configs/test/cli_test_config.yaml

# Run tests
make test-e2e
```

## 7. Test Data

### 7.1 Test Fixtures

Location: `test/e2e/testdata/`

Files:
- `sample_diagnosis_report.json` - Sample diagnosis output
- `sample_config.yaml` - Test configuration
- `middleware_instances.yaml` - Test instance definitions

### 7.2 Mock Data

Tests use mock data for:
- Middleware responses
- LLM responses
- Configuration values

## 8. Success Criteria

### 8.1 Coverage Metrics

- **Overall Test Coverage**: > 80%
- **CLI Package Coverage**: > 90%
- **Command Coverage**: 100% (all commands tested)
- **Plugin Coverage**: 100% (all 5 implemented plugins tested)
- **Output Format Coverage**: 100% (all 3 formats tested)

### 8.2 Quality Metrics

- **Zero Critical Bugs**: No P0 bugs in production
- **Fast Execution**: All smoke tests complete < 30 seconds
- **Documentation**: 100% of commands documented
- **Help Text**: 100% of flags have usage text

### 8.3 Acceptance Criteria

All of the following must pass:

- ✅ AC-1: Command completeness - All commands have complete help text
- ✅ AC-2: Plugin coverage - 5 plugins 100% tested
- ✅ AC-3: Output formats - text/json/yaml 100% coverage
- ✅ AC-4: Config validation - All config tests pass
- ✅ AC-5: E2E tests - At least 20 test cases, all passing
- ✅ AC-6: Documentation sync - Docs match implementation
- ✅ AC-7: Binary compilation - Builds on Linux/Mac/Windows
- ✅ AC-8: Smoke tests - All tests pass without errors

## 9. Defect Management

### 9.1 Bug Severity Levels

- **P0 (Critical)**: CLI crashes, data loss, security issues
- **P1 (High)**: Major functionality broken, incorrect output
- **P2 (Medium)**: Minor functionality issues, cosmetic problems
- **P3 (Low)**: Documentation errors, enhancement requests

### 9.2 Bug Lifecycle

1. **Detection**: Found during testing
2. **Logging**: Recorded in issue tracker
3. **Triage**: Severity assigned
4. **Fix**: Developer implements fix
5. **Verification**: Tester verifies fix
6. **Closure**: Bug marked as resolved

## 10. Test Schedule

| Phase | Duration | Activities |
|-------|----------|------------|
| Test Planning | Day 1 | Create test plan, define test cases |
| Test Development | Day 1-2 | Write test code, create fixtures |
| Test Execution | Day 2 | Run all tests, record results |
| Bug Fixing | Day 2-3 | Fix found issues, retest |
| Verification | Day 3 | Final test pass, coverage check |
| Documentation | Day 3 | Update docs, create reports |

## 11. Test Reports

### 11.1 Coverage Report

Generated by: `go test -coverprofile=coverage.out`

Includes:
- Line coverage by package
- Function coverage
- Branch coverage
- Uncovered lines

### 11.2 Test Results Report

Generated after each test run

Includes:
- Total tests run
- Pass/fail count
- Execution time
- Failed test details

### 11.3 Smoke Test Report

Generated by: `scripts/cli_smoke_test.sh`

Format:
```
==========================================
     SMOKE TEST SUMMARY
==========================================
Total Tests Run:    15
Tests Passed:       15
Tests Failed:       0
==========================================
✓ All smoke tests passed!
```

## 12. Continuous Testing

### 12.1 CI/CD Integration

Tests run automatically on:
- Every commit (unit tests)
- Every PR (integration tests)
- Every merge (full test suite)
- Nightly builds (extended tests)

### 12.2 Test Automation

All tests are automated and can be run with:
```bash
make test          # Unit tests
make test-integration  # Integration tests
make test-e2e      # E2E tests
make test-all      # All tests
```

## 13. Test Maintenance

### 13.1 Test Updates

Tests should be updated when:
- New commands are added
- Command behavior changes
- New flags are introduced
- Output format changes
- Bugs are fixed (add regression tests)

### 13.2 Test Review

Regular reviews:
- Monthly test coverage review
- Quarterly test plan update
- Annual test strategy review

## 14. Appendix

### A. Test Command Reference

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -run TestSpecificTest ./...

# Run smoke tests
./scripts/cli_smoke_test.sh

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### B. Test Environment Variables

```bash
# Enable test mode
export KSA_TEST_MODE=true

# Set test config
export KSA_CONFIG=configs/test/cli_test_config.yaml

# Mock middleware endpoints
export TEST_REDIS_ENDPOINT=localhost:6379
export TEST_MYSQL_ENDPOINT=localhost:3306
```

### C. Common Test Issues

**Issue**: Tests fail due to missing dependencies  
**Solution**: Run `go mod download` and ensure all dependencies are installed

**Issue**: Plugin tests fail  
**Solution**: Verify plugin directory exists and plugins are built

**Issue**: Config tests fail  
**Solution**: Ensure config files exist in expected locations

**Issue**: Smoke tests fail  
**Solution**: Build binary first with `make build`

---

**Test Plan Version**: 1.0  
**Next Review Date**: 2025-01-18  
**Owner**: KubeStack-AI Development Team
