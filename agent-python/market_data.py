from __future__ import annotations

import csv
import json
import time
import urllib.parse
import urllib.request
from dataclasses import asdict, dataclass
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from typing import Iterable, List
from urllib.error import HTTPError, URLError


A_SHARE_EDUCATION_STOCKS = {
    "002607.SZ": "中公教育",
    "003032.SZ": "传智教育",
    "300010.SZ": "豆神教育",
    "300338.SZ": "开元教育",
    "600661.SS": "昂立教育",
}

# Backwards-compatible name used by fundamental_data.py and the CLI flag.
EDUCATION_STOCKS = A_SHARE_EDUCATION_STOCKS


@dataclass
class DailyQuote:
    date: str
    open: float
    close: float
    high: float
    low: float
    volume: int


def _get_json(url: str) -> dict:
    request = urllib.request.Request(
        url,
        headers={
            "User-Agent": "reservesanalysis/0.1",
            "Accept": "application/json",
        },
    )
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                return json.loads(response.read().decode("utf-8"))
        except (HTTPError, URLError, TimeoutError) as exc:
            last_error = exc
            if attempt == 2:
                break
            time.sleep(1.5 * (attempt + 1))
    raise RuntimeError(f"Failed to fetch JSON from {url}: {last_error}") from last_error


def _to_unix(day: str, inclusive_end: bool = False) -> int:
    value = date.fromisoformat(day)
    if inclusive_end:
        value = value + timedelta(days=1)
    return int(datetime(value.year, value.month, value.day, tzinfo=timezone.utc).timestamp())


def collect_daily_quotes(
    symbol: str,
    end_date: str,
    days: int = 5,
    start_date: str | None = None,
) -> List[dict]:
    """Fetch real daily OHLCV data from Yahoo Finance chart API."""
    use_recent_window = start_date is None
    if start_date is None:
        end = date.fromisoformat(end_date)
        start_date = (end - timedelta(days=days * 2)).isoformat()

    params = urllib.parse.urlencode(
        {
            "period1": _to_unix(start_date),
            "period2": _to_unix(end_date, inclusive_end=True),
            "interval": "1d",
            "events": "history",
        }
    )
    url = f"https://query1.finance.yahoo.com/v8/finance/chart/{urllib.parse.quote(symbol)}?{params}"
    payload = _get_json(url)

    chart = payload.get("chart", {})
    if chart.get("error"):
        raise RuntimeError(f"Yahoo Finance error for {symbol}: {chart['error']}")

    result = chart.get("result") or []
    if not result:
        return []

    timestamps = result[0].get("timestamp") or []
    quote = (result[0].get("indicators", {}).get("quote") or [{}])[0]
    rows: list[DailyQuote] = []

    for idx, ts in enumerate(timestamps):
        open_price = quote.get("open", [None])[idx]
        close_price = quote.get("close", [None])[idx]
        high_price = quote.get("high", [None])[idx]
        low_price = quote.get("low", [None])[idx]
        volume = quote.get("volume", [None])[idx]
        if None in (open_price, close_price, high_price, low_price, volume):
            continue

        rows.append(
            DailyQuote(
                date=datetime.fromtimestamp(ts, tz=timezone.utc).date().isoformat(),
                open=round(float(open_price), 4),
                close=round(float(close_price), 4),
                high=round(float(high_price), 4),
                low=round(float(low_price), 4),
                volume=int(volume),
            )
        )

    if use_recent_window:
        rows = rows[-days:]
    return [asdict(row) for row in rows]


def write_daily_quotes_csv(symbols: Iterable[str], start_date: str, end_date: str, output_dir: str) -> list[str]:
    Path(output_dir).mkdir(parents=True, exist_ok=True)
    path = Path(output_dir) / "a_share_education_daily_quotes.csv"

    with path.open("w", newline="", encoding="utf-8") as file:
        fieldnames = ["symbol", "company", "date", "open", "close", "high", "low", "volume"]
        writer = csv.DictWriter(file, fieldnames=fieldnames)
        writer.writeheader()

        for symbol in symbols:
            company = EDUCATION_STOCKS.get(symbol, "")
            for row in collect_daily_quotes(symbol, end_date, start_date=start_date):
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

    parser = argparse.ArgumentParser(description="Collect real daily quotes for A-share education stocks.")
    parser.add_argument("--symbol")
    parser.add_argument("--symbols")
    parser.add_argument("--education-stocks", action="store_true")
    parser.add_argument("--date", default=date.today().isoformat())
    parser.add_argument("--start-date")
    parser.add_argument("--end-date")
    parser.add_argument("--days", type=int, default=5)
    parser.add_argument("--output-dir")
    args = parser.parse_args()

    symbols = _parse_symbols(args.symbols or args.symbol, args.education_stocks)
    end_date = args.end_date or args.date

    if args.output_dir:
        paths = write_daily_quotes_csv(symbols, args.start_date or end_date, end_date, args.output_dir)
        print(json.dumps({"files": paths}, ensure_ascii=False))
    elif len(symbols) == 1:
        print(
            json.dumps(
                collect_daily_quotes(symbols[0], end_date, args.days, args.start_date),
                ensure_ascii=False,
            )
        )
    else:
        data = {symbol: collect_daily_quotes(symbol, end_date, args.days, args.start_date) for symbol in symbols}
        print(json.dumps(data, ensure_ascii=False))
