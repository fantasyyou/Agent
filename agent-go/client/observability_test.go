package client

import (
	"agent-go/config"
	"agent-go/model"
	"agent-go/observability"
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"net"
	"testing"
	"time"
)

type traceService interface{}

func TestGRPCPropagatesTraceContext(t *testing.T) {
	shutdown, err := observability.Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown(context.Background())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	var received trace.SpanContext
	server.RegisterService(&grpc.ServiceDesc{ServiceName: "customer.CustomerService", HandlerType: (*traceService)(nil), Methods: []grpc.MethodDesc{{MethodName: "Answer", Handler: func(_ interface{}, ctx context.Context, decode func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
		req := &structpb.Struct{}
		if err := decode(req); err != nil {
			return nil, err
		}
		md, _ := metadata.FromIncomingContext(ctx)
		carrier := propagation.MapCarrier{}
		for key, values := range md {
			if len(values) > 0 {
				carrier[key] = values[0]
			}
		}
		received = trace.SpanContextFromContext(otel.GetTextMapPropagator().Extract(ctx, carrier))
		return structpb.NewStruct(map[string]interface{}{"answer": "ok"})
	}}}}, struct{}{})
	go server.Serve(listener)
	defer server.Stop()
	python, err := NewPythonAgent(config.PythonAgentConfig{Address: listener.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}
	defer python.Close()
	ctx, span := observability.Start(observability.WithRequest(context.Background()), "test.root")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = python.Answer(ctx, model.AgentRequest{RequestID: observability.RequestID(ctx), Question: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if !received.IsRemote() || received.TraceID() != span.SpanContext().TraceID() || received.SpanID() == span.SpanContext().SpanID() {
		t.Fatal("gRPC did not inject child span context")
	}
}
