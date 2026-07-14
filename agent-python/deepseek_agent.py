from __future__ import annotations

import argparse
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

from final_agent import run_final_agent
from fundamental_agent import run_fundamental_agent
from market_agent import DEFAULT_CSV_DIR, run_market_agent


def run_agent(
    role: str,
    symbol: str,
    trade_date: str,
    deps: dict[str, Any] | None = None,
    data_source: str = "csv",
    csv_dir: str | Path = DEFAULT_CSV_DIR,
) -> dict[str, Any]:
    if role == "market":
        return run_market_agent(symbol, trade_date, data_source, csv_dir)
    if role == "fundamental":
        return run_fundamental_agent(symbol, trade_date, data_source, csv_dir)
    if role == "final":
        return run_final_agent(symbol, trade_date, deps)
    raise ValueError("role must be one of: market, fundamental, final")


def run_workflow(
    symbol: str,
    trade_date: str,
    data_source: str = "csv",
    csv_dir: str | Path = DEFAULT_CSV_DIR,
) -> dict[str, Any]:
    market = run_market_agent(symbol, trade_date, data_source, csv_dir)
    fundamental = run_fundamental_agent(symbol, trade_date, data_source, csv_dir)
    final = run_final_agent(
        symbol,
        trade_date,
        {
            "market_report": market["report"],
            "fundamental_report": fundamental["report"],
        },
    )
    return {
        "symbol": symbol,
        "date": trade_date,
        "market": market,
        "fundamental": fundamental,
        "final": final,
    }


class AgentRequestHandler(BaseHTTPRequestHandler):
    server_version = "DeepSeekAgentMock/0.1"

    def _send_json(self, status: int, payload: dict[str, Any]) -> None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_json_body(self) -> dict[str, Any]:
        length = int(self.headers.get("Content-Length", "0"))
        if length == 0:
            return {}
        raw = self.rfile.read(length).decode("utf-8")
        return json.loads(raw)

    def do_GET(self) -> None:
        if self.path == "/health":
            self._send_json(200, {"status": "ok"})
            return
        self._send_json(404, {"error": "not found"})

    def do_POST(self) -> None:
        try:
            body = self._read_json_body()
            if self.path == "/agent":
                result = run_agent(
                    role=body["role"],
                    symbol=body["symbol"],
                    trade_date=body["date"],
                    deps=body.get("deps"),
                    data_source=body.get("data_source", "csv"),
                    csv_dir=body.get("csv_dir", DEFAULT_CSV_DIR),
                )
                self._send_json(200, result)
                return

            if self.path == "/workflow":
                result = run_workflow(
                    symbol=body["symbol"],
                    trade_date=body["date"],
                    data_source=body.get("data_source", "csv"),
                    csv_dir=body.get("csv_dir", DEFAULT_CSV_DIR),
                )
                self._send_json(200, result)
                return

            self._send_json(404, {"error": "not found"})
        except KeyError as exc:
            self._send_json(400, {"error": f"missing required field: {exc.args[0]}"})
        except json.JSONDecodeError as exc:
            self._send_json(400, {"error": f"invalid json: {exc}"})
        except Exception as exc:
            self._send_json(500, {"error": str(exc)})

    def log_message(self, format: str, *args: Any) -> None:
        print(f"{self.address_string()} - {format % args}", file=sys.stderr)


def serve(host: str, port: int) -> None:
    server = ThreadingHTTPServer((host, port), AgentRequestHandler)
    print(f"DeepSeek agent mock service listening on http://{host}:{port}", file=sys.stderr)
    server.serve_forever()


def main() -> int:
    parser = argparse.ArgumentParser(description="Run simulated DeepSeek financial agents.")
    parser.add_argument("--serve", action="store_true", help="Start the HTTP service.")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8765)
    parser.add_argument("--role", choices=["market", "fundamental", "final"])
    parser.add_argument("--symbol")
    parser.add_argument("--date")
    parser.add_argument("--deps", default="{}", help="JSON dependencies for final role.")
    parser.add_argument("--data-source", choices=["csv", "online"], default="csv")
    parser.add_argument("--csv-dir", default=str(DEFAULT_CSV_DIR))
    args = parser.parse_args()

    try:
        if args.serve:
            serve(args.host, args.port)
            return 0
        if not args.role or not args.symbol or not args.date:
            parser.error("--role, --symbol, and --date are required unless --serve is used")
        result = run_agent(
            args.role,
            args.symbol,
            args.date,
            json.loads(args.deps),
            args.data_source,
            args.csv_dir,
        )
    except Exception as exc:
        print(json.dumps({"error": str(exc)}, ensure_ascii=False), file=sys.stderr)
        return 1

    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    # python deepseek_agent.py --serve --host 127.0.0.1 --port 8765
    raise SystemExit(main())
