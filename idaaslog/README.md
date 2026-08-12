# idaaslog - Alibaba Cloud IDaaS Logging Package

`idaaslog` is the logging package for Alibaba Cloud IDaaS CLI. It provides structured logging with multiple log levels, automatic log rotation, and optional console output.

## Overview

The `idaaslog` package is designed for secure, production-ready logging in the Alibaba Cloud IDaaS CLI. It supports:

- **Multiple log levels**: DEBUG, INFO, WARN, ERROR, UNSAFE
- **Automatic log rotation**: Keeps up to 100 log files, auto-cleanup
- **Console output toggle**: Optional stderr output for debugging
- **Unsafe data protection**: Sensitive data logging requires explicit enablement
- **Stack trace support**: Error dumping with stack traces (using `github.com/pkg/errors`)

## Quick Start

### Initialization

```go
import "github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"

func main() {
    // Initialize logging system
    idaaslog.InitLog()
    defer idaaslog.CloseLog()
    
    // Use loggers
    idaaslog.Info.PrintfLn("Application started")
    idaaslog.Debug.PrintfLn("Debug info: %s", debugData)
}
```

## Log Levels

| Level  | Constant         | Description                | Console Output                                                                                  |
| ------ | ---------------- | -------------------------- | ----------------------------------------------------------------------------------------------- |
| DEBUG  | `LogLevelDebug`  | Detailed debug information | Requires `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT`                                             |
| INFO   | `LogLevelInfo`   | General information        | Requires `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT`                                             |
| WARN   | `LogLevelWarn`   | Warning messages           | Requires `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT`                                             |
| ERROR  | `LogLevelError`  | Error messages             | Requires `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT`                                             |
| UNSAFE | `LogLevelUnsafe` | Sensitive/unsafe data      | Requires BOTH `ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG` AND `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT` |

### Usage Examples

```go
// Info level
idaaslog.Info.PrintfLn("Profile loaded: %s", profileName)

// Debug level
idaaslog.Debug.PrintfLn("Token request: endpoint=%s", endpoint)

// Warning level
idaaslog.Warn.PrintfLn("Deprecated config field: %s", fieldName)

// Error level
idaaslog.Error.PrintfLn("Failed to fetch token: %v", err)

// Unsafe level (sensitive data)
idaaslog.Unsafe.PrintfLn("Access token: %s", accessToken)
```

## Environment Variables

### `ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG`

- **Purpose**: Enable logging of sensitive/unsafe data
- **Values**: `1`, `true`, `yes`, `y`, `on` (case-insensitive)
- **Default**: Disabled
- **Effect**: Allows `LogLevelUnsafe` messages to be written to log files

**Security Note**: Never enable this in production. Only use for local debugging.

```bash
# Enable unsafe logging
export ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG=true
./alibaba-cloud-idaas fetch-token --profile test
```

### `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT`

- **Purpose**: Copy all log output to stderr (console)
- **Values**: `1`, `true`, `yes`, `y`, `on` (case-insensitive)
- **Default**: Disabled
- **Effect**: All log messages (except UNSAFE without UNSAFE_DEBUG) are printed to stderr

**Use Case**: Real-time debugging without checking log files.

```bash
# Enable console output
export ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT=true
./alibaba-cloud-idaas fetch-token --profile test
```

### Combined Usage

```bash
# Enable both for full debug output including sensitive data
export ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG=true
export ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT=true
./alibaba-cloud-idaas fetch-token --profile test
```

## Log File Management

### Location

Log files are stored in:

```
$HOME/.aliyun/alibaba-cloud-idaas/logs/
```

### File Naming

Format: `YYYY-MM-DD-HHMMSS_xxxxxxxxxxxxxxxx.log`

Example: `2025-04-07-143022_a1b2c3d4e5f67890.log`

### Automatic Cleanup

- **Trigger**: When log file count exceeds 110
- **Action**: Deletes oldest files
- **Retention**: Keeps maximum 100 log files
- **Sorting**: By modification time (oldest first)

## API Reference

### Logger Instances

```go
var Debug IdaasLog   // LogLevelDebug
var Info IdaasLog    // LogLevelInfo
var Warn IdaasLog    // LogLevelWarn
var Error IdaasLog   // LogLevelError
var Unsafe IdaasLog  // LogLevelUnsafe
```

### Methods

#### `PrintfLn(format string, a ...interface{})`

Print formatted log message with automatic newline.

```go
idaaslog.Info.PrintfLn("User %s logged in at %s", username, time.Now())
```

#### `InitLog()`

Initialize the logging system. Opens a new log file.

**Must be called before any logging operations.**

```go
idaaslog.InitLog()
defer idaaslog.CloseLog()
```

#### `CloseLog()`

Close the current log file. Should be called on application exit.

#### `IsCurrentLog(filename string) bool`

Check if the given filename is the currently active log file.

```go
if idaaslog.IsCurrentLog("/path/to/logfile.log") {
    // This is the active log file
}
```

#### `DumpError(err error) string`

Convert error to string with stack trace (if available).

Supports `github.com/pkg/errors` stack traces.

```go
err := errors.Wrap(someErr, "failed to process")
logMsg := idaaslog.DumpError(err)
idaaslog.Error.PrintfLn("%s", logMsg)
```

