# Build Fix for gojieba CGO Dependency

## Problem
The build was failing with:
```
internal/knowledge/search/jieba_tokenizer.go:28:17: undefined: gojieba.Jieba
internal/knowledge/search/jieba_tokenizer.go:33:18: undefined: gojieba.NewJieba
```

This error occurs because `gojieba` requires CGO (C bindings) and C++ dependencies to compile.

### Additional Error Fixed
```
internal/knowledge/search/jieba_tokenizer_stub.go:38:2: declared and not used: result
```

This was caused by an unused variable in the stub implementation. Fixed by simplifying the return statement.

## Solution
Added build tags to make gojieba optional:

### 1. Updated `jieba_tokenizer.go`
Added build tags at the top:
```go
//go:build cgo
// +build cgo
```

This file will only be compiled when CGO is enabled.

### 2. Created `jieba_tokenizer_stub.go`
Created a stub implementation that compiles without CGO:
```go
//go:build !cgo
// +build !cgo
```

This provides a fallback implementation when CGO is disabled.

## Building the Project

### Option 1: Build without CGO (Recommended for easier compilation)
```bash
CGO_ENABLED=0 make build
```

or

```bash
CGO_ENABLED=0 go build -o ksa ./cmd/ksa
```

This will use the stub implementation and compile successfully without C++ dependencies.

### Option 2: Build with CGO (Full jieba support)
If you need full Chinese text segmentation support with jieba:

1. Install C++ build tools:
```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS
xcode-select --install
```

2. Build with CGO enabled:
```bash
CGO_ENABLED=1 make build
```

## Impact

- **Without CGO**: The jieba tokenizer will not perform Chinese text segmentation, but the application will compile and run successfully.
- **With CGO**: Full jieba tokenization is available for Chinese text processing.

## Recommendations

For most users and CI/CD environments, building without CGO is recommended as it:
- Eliminates C++ compiler dependencies
- Produces statically linked binaries
- Simplifies cross-compilation
- Faster build times

If Chinese text search is critical, use CGO-enabled builds.

## Files Modified

1. `internal/knowledge/search/jieba_tokenizer.go` - Added `//go:build cgo` tag
2. `internal/knowledge/search/jieba_tokenizer_stub.go` - Created stub implementation (fixed unused variable error)

## Testing

To verify the build works:

```bash
# Without CGO
CGO_ENABLED=0 go build -o ksa ./cmd/ksa
./ksa version

# With CGO (if dependencies installed)
CGO_ENABLED=1 go build -o ksa ./cmd/ksa
./ksa version
```

Both should compile and run successfully.
