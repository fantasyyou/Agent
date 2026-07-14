from __future__ import annotations

import csv
import json
from dataclasses import asdict, dataclass
from datetime import date
from pathlib import Path
from typing import Iterable, List

import pandas as pd
import yfinance as yf

from market_data import EDUCATION_STOCKS


REVENUE_FIELDS = [
    "Total Revenue",
    "Operating Revenue",
]
NET_PROFIT_FIELDS = [
    "Net Income",
    "Net Income Common Stockholders",
    "Net Income Continuous Operations",
]
EQUITY_FIELDS = [
    "Stockholders Equity",
    "Common Stock Equity",
    "Total Equity Gross Minority Interest",
]
SHARE_FIELDS = [
    "Ordinary Shares Number",
    "Share Issued",
    "Common Stock Shares Outstanding",
]


@dataclass
class FundamentalRow:
    date: str
    revenue: float | None
    net_profit: float | None
    navps: float | None


def _field_value(frame: pd.DataFrame, fields: list[str], report_date: pd.Timestamp) -> float | None:
    if frame.empty or report_date not in frame.columns:
        return None

    normalized_index = {str(index).strip().lower(): index for index in frame.index}
    for field in fields:
        actual_field = normalized_index.get(field.lower())
        if actual_field is None:
            continue
        value = frame.at[actual_field, report_date]
        if pd.isna(value):
            continue
        try:
            return float(value)
        except (TypeError, ValueError):
            continue
    return None


def _statement_dates(*frames: pd.DataFrame) -> list[pd.Timestamp]:
    values = set()
    for frame in frames:
        if frame.empty:
            continue
        for column in frame.columns:
            values.add(pd.Timestamp(column).normalize())
    return sorted(values)


def collect_fundamentals(
    symbol: str,
    end_date: str,
    periods: int = 4,
    start_date: str | None = None,
) -> List[dict]:
    """Fetch real fundamentals for A-shares from Yahoo Finance statements.

    A-share financial reports are disclosure based. If a quarter has not been
    published yet, it will not appear in the output.
    """
    if start_date is None:
        year = date.fromisoformat(end_date).year
        start_date = f"{year}-01-01"

    ticker = yf.Ticker(symbol)
    income = ticker.quarterly_income_stmt
    balance = ticker.quarterly_balance_sheet
    if income.empty:
        income = ticker.income_stmt
    if balance.empty:
        balance = ticker.balance_sheet

    rows: list[FundamentalRow] = []
    for report_timestamp in _statement_dates(income, balance):
        report_date = report_timestamp.date().isoformat()
        if not (start_date <= report_date <= end_date):
            continue
        equity = _field_value(balance, EQUITY_FIELDS, report_timestamp)
        shares = _field_value(balance, SHARE_FIELDS, report_timestamp)
        rows.append(
            FundamentalRow(
                date=report_date,
                revenue=_field_value(income, REVENUE_FIELDS, report_timestamp),
                net_profit=_field_value(income, NET_PROFIT_FIELDS, report_timestamp),
                navps=round(equity / shares, 4) if equity and shares else None,
            )
        )

    rows.sort(key=lambda row: row.date)
    return [asdict(row) for row in rows[-periods:]]


def write_fundamentals_csv(symbols: Iterable[str], start_date: str, end_date: str, output_dir: str) -> list[str]:
    Path(output_dir).mkdir(parents=True, exist_ok=True)
    path = Path(output_dir) / "a_share_education_fundamentals.csv"

    with path.open("w", newline="", encoding="utf-8") as file:
        fieldnames = ["symbol", "company", "date", "revenue", "net_profit", "navps"]
        writer = csv.DictWriter(file, fieldnames=fieldnames)
        writer.writeheader()

        for symbol in symbols:
            company = EDUCATION_STOCKS.get(symbol, "")
            for row in collect_fundamentals(symbol, end_date, start_date=start_date):
                writer.writerow({"symbol": symbol, "company": company, **row})

    return [str(path)]


def _parse_symbols(raw: str | None, use_education_stocks: bool) -> list[str]:
    if use_education_stocks:
        return list(EDUCATION_STOCKS)
    if raw:
        return [item.strip().upper() for item in raw.split(",") if item.strip()]
    raise ValueError("provide --symbol, --symbols, or --education-stocks")


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Collect real fundamentals for A-share education stocks.")
    parser.add_argument("--symbol")
    parser.add_argument("--symbols")
    parser.add_argument("--education-stocks", action="store_true")
    parser.add_argument("--date", default=date.today().isoformat())
    parser.add_argument("--start-date")
    parser.add_argument("--end-date")
    parser.add_argument("--periods", type=int, default=4)
    parser.add_argument("--output-dir")
    args = parser.parse_args()

    symbols = _parse_symbols(args.symbols or args.symbol, args.education_stocks)
    end_date = args.end_date or args.date
    start_date = args.start_date or f"{date.fromisoformat(end_date).year}-01-01"

    if args.output_dir:
        paths = write_fundamentals_csv(symbols, start_date, end_date, args.output_dir)
        print(json.dumps({"files": paths}, ensure_ascii=False))
    elif len(symbols) == 1:
        print(
            json.dumps(
                collect_fundamentals(symbols[0], end_date, args.periods, start_date),
                ensure_ascii=False,
            )
        )
    else:
        data = {symbol: collect_fundamentals(symbol, end_date, args.periods, start_date) for symbol in symbols}
        print(json.dumps(data, ensure_ascii=False))
