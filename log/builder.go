// MIT License
//
// Copyright (c) 2019 Thibault NORMAND
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package log

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// -----------------------------------------------------------------------------

// Options declares logger options for builder
type Options struct {
	LogLevel      string
	AppName       string
	AppID         string
	Version       string
	Revision      string
	Debug         bool
	DisableFields bool
}

// -----------------------------------------------------------------------------

// DefaultOptions defines default logger options
var DefaultOptions = &Options{
	Debug:         false,
	LogLevel:      "info",
	AppName:       "changeme",
	AppID:         "changeme",
	Version:       "0.0.1",
	Revision:      "123456789",
	DisableFields: false,
}

// -----------------------------------------------------------------------------

// Setup the logger
// return atomicLevel for thread-safe log level change
func Setup(ctx context.Context, opts Options) zap.AtomicLevel {

	// Initialize logs
	var config zap.Config

	if opts.Debug {
		opts.LogLevel = "debug"
		config = zap.NewDevelopmentConfig()
		config.DisableCaller = false
		config.DisableStacktrace = true
	} else {
		config = zap.NewProductionConfig()
		config.DisableCaller = true
		config.DisableStacktrace = true
		config.EncoderConfig.MessageKey = "@message"
		config.EncoderConfig.TimeKey = "@timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Parse log level
	errLogLevel := config.Level.UnmarshalText([]byte(opts.LogLevel))
	if errLogLevel != nil {
		panic(errLogLevel)
	}

	// Build real logger
	logger, err := config.Build(
		zap.AddCallerSkip(1),
	)
	if err != nil {
		panic(err)
	}

	// Add prefix to logger
	if !opts.DisableFields {
		logger = logger.With(
			zap.String("@appName", opts.AppName),
			zap.String("@version", opts.Version),
			zap.String("@revision", opts.Revision),
			zap.String("@appID", opts.AppID),
			zap.Namespace("@fields"),
		)
	}

	// Prepare factory
	logFactory := NewFactory(logger)

	// Override the global factory
	SetLoggerFactory(logFactory)

	// Override zap default logger
	zap.ReplaceGlobals(logger)

	return config.Level
}
