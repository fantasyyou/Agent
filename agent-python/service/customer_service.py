from __future__ import annotations

from client.deepseek_client import DeepSeekClient
from model.agent_models import AnswerRequest, AnswerResponse
from service.prompt_service import PromptService


class CustomerService:
    """Coordinates prompt construction and one provider inference call."""

    def __init__(self, prompt_service: PromptService, model_client: DeepSeekClient) -> None:
        self.prompt_service = prompt_service
        self.model_client = model_client

    def answer(self, request: AnswerRequest) -> AnswerResponse:
        system_prompt, user_prompt = self.prompt_service.build(request)
        return self.model_client.complete(system_prompt, user_prompt)
