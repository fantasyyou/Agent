from __future__ import annotations

import logging

from client.deepseek_client import DeepSeekClient
from model.agent_models import AnswerRequest, AnswerResponse, ModelUsage, PROVIDER_DEEPSEEK
from model.requirement_analysis_models import (
    ANALYSIS_STATUS_NEED_CLARIFICATION,
    NEXT_ACTION_ANSWER_USER,
    NEXT_ACTION_REQUEST_HUMAN,
)
from service.prompt_service import PromptService
from service.requirement_analysis_service import RequirementAnalysisService

logger = logging.getLogger(__name__)


class CustomerService:
    """组合需求理解、确定性多轮策略和最终金融客服回答。"""

    def __init__(
        self,
        prompt_service: PromptService,
        model_client: DeepSeekClient,
        requirement_service: RequirementAnalysisService,
    ) -> None:
        self.prompt_service = prompt_service
        self.model_client = model_client
        self.requirement_service = requirement_service

    def answer(self, request: AnswerRequest) -> AnswerResponse:
        state = request.dialogue_state or {}
        workflow = state.get("workflow") if isinstance(state.get("workflow"), dict) else {}
        existing_slots = _confirmed_slot_values(state.get("slots"))
        analysis = self.requirement_service.analyze(
            user_text=request.question,
            current_intent=_string_or_none(workflow.get("intent") or state.get("intent")),
            existing_slots=existing_slots,
            last_question=str(state.get("last_question") or ""),
            active_slot=str(state.get("active_slot") or ""),
        )
        logger.info(
            "requirement_analyzed",
            extra={
                "request_id": request.request_id,
                "intent": analysis.intent,
                "status": analysis.status,
                "missing_slot_count": len(analysis.missing_slots),
                "collected_slot_count": len(analysis.slots),
                "confidence": analysis.confidence,
            },
        )
        extraction_usage = analysis.usage or _empty_usage()
        slot_updates = _build_slot_updates(existing_slots, analysis.slots, request.question, analysis.confidence)
        if analysis.status == ANALYSIS_STATUS_NEED_CLARIFICATION:
            answer = analysis.next_question
            if analysis.suggested_options:
                answer += "\n可选：" + "、".join(analysis.suggested_options)
            return AnswerResponse(
                answer=answer,
                usage=extraction_usage,
                decision=_decision(analysis, slot_updates),
            )

        if analysis.intent == "human_service":
            return AnswerResponse(
                answer="好的，我将为您转接人工客服。当前演示环境尚未连接真实工单系统。",
                usage=extraction_usage,
                decision=_decision(analysis, slot_updates, NEXT_ACTION_REQUEST_HUMAN),
            )

        if analysis.intent == "financial_consultation":
            system_prompt, user_prompt = self.prompt_service.build(request)
        else:
            system_prompt, user_prompt = self.prompt_service.build_workflow_answer(
                request, analysis.intent, analysis.slots
            )
        generated = self.model_client.complete(system_prompt, user_prompt)
        return AnswerResponse(
            answer=generated.answer,
            usage=_sum_usage(extraction_usage, generated.usage),
            decision=_decision(
                analysis,
                slot_updates,
                NEXT_ACTION_ANSWER_USER if analysis.intent == "financial_consultation" else analysis.next_action,
            ),
        )


def _confirmed_slot_values(raw_slots) -> dict:
    if not isinstance(raw_slots, dict):
        return {}
    values = {}
    for name, raw in raw_slots.items():
        if isinstance(raw, dict) and "value" in raw:
            if raw.get("status") == "confirmed":
                values[name] = raw.get("value")
        else:
            # 兼容升级前的 Redis 状态。
            values[name] = raw
    return values


def _build_slot_updates(existing: dict, current: dict, evidence: str, confidence: float) -> dict:
    return {
        name: {
            "value": value,
            "status": "confirmed",
            "source": "user",
            "evidence": evidence,
            "confidence": confidence,
        }
        for name, value in current.items()
        if name not in existing or existing[name] != value
    }


def _decision(analysis, slot_updates: dict, next_action: str | None = None) -> dict:
    return {
        "intent": analysis.intent,
        "status": analysis.status,
        "active_slot": analysis.active_slot,
        "slot_updates": slot_updates,
        "next_action": next_action or analysis.next_action,
        "next_question": analysis.next_question,
        "suggested_options": analysis.suggested_options,
    }


def _string_or_none(value) -> str | None:
    text = str(value or "").strip()
    return text or None


def _empty_usage() -> ModelUsage:
    return ModelUsage(PROVIDER_DEEPSEEK, "deepseek-chat", 0, 0, 0, 0, 0)


def _sum_usage(first: ModelUsage, second: ModelUsage) -> ModelUsage:
    return ModelUsage(
        provider=second.provider or first.provider,
        model=second.model or first.model,
        input_tokens=first.input_tokens + second.input_tokens,
        cached_tokens=first.cached_tokens + second.cached_tokens,
        output_tokens=first.output_tokens + second.output_tokens,
        total_tokens=first.total_tokens + second.total_tokens,
        latency_ms=first.latency_ms + second.latency_ms,
    )
