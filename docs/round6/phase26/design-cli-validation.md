# CLI Validation Design Document

**Phase**: P26 - KSA CLI Full-Scenario Validation & Test Coverage  
**Author**: KubeStack-AI Development Team  
**Date**: 2024-12-18  
**Status**: Implementation Complete

## 1. Overview

This document outlines the design and implementation of comprehensive CLI validation for the KubeStack-AI (`ksa`) command-line interface. The validation system ensures all commands, flags, and outputs meet quality standards and provide a consistent user experience.

## 2. Objectives

1. **Completeness**: Verify all commands have complete help text and documentation
2. **Correctness**: Ensure command syntax and flag combinations are valid
3. **Consistency**: Maintain consistent behavior across all commands
4. **User Experience**: Provide clear error messages and helpful guidance
5. **Quality Assurance**: Establish automated validation for ongoing development

## 3. Validation Architecture

### 3.1 Validation Layers

```
┌─────────────────────────────────────────┐
│         CLI Validation System            │
├─────────────────────────────────────────┤
│  1. Structural Validation                │
│     - Command tree completeness          │
│     - Required fields presence           │
│     - Flag definitions                   │
├─────────────────────────────────────────┤
│  2. Semantic Validation                  │
│     - Parameter type checking            │
│     - Value range validation             │
│     - Flag compatibility                 │
├─────────────────────────────────────────┤
│  3. Runtime Validation                   │
│     - Connection string format           │
│     - File path existence                │
│     - Configuration correctness          │
├─────────────────────────────────────────┤
│  4. Output Validation                    │
│     - Format compliance (JSON/YAML/text) │
│     - Schema validation                  │
│     - Data completeness                  │
└─────────────────────────────────────────┘
```

### 3.2 Validator Components

The CLI validation system consists of the following components:

#### internal/cli/validator/cli_validator.go

Core validation functions:
- `ValidateCommand()` - Validates individual command structure
- `ValidateCommandTree()` - Recursively validates entire command tree
- `ValidateMiddlewareType()` - Validates middleware type strings
- `ValidateOutputFormat()` - Validates output format specifications
- `ValidateConnectionString()` - Validates connection string formats
- `ValidateFlagsCompatibility()` - Validates flag combinations
- `ValidateInstanceName()` - Validates instance naming conventions

## 4. Command Validation Rules

### 4.1 Required Fields

Every command MUST have:
- `Use` field - Command syntax
- `Short` field - Brief description (< 120 characters)
- `Long` field (optional but recommended) - Detailed description
- `Example` field (if command has complex usage)

### 4.2 Flag Requirements

Every flag MUST have:
- Descriptive name (lowercase, hyphen-separated)
- Usage text explaining the flag's purpose
- Appropriate default value (if applicable)
- Type validation

### 4.3 Help Text Quality Standards

Help text should:
- Be concise and clear
- Use consistent terminology
- Include examples for complex commands
- Follow standard conventions (e.g., "[required]", "(optional)")

## 5. Middleware Type Validation

### 5.1 Supported Middleware Types

The system validates against these supported middleware types:
- Redis
- MySQL
- Kafka
- Elasticsearch
- PostgreSQL
- MongoDB (TODO)
- RabbitMQ (TODO)
- MinIO (TODO)
- Prometheus (TODO)
- ClickHouse (TODO)

### 5.2 Validation Process

```go
func ValidateMiddlewareType(mwType string) error {
    _, err := enum.ParseMiddlewareType(mwType)
    if err != nil {
        return fmt.Errorf("invalid middleware type '%s', must be one of: %s",
            mwType, strings.Join(enum.AllowedMiddlewareTypes(), ", "))
    }
    return nil
}
```

## 6. Output Format Validation

### 6.1 Supported Formats

- **text**: Human-readable text output
- **json**: JSON format (machine-parseable)
- **yaml**: YAML format (human and machine readable)
- **table**: Tabular format (for structured data)

### 6.2 Format Validation

Each format must:
- Produce valid syntax
- Include all required fields
- Be parseable back to original data structure (for JSON/YAML)
- Maintain data integrity through round-trip conversion

## 7. Connection String Validation

### 7.1 Supported Formats

1. **URL Format**: `protocol://host:port/path`
   - Example: `redis://localhost:6379/0`
   - Example: `mysql://user:pass@localhost:3306/dbname`

2. **Host:Port Format**: `hostname:port`
   - Example: `localhost:6379`
   - Example: `redis-master.default.svc:6379`

3. **Hostname Only**: `hostname`
   - Example: `localhost`
   - Example: `redis-service`

### 7.2 Validation Rules

- Non-empty string
- Valid hostname/IP format
- Port number in valid range (1-65535)
- URL components properly formatted

## 8. Flag Compatibility Validation

### 8.1 Incompatible Flag Combinations

Certain flags are mutually exclusive:

