// Copyright 2023 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package slog provides a log interface to be compatible with "log/slog".
package slog

import (
	"context"
	"io"
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

// Predefine some log level.
const (
	LevelTrace = slog.LevelDebug - 4
	LevelDebug = slog.LevelDebug
)

// SetDiscardWriter sets the log writer to discard all log outputs.
func SetDiscardWriter() {
	handler := slog.HandlerOptions{}.NewJSONHandler(io.Discard)
	slog.SetDefault(slog.New(handler))
}

// Enabled reports whether the level is enabled.
func Enabled(ctx context.Context, level slog.Level) bool {
	return slog.Default().Enabled(ctx, level)
}

// Trace emits an log with the Trace level.
func Trace(msg string, args ...any) { emit(1, LevelTrace, msg, args...) }

// Debug emits an log with the Debug level.
func Debug(msg string, args ...any) { emit(1, LevelDebug, msg, args...) }

// Error emits an log with the Error level.
func Error(msg string, args ...any) { emit(1, slog.LevelError, msg, args...) }

func emit(skipStackDepth int, level slog.Level, msg string, kvs ...interface{}) {
	if !Enabled(context.Background(), level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(skipStackDepth+2, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(kvs...)
	slog.Default().Handler().Handle(context.Background(), r)
}
