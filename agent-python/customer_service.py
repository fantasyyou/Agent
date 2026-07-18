"""Python 推理服务的依赖装配和启动入口。"""

import argparse
import logging
import os

from client.deepseek_client import DeepSeekClient
from client.deepseek_requirement_extractor import DeepSeekRequirementExtractor
from controller.grpc_controller import GRPCCustomerController
from logging_config import configure_logging
from service.customer_service import CustomerService
from service.prompt_service import PromptService
from service.requirement_analysis_service import RequirementAnalysisService
from service.workflow_catalog import default_workflows


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
    model_client = DeepSeekClient()
    requirement_service = RequirementAnalysisService(
        DeepSeekRequirementExtractor(model_client),
        default_workflows(),
    )
    service = CustomerService(PromptService(), model_client, requirement_service)
    GRPCCustomerController(service).serve(args.host, args.port)


if __name__ == "__main__":
    main()