| Flag 1 | Flag 2 | Reason |
|--------|--------|--------|
| `--dry-run` | `--auto-approve` | Dry run doesn't execute, so approval is meaningless |
| `--output json` | `--interactive` | Interactive mode requires text output |

### 8.2 Required Flag Dependencies

Some flags require others:

| Flag | Requires | Reason |
|------|----------|--------|
| `--auto-approve` | `--id` | Need diagnosis ID to approve fixes |
| `--namespace` | `--instance` (in K8s) | Namespace is specific to K8s instances |

## 9. Testing Strategy

### 9.1 Test Levels

1. **Unit Tests**: Validate individual validation functions
2. **Integration Tests**: Test command execution with various inputs
3. **E2E Tests**: Full command-line workflow testing
4. **Smoke Tests**: Quick sanity checks for all commands

### 9.2 Test Coverage

Test files:
- `test/e2e/cli_commands_comprehensive_test.go` - Command structure tests
- `test/e2e/cli_plugins_full_coverage_test.go` - Plugin capability tests
- `test/e2e/cli_config_validation_test.go` - Configuration validation tests
- `test/e2e/cli_output_formats_test.go` - Output format tests

### 9.3 Automated Validation

Smoke test script: `scripts/cli_smoke_test.sh`

Tests performed:
1. Version command output
2. Help text availability for all commands
3. Global flags functionality
4. Output format handling
5. Error handling for invalid inputs
6. Config file validation
7. Binary size and performance checks

## 10. Error Handling

### 10.1 Error Message Guidelines

Error messages should:
- Clearly state what went wrong
- Provide actionable guidance
- Include examples when helpful
- Use consistent formatting

Good example:
```
Error: invalid middleware type 'redi'
Did you mean 'redis'?
Supported types: Redis, MySQL, Kafka, Elasticsearch, PostgreSQL
```

Bad example:
```
Error: invalid input
```

### 10.2 Exit Codes

Standard exit codes:
- `0` - Success
- `1` - General error
- `2` - Usage error (invalid flags/arguments)
- `3` - Configuration error
- `4` - Connection error
- `5` - Execution error

## 11. Validation Workflow

### 11.1 Command Execution Flow

```
User Input
    ↓
Parse Command & Flags
    ↓
Validate Command Structure ──→ [Error: Show usage]
    ↓
Validate Flag Values ──────→ [Error: Invalid value]
    ↓
Validate Flag Compatibility ──→ [Error: Incompatible flags]
    ↓
Execute Command
    ↓
Validate Output Format ────→ [Error: Format error]
    ↓
Render Output
    ↓
Success
```

### 11.2 Pre-execution Validation

Before executing any command, validate:
1. Command syntax is correct
2. All required flags are present
3. Flag values are within acceptable ranges
4. Flag combinations are compatible
5. Configuration is loadable (if needed)

## 12. Continuous Validation

### 12.1 CI/CD Integration

Validation runs automatically in CI/CD:
1. Run unit tests
2. Execute smoke tests
3. Verify all commands have documentation
4. Check output formats for all commands
5. Validate example commands

### 12.2 Pre-commit Hooks

Recommended pre-commit checks:
- Run `go test ./test/e2e/cli_*` 
- Execute `scripts/cli_smoke_test.sh`
- Verify no TODO comments in production code

## 13. Performance Considerations

### 13.1 Validation Performance

Validation should be fast:
- Command tree validation: < 10ms
- Parameter validation: < 1ms
- Output format validation: < 5ms per format

### 13.2 Caching

Cache validation results where appropriate:
- Middleware type enum parsing
- Configuration validation
- Plugin capability checks

## 14. Future Enhancements

### 14.1 Planned Improvements

1. **Auto-correction**: Suggest corrections for common typos
2. **Interactive Mode**: Prompt for missing required fields
3. **Shell Completion**: Generate bash/zsh completion scripts
4. **Config Wizard**: Interactive configuration file generator
5. **Validation Plugins**: Allow plugins to define custom validation rules

### 14.2 Advanced Validation

1. **Semantic Validation**: Understand command intent
2. **Context-Aware Validation**: Validate based on environment
3. **Historical Analysis**: Learn from past command patterns
4. **AI-Powered Suggestions**: Use LLM to suggest command improvements

## 15. References

- Cobra Documentation: https://github.com/spf13/cobra
- CLI Best Practices: https://clig.dev/
- POSIX Utility Conventions: https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html

## 16. Appendix: Validation Checklist

- [ ] All commands have `Use`, `Short`, and `Long` fields
- [ ] All flags have usage text
- [ ] Examples are provided for complex commands
- [ ] Invalid inputs produce helpful error messages
- [ ] All middleware types are validated
- [ ] Output formats are tested (JSON, YAML, text)
- [ ] Connection strings are validated
- [ ] Flag combinations are checked
- [ ] Help text is clear and consistent
- [ ] Tests cover all validation scenarios
- [ ] Smoke tests pass for all commands
- [ ] Documentation matches implementation