Output example:
```
Error: failed to process: original error message
/path/to/file.go:42
/path/to/another/file.go:15
```

### Utility Functions

#### `IsOn(val string) bool`

Check if a string value represents "on" or "true".

Used internally for environment variable parsing.

```go
if idaaslog.IsOn(os.Getenv("SOME_FLAG")) {
    // Flag is enabled
}
```

## Security Best Practices

1. **Never commit logs with sensitive data**
   - Logs may contain tokens, credentials, or PII
   - Always review before sharing

2. **Disable unsafe logging in production**
   ```bash
   # Ensure these are NOT set in production
   unset ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG
   unset ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT
   ```

3. **Use UNSAFE level sparingly**
   - Only for temporary debugging
   - Never in production code paths

4. **Log file permissions**
   - Log files are created with `0644` permissions
   - Ensure `$HOME/.aliyun/` directory has proper permissions

## Example: Complete Usage

```go
package main

import (
    "os"
    "time"
    
    "github.com/aliyunidaas/alibaba-cloud-idaas/idaaslog"
)

func main() {
    // Initialize logging
    idaaslog.InitLog()
    defer idaaslog.CloseLog()
    
    idaaslog.Info.PrintfLn("Application started: %s", time.Now())
    
    profile := "test-profile"
    idaaslog.Debug.PrintfLn("Loading profile: %s", profile)
    
    // Simulate operation
    err := processProfile(profile)
    if err != nil {
        idaaslog.Error.PrintfLn("Failed to process profile: %s", 
            idaaslog.DumpError(err))
        os.Exit(1)
    }
    
    idaaslog.Info.PrintfLn("Profile processed successfully")
}

func processProfile(profile string) error {
    // Your logic here
    return nil
}
```

## Troubleshooting

### No log files created

- Check if `InitLog()` was called
- Verify write permissions on `$HOME/.aliyun/`
- Check for errors in stderr output

### Console output not showing

- Ensure `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT` is set
- Check environment variable spelling
- Restart application after setting variable

### Log files not rotating

- Rotation only triggers when count > 110
- Check actual file count in log directory
- Manual cleanup may be needed if disk is full

### Unsafe logs not appearing

- Both `ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG` AND `ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT` must be set
- UNSAFE level requires explicit enablement for security

## Agent Instructions

### For Debugging Issues

1. **Enable full logging**:
   ```bash
   export ALIBABA_CLOUD_IDAAS_UNSAFE_DEBUG=true
   export ALIBABA_CLOUD_IDAAS_UNSAFE_CONSOLE_PRINT=true
   ```

2. **Reproduce the issue**:
   ```bash
   ./alibaba-cloud-idaas <command> --profile <profile>
   ```

3. **Locate log file**:
   - Check stderr output for "Log file: /path/to/file.log"
   - Or list recent logs: `ls -lt ~/.aliyun/alibaba-cloud-idaas/logs/`

4. **Share relevant logs**:
   - **WARNING**: Review logs before sharing - may contain sensitive data
   - Redact tokens, credentials, PII before sharing
   - Use `grep` to extract relevant sections:
     ```bash
     grep -A5 -B5 "ERROR" ~/.aliyun/alibaba-cloud-idaas/logs/latest.log
     ```

### For Adding New Log Statements

1. **Choose appropriate level**:
   - DEBUG: Development/tracing only
   - INFO: Important milestones/events
   - WARN: Recoverable issues, deprecations
   - ERROR: Failures requiring attention
   - UNSAFE: Sensitive data (tokens, credentials) - use sparingly

2. **Format messages consistently**:
   ```go
   // Good
   idaaslog.Info.PrintfLn("Fetched token for profile: %s", profileName)
   
   // Bad - too vague
   idaaslog.Info.PrintfLn("Done")
   ```

3. **Include context**:
   ```go
   // Include profile name, endpoint, etc.
   idaaslog.Debug.PrintfLn("Token request to %s with scope %s", 
       endpoint, scope)
   ```

4. **Use DumpError for errors**:
   ```go
   // Good
   idaaslog.Error.PrintfLn("Failed: %s", idaaslog.DumpError(err))
   
   // Bad - loses stack trace
   idaaslog.Error.PrintfLn("Failed: %v", err)
   ```

### For Log Analysis

Common patterns:

```bash
# Find all errors
grep "ERROR " ~/.aliyun/alibaba-cloud-idaas/logs/*.log

# Find errors with context (5 lines before/after)
grep -B5 -A5 "ERROR " ~/.aliyun/alibaba-cloud-idaas/logs/latest.log

# Count log levels
grep -c "DEBUG " logfile.log
grep -c "INFO  " logfile.log
grep -c "WARN  " logfile.log
grep -c "ERROR " logfile.log

# Check recent activity (last 100 lines)
tail -100 ~/.aliyun/alibaba-cloud-idaas/logs/latest.log
```

## File Structure

```
idaaslog/
├── README.md          # This file
├── log.go             # Main logging implementation
└── util.go            # Utility functions (IsOn)
```

## Dependencies

- `github.com/pkg/errors` - For stack trace support
- Standard library: `os`, `io`, `fmt`, `time`, `crypto/rand`, etc.

## License

Part of Alibaba Cloud IDaaS project.
