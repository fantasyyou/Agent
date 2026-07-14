from __future__ import annotations

import csv
import json
from pathlib import Path
from typing import Any

from deepseek_client import call_deepseek_or_fallback
from market_data import collect_daily_quotes


DEFAULT_CSV_DIR = Path(__file__).resolve().parent / "data" / "a_share_education_2025"
DAILY_QUOTES_CSV = "a_share_education_daily_quotes.csv"


def _read_csv_rows(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as file:
        return list(csv.DictReader(file))


def _recent_csv_rows(rows: list[dict[str, str]], symbol: str, trade_date: str, limit: int) -> list[dict[str, str]]:
    filtered = [
        row
        for row in rows
        if row.get("symbol", "").upper() == symbol.upper() and row.get("date", "") <= trade_date
    ]
    filtered.sort(key=lambda row: row["date"])
    return filtered[-limit:]


def load_daily_quotes_from_csv(
    symbol: str,
    trade_date: str,
    csv_dir: str | Path = DEFAULT_CSV_DIR,
    days: int = 5,
) -> list[dict[str, Any]]:
    path = Path(csv_dir) / DAILY_QUOTES_CSV
    rows = _recent_csv_rows(_read_csv_rows(path), symbol, trade_date, days)
    return [
        {
            "date": row["date"],
            "open": float(row["open"]),
            "close": float(row["close"]),
            "high": float(row["high"]),
            "low": float(row["low"]),
            "volume": int(row["volume"]),
        }
        for row in rows
    ]


def build_market_report(symbol: str, trade_date: str, quotes: list[dict[str, Any]]) -> str:
    if not quotes:
        return f"[市场分析] {symbol} 截至 {trade_date}: CSV 中没有可用行情数据。"

    first = quotes[0]
    last = quotes[-1]
    change = round(last["close"] - first["open"], 2)
    fallback = (
        f"[市场分析] {symbol} 截至 {trade_date}: "
        f"近{len(quotes)}日收盘较首日开盘变化 {change}, "
        f"最新成交量 {last['volume']}。短期价格结构以模拟数据为准。"
    )
    system_prompt = "你是A股教育行业的市场分析师。基于给定OHLCV数据输出简洁、审慎的中文市场分析，不要编造数据。"
    user_prompt = (
        f"股票代码: {symbol}\n"
        f"分析日期: {trade_date}\n"
        f"最近行情数据JSON:\n{json.dumps(quotes, ensure_ascii=False)}\n\n"
        "请输出一段市场分析，包含价格变化、成交量观察和短期风险提示。"
    )
    return call_deepseek_or_fallback(system_prompt, user_prompt, fallback)


def run_market_agent(
    symbol: str,
    trade_date: str,
    data_source: str = "csv",
    csv_dir: str | Path = DEFAULT_CSV_DIR,
) -> dict[str, Any]:
    if data_source == "csv":
        quotes = load_daily_quotes_from_csv(symbol, trade_date, csv_dir)
    else:
        quotes = collect_daily_quotes(symbol, trade_date)

    payload: dict[str, Any] = {
        "symbol": symbol,
        "date": trade_date,
        "daily_quotes": quotes,
    }
    return {
        "role": "market",
        "symbol": symbol,
        "date": trade_date,
        "report": build_market_report(symbol, trade_date, quotes),
        "data": payload,
    }
