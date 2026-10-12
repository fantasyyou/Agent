"""Request-scoped tracing; never capture prompts, credentials or raw exceptions."""
from contextlib import contextmanager
from contextvars import ContextVar
from functools import wraps
import logging
import os
import time

from opentelemetry import trace, propagate
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.trace import Status, StatusCode

request_id = ContextVar("request_id", default="")
session_id = ContextVar("session_id", default="")
logger = logging.getLogger(__name__)

def configure_tracing():
    provider = TracerProvider(resource=Resource.create({"service.name": "agent-python"}))
    if os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT") or os.getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"):
        provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter(timeout=3)))
    trace.set_tracer_provider(provider)
    return provider

@contextmanager
def stage(name, attributes=None, context=None):
    started = time.monotonic()
    with trace.get_tracer("agent-python").start_as_current_span(
        name, context=context, attributes=attributes or {},
        record_exception=False, set_status_on_exception=False,
    ) as span:
        span.set_attribute("request_id", request_id.get())
        span.set_attribute("session_id", session_id.get())
        result = "success"
        try:
            yield span
        except Exception as exc:
            result = "failed"
            span.set_status(Status(StatusCode.ERROR, name + " failed"))
            span.set_attribute("error.type", type(exc).__name__)
            raise
        finally:
            logger.info("stage_completed", extra={"stage": name, "result": result,
                "duration_ms": int((time.monotonic()-started)*1000)})

def traced(name):
    def decorator(fn):
        @wraps(fn)
        def wrapper(*args, **kwargs):
            with stage(name):
                return fn(*args, **kwargs)
        return wrapper
    return decorator

def grpc_traced(fn):
    @wraps(fn)
    def wrapper(self, request, context):
        payload = request.AsMap() if hasattr(request, "AsMap") else dict(request)
        rid = request_id.set(str(payload.get("request_id", "")))
        sid = session_id.set(str(payload.get("session_id", "")))
        metadata = dict(context.invocation_metadata()) if hasattr(context, "invocation_metadata") else {}
        try:
            with stage("grpc.CustomerService.Answer", context=propagate.extract(metadata)):
                return fn(self, request, context)
        finally:
            request_id.reset(rid)
            session_id.reset(sid)
    return wrapper
