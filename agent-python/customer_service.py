"""Python 推理服务的依赖装配和启动入口。"""

import argparse
import logging
import os

from client.deepseek_client import DeepSeekClient
from controller.grpc_controller import GRPCCustomerController
from logging_config import configure_logging
from service.customer_service import CustomerService
from service.prompt_service import PromptService


def main() -> None:
    configure_logging()
    parser = argparse.ArgumentParser(description="Financial customer-service gRPC server")
    parser.add_argument("--host", default=os.getenv("GRPC_HOST", "0.0.0.0"))
    parser.add_argument("--port", type=int, default=int(os.getenv("GRPC_PORT", "8765")))
    args = parser.parse_args()
    logging.getLogger(__name__).info(
        "service_starting",
        extra={"host": args.host, "port": args.port, "model": os.getenv("DEEPSEEK_MODEL", "deepseek-chat")},
    )
    service = CustomerService(PromptService(), DeepSeekClient())
    GRPCCustomerController(service).serve(args.host, args.port)


if __name__ == "__main__":
    main()
