from __future__ import annotations

import argparse
import json
import random
from pathlib import Path
from typing import Any


def load_config(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def resolve_from_config(config_path: Path, value: str) -> Path:
    return (config_path.parent / value).resolve()


def validate_source_item(item: Any, index: int) -> None:
    if not isinstance(item, dict):
        raise ValueError(f"样本 {index} 不是 JSON 对象")
    for field in ("instruction", "input", "output", "history"):
        if field not in item:
            raise ValueError(f"样本 {index} 缺少字段 {field}")
    if not str(item["instruction"]).strip() or not str(item["output"]).strip():
        raise ValueError(f"样本 {index} 的 instruction/output 不能为空")
    if not isinstance(item["history"], list):
        raise ValueError(f"样本 {index} 的 history 必须是数组")


def to_messages(item: dict[str, Any], source_index: int) -> dict[str, Any]:
    messages: list[dict[str, str]] = []
    for turn_index, turn in enumerate(item["history"]):
        if not isinstance(turn, list) or len(turn) != 2:
            raise ValueError(f"样本 {source_index} 的历史轮次 {turn_index} 格式错误")
        messages.extend((
            {"role": "user", "content": str(turn[0]).strip()},
            {"role": "assistant", "content": str(turn[1]).strip()},
        ))
    current = str(item["instruction"]).strip()
    extra_input = str(item["input"]).strip()
    if extra_input:
        current = f"{current}\n\n补充材料：\n{extra_input}"
    messages.extend((
        {"role": "user", "content": current},
        {"role": "assistant", "content": str(item["output"]).strip()},
    ))
    return {"id": f"disc-consulting-{source_index:04d}", "messages": messages}


def write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="\n") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")


def main() -> None:
    parser = argparse.ArgumentParser(description="准备 DISC 金融咨询离线微调数据")
    parser.add_argument("--config", default="config.json")
    args = parser.parse_args()
    config_path = Path(args.config).resolve()
    config = load_config(config_path)

    ratios = [float(config[key]) for key in ("train_ratio", "validation_ratio", "test_ratio")]
    if abs(sum(ratios) - 1.0) > 1e-9 or any(value <= 0 for value in ratios):
        raise ValueError("train/validation/test 比例必须均大于0且合计为1")

    source = json.loads(resolve_from_config(config_path, config["source_file"]).read_text(encoding="utf-8"))
    if not isinstance(source, list):
        raise ValueError("源数据必须是 JSON 数组")
    rows = []
    for index, item in enumerate(source):
        validate_source_item(item, index)
        rows.append(to_messages(item, index))

    random.Random(int(config["seed"])).shuffle(rows)
    train_end = int(len(rows) * ratios[0])
    validation_end = train_end + int(len(rows) * ratios[1])
    splits = {
        "train": rows[:train_end],
        "validation": rows[train_end:validation_end],
        "test": rows[validation_end:],
    }
    output_dir = resolve_from_config(config_path, config["data_dir"])
    for name, split_rows in splits.items():
        write_jsonl(output_dir / f"{name}.jsonl", split_rows)
    manifest = {
        "source": str(resolve_from_config(config_path, config["source_file"])),
        "seed": int(config["seed"]),
        "total": len(rows),
        "splits": {name: len(values) for name, values in splits.items()},
    }
    (output_dir / "manifest.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    print(json.dumps(manifest, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
