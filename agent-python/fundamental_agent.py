from __future__ import annotations

import csv
import json
from pathlib import Path
from typing import Any

from deepseek_client import call_deepseek_or_fallback
from fundamental_data import collect_fundamentals


DEFAULT_CSV_DIR = Path(__file__).resolve().parent / "data" / "a_share_education_2025"
FUNDAMENTALS_CSV = "a_share_education_fundamentals.csv"


def _parse_float(value: str) -> float | None:
    if value == "":
        return None
    return float(value)


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


def load_fundamentals_from_csv(
    symbol: str,
    trade_date: str,
    csv_dir: str | Path = DEFAULT_CSV_DIR,
    periods: int = 4,
) -> list[dict[str, Any]]:
    path = Path(csv_dir) / FUNDAMENTALS_CSV
    rows = _recent_csv_rows(_read_csv_rows(path), symbol, trade_date, periods)
    return [
        {
            "date": row["date"],
            "revenue": _parse_float(row["revenue"]),
            "net_profit": _parse_float(row["net_profit"]),
            "navps": _parse_float(row["navps"]),
        }
        for row in rows
    ]


def build_fundamental_report(symbol: str, trade_date: str, rows: list[dict[str, Any]]) -> str:
    if not rows:
        return f"[基本面分析] {symbol} 截至 {trade_date}: CSV 中没有可用基本面数据。"

    latest = rows[-1]
    revenue = latest["revenue"] or 0
    net_profit = latest["net_profit"] or 0
    navps = latest["navps"] or 0
    margin = net_profit / revenue if revenue else 0
    fallback = (
        f"[基本面分析] {symbol} 最新营收 {revenue:.2f}, "
        f"净利润 {net_profit:.2f}, 净利率 {margin:.2%}, "
        f"每股净资产 {navps:.2f}。"
    )
    system_prompt = "你是A股教育行业的基本面分析师。基于给定财务数据输出简洁、审慎的中文基本面分析，不要编造数据。"
    user_prompt = (
        f"股票代码: {symbol}\n"
        f"分析日期: {trade_date}\n"
        f"基本面数据JSON:\n{json.dumps(rows, ensure_ascii=False)}\n\n"
        "请输出一段基本面分析，包含营收、净利润、净利率、每股净资产和主要风险。"
    )
    return call_deepseek_or_fallback(system_prompt, user_prompt, fallback)


def run_fundamental_agent(
    symbol: str,
    trade_date: str,
    data_source: str = "csv",
    csv_dir: str | Path = DEFAULT_CSV_DIR,
) -> dict[str, Any]:
    if data_source == "csv":
        rows = load_fundamentals_from_csv(symbol, trade_date, csv_dir)
    else:
        rows = collect_fundamentals(symbol, trade_date)

    payload: dict[str, Any] = {
        "symbol": symbol,
        "date": trade_date,
        "fundamentals": rows,
    }
    return {
        "role": "fundamental",
        "symbol": symbol,
        "date": trade_date,
        "report": build_fundamental_report(symbol, trade_date, rows),
        "data": payload,
    }
