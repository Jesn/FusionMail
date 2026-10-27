package logger

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"
)

func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

func getFileName(file string) string {
	parts := strings.Split(file, "/")
	return parts[len(parts)-1]
}

func hasPrintfVerb(format string) bool {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i >= len(format) {
			return false
		}
		if format[i] == '%' {
			continue
		}

		if format[i] == '[' {
			for i < len(format) && format[i] != ']' {
				i++
			}
			if i >= len(format) {
				return false
			}
			i++
		}

		for i < len(format) && strings.ContainsRune("#0+- ", rune(format[i])) {
			i++
		}
		for i < len(format) && ((format[i] >= '0' && format[i] <= '9') || format[i] == '*') {
			i++
		}
		if i < len(format) && format[i] == '.' {
			i++
			for i < len(format) && ((format[i] >= '0' && format[i] <= '9') || format[i] == '*') {
				i++
			}
		}
		if i < len(format) && format[i] == '[' {
			for i < len(format) && format[i] != ']' {
				i++
			}
			if i >= len(format) {
				return false
			}
			i++
		}
		if i < len(format) && strings.ContainsRune("vTtbcdoOqxXUeEfFgGspw", rune(format[i])) {
			return true
		}
	}

	return false
}

func formatPrintfMessage(format string, args []interface{}) string {
	return fmt.Sprintf(format, args...)
}

func appendStructuredArgs(sb *strings.Builder, args []interface{}) {
	for i := 0; i < len(args); i += 2 {
		key := fmt.Sprintf("arg%d", i+1)
		value := args[i]
		if keyArg, ok := args[i].(string); ok && keyArg != "" && i+1 < len(args) {
			key = keyArg
			value = args[i+1]
		}

		sb.WriteString(fmt.Sprintf(" %s=%v", key, value))
	}
}

func (l *Logger) formatMessage(level LogLevel, msg string, args ...interface{}) string {
	var sb strings.Builder
	sb.WriteString(time.Now().Format("2006/01/02 15:04:05"))
	sb.WriteString(" [")
	sb.WriteString(levelNames[level])
	sb.WriteString("]")
	if l.module != "" {
		sb.WriteString("[")
		sb.WriteString(l.module)
		sb.WriteString("]")
	}
	sb.WriteString(" ")
	hasStructuredArgs := len(args) > 0 && !hasPrintfVerb(msg)
	if len(args) > 0 && !hasStructuredArgs {
		sb.WriteString(formatPrintfMessage(msg, args))
	} else {
		sb.WriteString(msg)
	}
	if len(l.fields) > 0 || hasStructuredArgs {
		sb.WriteString(" |")
		for k, v := range l.fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
		if hasStructuredArgs {
			appendStructuredArgs(&sb, args)
		}
	}
	if _, file, line, ok := runtime.Caller(l.calldepth); ok {
		sb.WriteString(fmt.Sprintf(" (%s:%d)", getFileName(file), line))
	}
	return sb.String()
}

func (l *Logger) formatJSONMessage(level LogLevel, msg string, args ...interface{}) string {
	hasStructuredArgs := len(args) > 0 && !hasPrintfVerb(msg)
	if len(args) > 0 && !hasStructuredArgs {
		msg = formatPrintfMessage(msg, args)
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"level":     levelNames[level],
		"message":   msg,
	}
	if l.module != "" {
		entry["module"] = l.module
	}
	for k, v := range l.fields {
		entry[k] = v
	}
	if hasStructuredArgs {
		for i := 0; i < len(args); i += 2 {
			key := fmt.Sprintf("arg%d", i+1)
			value := args[i]
			if keyArg, ok := args[i].(string); ok && keyArg != "" && i+1 < len(args) {
				key = keyArg
				value = args[i+1]
			}
			entry[key] = value
		}
	}
	if _, file, line, ok := runtime.Caller(l.calldepth); ok {
		entry["file"] = getFileName(file)
		entry["line"] = line
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"level":"ERROR","message":"failed to marshal log entry: %v"}`, err)
	}
	return string(data)
}

func (l *Logger) log(level LogLevel, msg string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	var output string
	if l.jsonFormat {
		output = l.formatJSONMessage(level, msg, args...)
	} else {
		output = l.formatMessage(level, msg, args...)
	}
	l.logger.Output(2, output)
}
