package idaaslog

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aliyunidaas/alibaba-cloud-idaas/constants"
	"github.com/pkg/errors"
)

var (
	UnsafeDebug        = IsOn(os.Getenv(constants.EnvUnsafeDebug))
	UnsafeConsolePrint = IsOn(os.Getenv(constants.EnvUnsafeConsolePrint))
)

type Level int

const (
	LogLevelDebug Level = 1 + iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelUnsafe
)

func (l Level) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG "
	case LogLevelInfo:
		return "INFO  "
	case LogLevelWarn:
		return "WARN  "
	case LogLevelError:
		return "ERROR "
	case LogLevelUnsafe:
		return "UNSAFE"
	default:
		return "UNKNOWN" // SHOULD NOT HAPPEN
	}
}

type IdaasLog struct {
	Level Level
}

var Debug = IdaasLog{
	Level: LogLevelDebug,
}
var Info = IdaasLog{
	Level: LogLevelInfo,
}
var Warn = IdaasLog{
	Level: LogLevelWarn,
}
var Error = IdaasLog{
	Level: LogLevelError,
}
var Unsafe = IdaasLog{
	Level: LogLevelUnsafe,
}

var logFileMutex sync.Mutex
var logFile *os.File

func (l *IdaasLog) PrintfLn(format string, a ...interface{}) {
	if (l.Level == LogLevelUnsafe) && !UnsafeDebug {
		// print unsafe log only turn unsafe debug
		return
	}
	internalPrintf("["+l.Level.String()+"] "+format+"\n", a...)
}

func InitLog() {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	if logFile == nil {
		logFile = openNewLogFile()
	}
}

func CloseLog() {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	if logFile != nil {
		_ = logFile.Close()
	}
}

func IsCurrentLog(filename string) bool {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	if logFile == nil {
		return false
	}
	return logFile.Name() == filename
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func DumpError(err error) string {
	errorMessage := err.Error()
	if stackTracerErr, ok := err.(stackTracer); ok {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Error: %s\n", errorMessage))
		for _, f := range stackTracerErr.StackTrace() {
			sb.WriteString(fmt.Sprintf("%+s:%d\n", f, f))
		}
		return sb.String()
	} else {
		return fmt.Sprintf("Error: %+v", err)
	}
}

func internalPrintf(format string, a ...interface{}) {
	logLn := fmt.Sprintf(format, a...)
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	if logFile != nil {
		_, _ = logFile.WriteString(logLn)
	}
	if UnsafeConsolePrint {
		_, _ = fmt.Fprintf(os.Stderr, "%s", logLn)
	}
}

func openNewLogFile() *os.File {
	currentTime := time.Now().Format("2006-01-02-150405")
	random := make([]byte, 8)
	_, _ = io.ReadFull(rand.Reader, random)
	logFilename, err := getLogFile(fmt.Sprintf("%s_%x.log", currentTime, random))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Get log file error: %v\n", err)
		return nil
	}
	if UnsafeConsolePrint {
		_, _ = fmt.Fprintf(os.Stderr, "Log file: %s\n", logFilename)
	}
	logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Open log file: %s for write error: %v\n", logFilename, err)
		return nil
	}
	return logFile
}

type DirEntryWithInfo struct {
	DirEntry os.DirEntry
	FileInfo os.FileInfo
	Error    error
}

func getLogFile(key string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	logCacheDir := filepath.Join(homeDir, constants.DotAliyunDir, constants.AlibabaCloudIdaasDir, constants.LogDir)
	if _, err := os.Stat(logCacheDir); os.IsNotExist(err) {
		mkdirErr := os.MkdirAll(logCacheDir, 0755)
		if mkdirErr != nil {
			return "", mkdirErr
		}
	}
	logFiles, err := os.ReadDir(logCacheDir)
	if err == nil && UnsafeConsolePrint {
		_, _ = fmt.Fprintf(os.Stderr, "Found: %d log file(s)\n", len(logFiles))
	}
	if err == nil && len(logFiles) > 110 {
		clearLogFiles(logFiles, logCacheDir)
	}
	return filepath.Join(logCacheDir, key), nil
}

func clearLogFiles(logFiles []os.DirEntry, logCacheDir string) {
	var dirEntriesWithInfo []DirEntryWithInfo
	for _, file := range logFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			fileInfo, fileInfoErr := file.Info()
			dirEntriesWithInfo = append(dirEntriesWithInfo, DirEntryWithInfo{file, fileInfo, fileInfoErr})
		}
	}
	sort.Slice(dirEntriesWithInfo, func(i, j int) bool {
		info1 := dirEntriesWithInfo[i]
		info2 := dirEntriesWithInfo[j]
		if info1.Error != nil || info2.Error != nil {
			if info1.Error != nil {
				return true
			}
			if info2.Error != nil {
				return false
			}
			return true
		}
		return info1.FileInfo.ModTime().Before(info2.FileInfo.ModTime())
	})
	totalFileCount := len(dirEntriesWithInfo)
	for _, file := range dirEntriesWithInfo {
		if totalFileCount <= 100 {
			// finally keep 100 log files
			return
		}
		if UnsafeConsolePrint {
			_, _ = fmt.Fprintf(os.Stderr, "Remove log file: %s %s\n", logCacheDir, file.DirEntry.Name())
		}
		_ = os.Remove(filepath.Join(logCacheDir, file.DirEntry.Name()))
		totalFileCount--
	}
}
