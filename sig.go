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
	name string
	ctx  context.Context
	span otrace.Span
}

type Log interface {
	Trace(string, ...Map)
	Info(string, ...Map)
	Debug(string, ...Map)
	Warn(string, ...Map)
	Error(error, ...Map)
	Fatal(error, ...Map)
	End()
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

func funcName() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	return runtime.FuncForPC(pc).Name()
}

func Start(ctx context.Context) Log {
	log := &log{ctx: ctx}
	if !global.ok.tracer && !global.ok.logger {
		return log
	}
	log.name = funcName()
	if global.ok.tracer {
		log.ctx, log.span = global.tracer.Start(ctx, log.name)
	}
	if global.ok.logger {
		record := olog.Record{}
		record.SetTimestamp(time.Now())
		record.SetSeverity(olog.SeverityTrace)
		record.SetBody(olog.StringValue(log.name + " started"))
		global.logger.Emit(log.ctx, record)
	}
	return log
}

func (log *log) End() {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if global.ok.logger {
		record := olog.Record{}
		record.SetTimestamp(time.Now())
		record.SetSeverity(olog.SeverityTrace)
		record.SetBody(olog.StringValue(log.name + " ended"))
		global.logger.Emit(log.ctx, record)
	}
	if global.ok.tracer {
		log.span.End()
	}
}

func (log *log) record(event string, level olog.Severity, attrsSlice ...Map) {
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
	if global.ok.tracer {
		log.span.AddEvent(event, otrace.WithAttributes(otraceAttrs...))
	}
	if global.ok.logger {
		record := olog.Record{}
		record.SetTimestamp(time.Now())
		record.SetSeverity(level)
		record.SetBody(olog.StringValue(event))
		record.AddAttributes(ologAttrs...)
		global.logger.Emit(log.ctx, record)
	}
}

func (log *log) Trace(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	log.record(event, olog.SeverityTrace, attributes...)
}

func (log *log) Info(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	log.record(event, olog.SeverityInfo, attributes...)
}

func (log *log) Debug(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	log.record(event, olog.SeverityDebug, attributes...)
}

func (log *log) Warn(event string, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
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
	event := err.Error()
	log.record(event, olog.SeverityError, attributes...)
	if global.ok.tracer {
		log.span.SetStatus(codes.Error, event)
	}
}

func (log *log) Fatal(err error, attributes ...Map) {
	if !global.ok.tracer && !global.ok.logger {
		return
	}
	if err == nil {
		return
	}
	event := err.Error()
	log.record(event, olog.SeverityFatal, attributes...)
	if global.ok.tracer {
		log.span.SetStatus(codes.Error, event)
	}
}
