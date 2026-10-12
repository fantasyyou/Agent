import io
import json
import logging
import unittest
from concurrent.futures import ThreadPoolExecutor
from unittest.mock import patch

from google.protobuf import struct_pb2
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter

from client.deepseek_client import DeepSeekClient
from client.deepseek_requirement_extractor import DeepSeekRequirementExtractor
from controller.grpc_controller import GRPCCustomerController
from logging_config import JsonFormatter
from model.agent_models import AnswerResponse, ModelUsage
from observability import request_id, stage

exporter = InMemorySpanExporter()
provider = TracerProvider()
provider.add_span_processor(SimpleSpanProcessor(exporter))
trace.set_tracer_provider(provider)

class Context:
    def invocation_metadata(self):
        return [("traceparent", "00-12345678901234567890123456789012-1234567890123456-01")]
    def abort(self, code, details):
        raise RuntimeError(details)

class FakeService:
    def answer(self, request):
        with stage("fake.answer"):
            return AnswerResponse("ok", ModelUsage("fake", "fake", 100, 0, 20, 120, 1))

class ObservabilityTest(unittest.TestCase):
    def setUp(self):
        exporter.clear()

    def test_propagation_and_concurrent_request_isolation(self):
        controller = GRPCCustomerController(FakeService())
        def run(index):
            request = struct_pb2.Struct()
            request.update({"request_id": "r"+str(index), "session_id": "s"+str(index), "question": "hi"})
            controller.answer(request, Context())
            self.assertEqual("", request_id.get())
        with ThreadPoolExecutor(max_workers=10) as pool:
            list(pool.map(run, range(20)))
        spans = exporter.get_finished_spans()
        roots = [s for s in spans if s.name == "grpc.CustomerService.Answer"]
        self.assertEqual(20, len(roots))
        for root in roots:
            self.assertEqual(int("12345678901234567890123456789012", 16), root.context.trace_id)
            self.assertEqual(int("1234567890123456", 16), root.parent.span_id)
            child = next(s for s in spans if s.parent.span_id == root.context.span_id)
            self.assertEqual(root.attributes["request_id"], child.attributes["request_id"])
            self.assertEqual(root.attributes["session_id"], child.attributes["session_id"])

    def test_usage_survives_invalid_json_and_generation_failure(self):
        client = DeepSeekClient()
        usage = ModelUsage("fake", "fake", 100, 30, 20, 120, 1, True)
        with stage("extraction.extract"):
            with patch.object(client, "_complete", return_value=AnswerResponse("not-json", usage)):
                response = client.complete("secret-prompt", "secret-question")
            with self.assertRaises(ValueError):
                DeepSeekRequirementExtractor.parse_response(response.answer)
        with patch.object(client, "_complete", side_effect=TimeoutError("secret-token")):
            with self.assertRaises(TimeoutError):
                client.complete("secret-prompt", "secret-question")
        spans = exporter.get_finished_spans()
        extraction = next(s for s in spans if s.name == "llm.extract")
        self.assertEqual(120, extraction.attributes["total_tokens"])
        self.assertTrue(extraction.attributes["usage_known"])
        parsing = next(s for s in spans if s.name == "extraction.parse")
        self.assertEqual(trace.StatusCode.ERROR, parsing.status.status_code)
        generation = next(s for s in spans if s.name == "llm.generate")
        self.assertFalse(generation.attributes["usage_known"])
        self.assertNotIn("secret", str([(s.attributes, s.events, s.status.description) for s in spans]))

    def test_json_log_has_trace_and_request_identifiers(self):
        token = request_id.set("r-log")
        try:
            with stage("log.test"):
                record = logging.LogRecord("test", logging.INFO, "", 0, "test_event", (), None)
                payload = json.loads(JsonFormatter().format(record))
                self.assertEqual("r-log", payload["request_id"])
                self.assertEqual(32, len(payload["trace_id"]))
                self.assertEqual(16, len(payload["span_id"]))
        finally:
            request_id.reset(token)
