from __future__ import annotations

from dataclasses import dataclass, field

# PROVIDER_DEEPSEEK 表示模型供应商为 DeepSeek；当前供应商只支持该值。
PROVIDER_DEEPSEEK = "deepseek"


@dataclass(frozen=True)
class ConversationMemory:
    """表示 Go 召回并传入的一轮相关记忆；Python 不直接从数据库读取。"""

    user_question: str  # 用户当时提交的原始问题。
    assistant_answer: str  # 智能客服当时给出的回答。
    created_at: str = ""  # 记忆产生时间，格式为 RFC3339，仅作为上下文元数据。


@dataclass(frozen=True)
class AnswerRequest:
    """表示 Python 推理服务接收的最小必要上下文。"""

    request_id: str  # 关联 Go、Python 和计量记录的单次请求追踪标识。
    user_id: str  # 已认证的内部用户标识，不包含完整用户资料。
    session_id: str  # 当前客服会话的唯一标识。
    question: str  # 用户本次提交的问题。
    memories: list[ConversationMemory] = field(default_factory=list)  # Go 完成权限过滤后提供的 Top-N 条记忆。
    dialogue_state: dict = field(default_factory=dict)  # Go 从 Redis 读取的当前任务状态；为空表示尚未进入工作流。


@dataclass(frozen=True)
class ModelUsage:
    """表示模型供应商返回的 Token 和耗时计量信息。"""

    provider: str  # 模型供应商，当前可选值为 PROVIDER_DEEPSEEK。
    model: str  # 供应商实际使用的模型名称，例如 deepseek-chat。
    input_tokens: int  # 供应商返回的输入 Token 总数。
    cached_tokens: int  # 输入 Token 中命中供应商缓存的数量。
    output_tokens: int  # 模型生成回答消耗的输出 Token 数量。
    total_tokens: int  # 供应商返回的输入和输出 Token 总数。
    latency_ms: int  # 调用模型供应商接口的耗时，单位为毫秒。


@dataclass(frozen=True)
class AnswerResponse:
    """表示 Python 推理服务成功返回的回答和计量数据。"""

    answer: str  # 模型生成的最终客服回答。
    usage: ModelUsage  # 需要由 Go 持久化到 MySQL 的模型计量信息。
    decision: dict = field(default_factory=dict)  # Python 返回的理解结果和下一步建议，最终状态由 Go 合并。
