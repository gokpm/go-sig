package sig

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	olog "go.opentelemetry.io/otel/log"
	ometric "go.opentelemetry.io/otel/metric"
	otrace "go.opentelemetry.io/otel/trace"
)

var global struct {
	ok struct {
		tracer bool
		meter  bool
		logger bool
	}
	tracer otrace.Tracer
	meter  ometric.Meter
	logger olog.Logger
}

type Map map[string]any

type log struct {
	function string
	file     string
	ctx      context.Context
	span     otrace.Span
}

type Log interface {
	Trace(string, ...Map)
	Info(string, ...Map)
	Debug(string, ...Map)
	Warn(string, ...Map)
	Error(error, ...Map)
	Fatal(error, ...Map)
	End()
	Ctx() context.Context
}

func Setup(tracer otrace.Tracer, meter ometric.Meter, logger olog.Logger) {
	if tracer != nil {
		global.ok.tracer = true
		global.tracer = tracer
	}
	if meter != nil {
		global.ok.meter = true
		global.meter = meter
	}
	if logger != nil {
		global.ok.logger = true
		global.logger = logger
	}
}

func callerMeta() (string, string, int) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "", "", 0
	}
	return runtime.FuncForPC(pc).Name(), file, line
}

func callerLine(skip int) int {
	_, _, line, ok := runtime.Caller(skip)
	if !ok {
		return 0
	}
	return line
}

func Start(ctx context.Context) Log {
	log := &log{ctx: ctx}
	if !global.ok.tracer && !global.ok.logger {
		return log
	}
	now := time.Now()
	var line int
	log.function, log.file, line = callerMeta()
	if global.ok.tracer {
		log.ctx, log.span = global.tracer.Start(
			ctx,
			log.function,
			otrace.WithTimestamp(now),
			otrace.WithAttributes(
				attribute.String("file", log.file),
				attribute.Int("line", line),
			),
		)
	}
	if global.ok.logger {
		record := olog.Record{}
		record.SetBody(olog.StringValue("started"))
		record.SetTimestamp(now)
		record.SetSeverity(olog.SeverityTrace)
		record.SetSeverityText(olog.SeverityTrace.String())
		record.AddAttributes(
			olog.String("function", log.function),
			olog.String("file", log.file),
			olog.Int("line", line),
		)
		global.logger.Emit(log.ctx, record)
	}
	return log
}

func (log *log) Ctx() context.Context {
	return log.ctx
}

func (log *log) End() {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	now := time.Now()
	line := callerLine(2)
	if global.ok.logger {
		record := olog.Record{}
		record.SetBody(olog.StringValue("ended"))
		record.SetTimestamp(now)
		record.SetSeverity(olog.SeverityTrace)
		record.SetSeverityText(olog.SeverityTrace.String())
		record.AddAttributes(
			olog.String("function", log.function),
			olog.String("file", log.file),
			olog.Int("line", line),
		)
		global.logger.Emit(log.ctx, record)
	}
	if global.ok.tracer {
		log.span.End(otrace.WithTimestamp(now))
	}
}

func (log *log) record(event string, level olog.Severity, attrsSlice ...Map) {
	now := time.Now()
	var otraceAttrs []attribute.KeyValue
	var ologAttrs []olog.KeyValue
	if global.ok.tracer {
		otraceAttrs = []attribute.KeyValue{}
	}
	if global.ok.logger {
		ologAttrs = []olog.KeyValue{}
	}
	for _, attrs := range attrsSlice {
		for key, value := range attrs {
			if global.ok.tracer {
				otraceAttr := attribute.String(key, fmt.Sprint(value))
				otraceAttrs = append(otraceAttrs, otraceAttr)
			}
			if global.ok.logger {
				ologAttr := olog.String(key, fmt.Sprint(value))
				ologAttrs = append(ologAttrs, ologAttr)
			}
		}
	}
	line := callerLine(3)
	if global.ok.tracer {
		otraceAttrs = append(
			otraceAttrs,
			attribute.String("file", log.file),
			attribute.Int("line", line),
		)
		options := []otrace.EventOption{
			otrace.WithTimestamp(now),
			otrace.WithAttributes(otraceAttrs...),
		}
		if level >= olog.SeverityError {
			log.span.AddEvent(event, options...)
			log.span.SetStatus(codes.Error, event)
		} else {
			log.span.AddEvent(event, options...)
		}
	}
	if global.ok.logger {
		record := olog.Record{}
		record.SetBody(olog.StringValue(event))
		record.SetTimestamp(now)
		record.SetSeverity(level)
		record.SetSeverityText(level.String())
		record.AddAttributes(
			olog.String("function", log.function),
			olog.String("file", log.file),
			olog.Int("line", line),
		)
		global.logger.Emit(log.ctx, record)
	}
}

func (log *log) Trace(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if event == "" {
		return
	}
	log.record(event, olog.SeverityTrace, attributes...)
}

func (log *log) Info(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if event == "" {
		return
	}
	log.record(event, olog.SeverityInfo, attributes...)
}

func (log *log) Debug(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if event == "" {
		return
	}
	log.record(event, olog.SeverityDebug, attributes...)
}

func (log *log) Warn(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if event == "" {
		return
	}
	log.record(event, olog.SeverityWarn, attributes...)
}

func (log *log) Error(err error, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if err == nil {
		return
	}
	log.record(err.Error(), olog.SeverityError, attributes...)
}

func (log *log) Fatal(err error, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if err == nil {
		return
	}
	log.record(err.Error(), olog.SeverityFatal, attributes...)
}
