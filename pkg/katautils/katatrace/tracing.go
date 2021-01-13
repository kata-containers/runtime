// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package katatrace

import (
	"context"
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go/config"
)

// Implements jaeger-client-go.Logger interface
type traceLogger struct {
}

var kataTraceLogger = logrus.NewEntry(logrus.New())
var TracingSet = false

// tracerCloser contains a copy of the closer returned by createTracer() which
// is used by stopTracing().
var tracerCloser io.Closer

func (t traceLogger) Error(msg string) {
	kataTraceLogger.Error(msg)
}

func (t traceLogger) Infof(msg string, args ...interface{}) {
	kataTraceLogger.Infof(msg, args...)
}

// SetLogger sets the logger to be used by katatrace, this fuction is
// implicitly called by katautils.SetLogger
func SetLogger(logger *logrus.Entry) {
	kataTraceLogger = logger
}

// CreateTracer create a tracer
func CreateTracer(name string) (opentracing.Tracer, error) {
	cfg := &config.Configuration{
		ServiceName: name,

		// If tracing is disabled, use a NOP trace implementation
		Disabled: !TracingSet,

		// Note that span logging reporter option cannot be enabled as
		// it pollutes the output stream which causes (atleast) the
		// "state" command to fail under Docker.
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},

		// Ensure that Jaeger logs each span.
		// This is essential as it is used by:
		//
		// https: //github.com/kata-containers/tests/blob/master/tracing/tracing-test.sh
		Reporter: &config.ReporterConfig{
			LogSpans: TracingSet,
		},
	}

	logger := traceLogger{}

	tracer, closer, err := cfg.NewTracer(config.Logger(logger))
	if err != nil {
		return nil, err
	}

	// save for stopTracing()'s exclusive use
	tracerCloser = closer

	// Seems to be essential to ensure non-root spans are logged
	opentracing.SetGlobalTracer(tracer)

	return tracer, nil
}

// StopTracing ends all tracing, reporting the spans to the collector.
func StopTracing(ctx context.Context) {
	if !TracingSet {
		return
	}

	span := opentracing.SpanFromContext(ctx)

	if span != nil {
		span.Finish()
	}

	// report all possible spans to the collector
	if tracerCloser != nil {
		tracerCloser.Close()
	}
}

// Trace creates a new tracing span based on the specified name and parent
// context. Accepts a logger to report tracing errors on and a variadic
// number of tags in key-value form (key1, value1, key2, value2, ...).
// The number of tags should be even.
func Trace(parent context.Context, logger *logrus.Entry, name string, tags ...string) (opentracing.Span, context.Context) {
	if parent == nil {
		if logger == nil {
			logger = kataTraceLogger
		}
		logger.WithField("type", "bug").Error("trace called before context set")
		parent = context.Background()
	}

	span, ctx := opentracing.StartSpanFromContext(parent, name)

	for i := 0; i < len(tags); i += 2 {
		if i+1 == len(tags) {
			span.SetTag(tags[i], "")
		} else {
			span.SetTag(tags[i], tags[i+1])
		}
	}

	// This is slightly confusing: when tracing is disabled, trace spans
	// are still created - but the tracer used is a NOP. Therefore, only
	// display the message when tracing is really enabled.
	if TracingSet {
		// This log message is *essential*: it is used by:
		// https: //github.com/kata-containers/tests/blob/master/tracing/tracing-test.sh
		kataTraceLogger.Debugf("created span %v", span)
	}

	return span, ctx
}
