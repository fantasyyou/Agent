package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-go/observability"
	"go.opentelemetry.io/otel/trace"
)

func TestRequestIdentifiersIncludeRejectedRequests(t *testing.T) {
	shutdown, err := observability.Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown(context.Background())
	var requestID, traceID string
	handler := requestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = observability.RequestID(r.Context())
		traceID = trace.SpanContextFromContext(r.Context()).TraceID().String()
		w.WriteHeader(http.StatusUnauthorized)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest("POST", "/api/chat", nil))
	if requestID == "" || len(traceID) != 32 {
		t.Fatal("missing request context")
	}
	if recorder.Header().Get("X-Request-ID") != requestID || recorder.Header().Get("X-Trace-ID") != traceID {
		t.Fatal("response identifiers do not match context")
	}
	if recorder.Code != 401 {
		t.Fatal("status changed")
	}
}
