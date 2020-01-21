package httptrace

import (
	"context"
	"net/http"
	"time"

	opentracing "github.com/opentracing/opentracing-go"

	"github.com/blend/go-sdk/stats/tracing"
	"github.com/blend/go-sdk/webutil"
)

var (
	_ webutil.HTTPTracer        = (*httpTracer)(nil)
	_ webutil.HTTPTraceFinisher = (*httpTraceFinisher)(nil)
)

// Tracer returns an HTTP tracer.
func Tracer(tracer opentracing.Tracer) webutil.HTTPTracer {
	return &httpTracer{tracer: tracer}
}

type httpTracer struct {
	tracer opentracing.Tracer
}

// StartHTTPSpan opens a span and creates a new request with a modified
// context, based on the span that was opened.
func StartHTTPSpan(ctx context.Context, tracer opentracing.Tracer, req *http.Request, resource string, startTime time.Time, extra ...opentracing.StartSpanOption) (opentracing.Span, *http.Request) {
	// set up basic start options (these are mostly tags).
	startOptions := []opentracing.StartSpanOption{
		opentracing.Tag{Key: tracing.TagKeyResourceName, Value: resource},
		opentracing.Tag{Key: tracing.TagKeySpanType, Value: tracing.SpanTypeWeb},
		opentracing.Tag{Key: tracing.TagKeyHTTPMethod, Value: req.Method},
		opentracing.Tag{Key: tracing.TagKeyHTTPURL, Value: req.URL.Path},
		opentracing.Tag{Key: "http.remote_addr", Value: webutil.GetRemoteAddr(req)},
		opentracing.Tag{Key: "http.host", Value: webutil.GetHost(req)},
		opentracing.Tag{Key: "http.user_agent", Value: webutil.GetUserAgent(req)},
		opentracing.StartTime(startTime),
	}
	startOptions = append(startOptions, extra...)

	// try to extract an incoming span context
	// this is typically done if we're a service being called in a chain from another (more ancestral)
	// span context.
	spanContext, _ := tracer.Extract(opentracing.TextMap, opentracing.HTTPHeadersCarrier(req.Header))
	if spanContext != nil {
		startOptions = append(startOptions, opentracing.ChildOf(spanContext))
	}
	// start the span.
	span, spanCtx := tracing.StartSpanFromContext(ctx, tracer, tracing.OperationHTTPRequest, startOptions...)
	// inject the new context
	newReq := req.WithContext(spanCtx)
	return span, newReq
}

// Start opens a span and creates a new request with a modified context, based
// on the span that was opened.
func (ht httpTracer) Start(req *http.Request, extraKV ...webutil.TagKV) (webutil.HTTPTraceFinisher, *http.Request) {
	resource := req.URL.Path
	startTime := time.Now().UTC()

	extra := make([]opentracing.StartSpanOption, 0, len(extraKV))
	for _, pair := range extraKV {
		extra = append(extra, opentracing.Tag{Key: pair.Key, Value: pair.Value})
	}
	span, newReq := StartHTTPSpan(req.Context(), ht.tracer, req, resource, startTime, extra...)
	return &httpTraceFinisher{span: span}, newReq
}

type httpTraceFinisher struct {
	span opentracing.Span
}

func (htf httpTraceFinisher) Finish(err error) {
	if htf.span == nil {
		return
	}
	tracing.SpanError(htf.span, err)
	htf.span.Finish()
}
