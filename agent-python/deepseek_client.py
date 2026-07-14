from __future__ import annotations

import json
import os
import urllib.request
from typing import Any


# Put your key here, or set DEEPSEEK_API_KEY in the environment.
DEEPSEEK_API_KEY = os.environ.get("DEEPSEEK_API_KEY", "")
DEEPSEEK_API_URL = os.environ.get("DEEPSEEK_API_URL", "https://api.deepseek.com/v1/chat/completions")
DEEPSEEK_MODEL = os.environ.get("DEEPSEEK_MODEL", "deepseek-chat")
DEEPSEEK_TIMEOUT_SECONDS = 60


def is_deepseek_configured() -> bool:
    return bool(DEEPSEEK_API_KEY.strip())

def call_deepseek(
    system_prompt: str,
    user_prompt: str,
    temperature: float = 0.2,
) -> str:
    if not is_deepseek_configured():
        raise RuntimeError("DEEPSEEK_API_KEY is empty")

    payload: dict[str, Any] = {
        "model": DEEPSEEK_MODEL,
        "messages": [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt},
        ],
        "temperature": temperature,
        "stream": False,
    }
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    request = urllib.request.Request(
        DEEPSEEK_API_URL,
        data=body,
        headers={
            "Authorization": f"Bearer {DEEPSEEK_API_KEY}",
            "Content-Type": "application/json",
            "Accept": "application/json",
        },
        method="POST",
    )

    with urllib.request.urlopen(request, timeout=DEEPSEEK_TIMEOUT_SECONDS) as response:
        data = json.loads(response.read().decode("utf-8"))

    choices = data.get("choices") or []
    if not choices:
        raise RuntimeError(f"DeepSeek response has no choices: {data}")
    content = choices[0].get("message", {}).get("content")
    if not content:
        raise RuntimeError(f"DeepSeek response has no message content: {data}")
    return str(content).strip()


def call_deepseek_or_fallback(
    system_prompt: str,
    user_prompt: str,
    fallback: str,
    temperature: float = 0.2,
) -> str:
    try:
        return call_deepseek(system_prompt, user_prompt, temperature)
    except Exception as exc:
        return f"{fallback}\n\n[DeepSeek调用未启用或失败: {exc}]"
