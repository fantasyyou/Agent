from __future__ import annotations

import json
from typing import Any

from deepseek_client import call_deepseek_or_fallback


def build_final_report(symbol: str, trade_date: str, deps: dict[str, Any]) -> str:
    fallback = (
        f"[投资经理结论] {symbol} 截至 {trade_date}: "
        f"综合市场报告和基本面报告，当前给出模拟版中性观察结论。"
        f"市场: {deps.get('market_report', '')} "
        f"基本面: {deps.get('fundamental_report', '')}"
    )
    system_prompt = "你是投资经理。综合市场分析和基本面分析，输出审慎、可执行的中文投资观察结论，不要编造数据。"
    user_prompt = (
        f"股票代码: {symbol}\n"
        f"分析日期: {trade_date}\n"
        f"依赖报告JSON:\n{json.dumps(deps, ensure_ascii=False)}\n\n"
        "请输出最终结论，包含综合判断、主要依据、风险和观察建议。"
    )
    return call_deepseek_or_fallback(system_prompt, user_prompt, fallback)


def run_final_agent(
    symbol: str,
    trade_date: str,
    deps: dict[str, Any] | None = None,
) -> dict[str, Any]:
    payload: dict[str, Any] = {"symbol": symbol, "date": trade_date}
    payload.update(deps or {})
    return {
        "role": "final",
        "symbol": symbol,
        "date": trade_date,
        "report": build_final_report(symbol, trade_date, payload),
        "data": payload,
    }
