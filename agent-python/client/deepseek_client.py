from __future__ import annotations

import json
import logging
import os
import time
import urllib.request
from typing import Any

from model.agent_models import AnswerResponse, ModelUsage, PROVIDER_DEEPSEEK

logger = logging.getLogger(__name__)


class DeepSeekClient:
    """封装 DeepSeek HTTP 协议，不持有任何业务数据库访问能力。"""

    def __init__(self) -> None:
        self.api_key = os.environ.get("DEEPSEEK_API_KEY", "")
        self.api_url = os.environ.get("DEEPSEEK_API_URL", "https://api.deepseek.com/v1/chat/completions")
        self.model = os.environ.get("DEEPSEEK_MODEL", "deepseek-chat")
        self.timeout = int(os.environ.get("DEEPSEEK_TIMEOUT_SECONDS", "60"))

    def complete(self, system_prompt: str, user_prompt: str) -> AnswerResponse:
        self._validate_api_key()
        payload: dict[str, Any] = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "temperature": 0.2,
            "stream": False,
        }
        request = urllib.request.Request(
            self.api_url,
            data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
            headers={"Authorization": f"Bearer {self.api_key}", "Content-Type": "application/json", "Accept": "application/json"},
            method="POST",
        )
        started = time.monotonic()
        logger.info("llm_request_started", extra={"provider": PROVIDER_DEEPSEEK, "model": self.model, "timeout_seconds": self.timeout})
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                data = json.loads(response.read().decode("utf-8"))
        except Exception:
            logger.exception(
                "llm_request_failed",
                extra={"provider": PROVIDER_DEEPSEEK, "model": self.model, "duration_ms": int((time.monotonic() - started) * 1000)},
            )
            raise
        latency_ms = int((time.monotonic() - started) * 1000)
        choices = data.get("choices") or []
        if not choices:
            raise RuntimeError("DeepSeek response has no choices")
        content = choices[0].get("message", {}).get("content")
        if not content:
            raise RuntimeError("DeepSeek response has no message content")
        usage = data.get("usage") or {}
        details = usage.get("prompt_tokens_details") or {}
        logger.info(
            "llm_request_completed",
            extra={
                "provider": PROVIDER_DEEPSEEK,
                "model": str(data.get("model") or self.model),
                "input_tokens": int(usage.get("prompt_tokens") or 0),
                "cached_tokens": int(details.get("cached_tokens") or usage.get("prompt_cache_hit_tokens") or 0),
                "output_tokens": int(usage.get("completion_tokens") or 0),
                "total_tokens": int(usage.get("total_tokens") or 0),
                "duration_ms": latency_ms,
            },
        )
        return AnswerResponse(
            answer=str(content).strip(),
            usage=ModelUsage(
                provider=PROVIDER_DEEPSEEK,
                model=str(data.get("model") or self.model),
                input_tokens=int(usage.get("prompt_tokens") or 0),
                cached_tokens=int(details.get("cached_tokens") or usage.get("prompt_cache_hit_tokens") or 0),
                output_tokens=int(usage.get("completion_tokens") or 0),
                total_tokens=int(usage.get("total_tokens") or 0),
                latency_ms=latency_ms,
            ),
        )

    def _validate_api_key(self) -> None:
        key = self.api_key.strip()
        if not key:
            raise RuntimeError("DEEPSEEK_API_KEY is empty")
        if not key.isascii() or not key.startswith("sk-"):
            raise RuntimeError("DEEPSEEK_API_KEY must be an ASCII key starting with 'sk-'")
