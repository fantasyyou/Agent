from __future__ import annotations

from concurrent import futures
import logging
import time
from typing import Any

import grpc
from google.protobuf import struct_pb2

from model.agent_models import AnswerRequest, ConversationMemory
from service.customer_service import CustomerService

SERVICE_NAME = "customer.CustomerService"
logger = logging.getLogger(__name__)


class GRPCCustomerController:
    """校验 gRPC 请求，并在传输模型和业务模型之间完成转换。"""

    def __init__(self, customer_service: CustomerService) -> None:
        self.customer_service = customer_service

    def answer(self, request: struct_pb2.Struct, context: grpc.ServicerContext) -> struct_pb2.Struct:
        started = time.monotonic()
        payload: dict[str, Any] = dict(request)
        question = str(payload.get("question", "")).strip()
        if not question:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "question is required")
        memories = [
            ConversationMemory(
                user_question=str(dict(item).get("user_question", "")),
                assistant_answer=str(dict(item).get("assistant_answer", "")),
                created_at=str(dict(item).get("created_at", "")),
            )
            for item in (payload.get("memories") or [])
        ]
        application_request = AnswerRequest(
            request_id=str(payload.get("request_id", "")),
            user_id=str(payload.get("user_id", "")),
            session_id=str(payload.get("session_id", "")),
            question=question,
            memories=memories,
        )
        logger.info(
            "grpc_request_started",
            extra={
                "request_id": application_request.request_id,
                "user_id": application_request.user_id,
                "session_id": application_request.session_id,
                "memory_count": len(memories),
                "question_chars": len(question),
            },
        )
        try:
            result = self.customer_service.answer(application_request)
        except Exception as exc:
            logger.exception(
                "grpc_request_failed",
                extra={"request_id": application_request.request_id, "duration_ms": int((time.monotonic() - started) * 1000)},
            )
            context.abort(grpc.StatusCode.INTERNAL, f"LLM call failed: {exc}")
        logger.info(
            "grpc_request_completed",
            extra={
                "request_id": application_request.request_id,
                "model": result.usage.model,
                "total_tokens": result.usage.total_tokens,
                "duration_ms": int((time.monotonic() - started) * 1000),
            },
        )
        response = struct_pb2.Struct()
        response.update({
            "answer": result.answer,
            "provider": result.usage.provider,
            "model": result.usage.model,
            "input_tokens": result.usage.input_tokens,
            "cached_tokens": result.usage.cached_tokens,
            "output_tokens": result.usage.output_tokens,
            "total_tokens": result.usage.total_tokens,
            "latency_ms": result.usage.latency_ms,
        })
        return response

    def serve(self, host: str, port: int) -> None:
        server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
        handler = grpc.unary_unary_rpc_method_handler(
            self.answer,
            request_deserializer=struct_pb2.Struct.FromString,
            response_serializer=struct_pb2.Struct.SerializeToString,
        )
        server.add_generic_rpc_handlers((grpc.method_handlers_generic_handler(SERVICE_NAME, {"Answer": handler}),))
        server.add_insecure_port(f"{host}:{port}")
        server.start()
        logger.info("grpc_server_started", extra={"host": host, "port": port, "max_workers": 10})
        server.wait_for_termination()
