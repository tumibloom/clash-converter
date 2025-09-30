package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime"

	"github.com/fatih/color"
)

type LogHandler struct {
	logger *log.Logger
}

func (handler *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (handler *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return nil
}

func (handler *LogHandler) WithGroup(name string) slog.Handler {
	return nil
}

var Logger *slog.Logger

func caller(pc uintptr) (file string, line int) {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()

	file = f.File
	if file == "" {
		file = "???"
	}
	line = f.Line

	return
}

func shortFile(file string) string {
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			return file[i+1:]
		}
	}

	return file
}

func (handler *LogHandler) Handle(_ context.Context, record slog.Record) error {
	level := fmt.Sprintf("[%5s]", record.Level.String())

	switch record.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	timeStr := record.Time.Format("15:04:05.000")
	msg := record.Message

	file, line := caller(record.PC)
	file = shortFile(file)

	callerInfo := fmt.Sprintf("[%10s:%3d]", file, line)
	callerInfo = color.YellowString(callerInfo)

	var jsonStr string

	if record.NumAttrs() == 0 {
		jsonStr = ""
	} else {
		fields := make(map[string]interface{}, record.NumAttrs())
		record.Attrs(func(a slog.Attr) bool {
			fields[a.Key] = a.Value.Any()
			return true
		})

		jsonBytes, err := json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}

		jsonStr = color.WhiteString(string(jsonBytes))
	}

	handler.logger.Println(timeStr, callerInfo, level, msg, jsonStr)
	return nil
}

func newLogger(
	out io.Writer,
) *LogHandler {
	h := &LogHandler{
		logger: log.New(out, "", 0),
	}
	return h
}

func L() *slog.Logger {
	if Logger == nil {
		Logger = slog.New(newLogger(os.Stdout))
	}

	return Logger
}
