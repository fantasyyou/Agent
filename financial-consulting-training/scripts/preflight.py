from __future__ import annotations

import argparse
import importlib.metadata
import json
from pathlib import Path

import torch

from runtime import collect_environment, load_config, require_training_data, resolve_path, sha256_file


def main() -> None:
    parser = argparse.ArgumentParser(description="检查Windows离线QLoRA训练环境")
    parser.add_argument("--config", default="config.json")
    args = parser.parse_args()
    config_path = Path(args.config).resolve()
    config = load_config(config_path)
    data_dir, manifest = require_training_data(config_path, config)
    environment = collect_environment(torch)
    environment["packages"] = {
        name: importlib.metadata.version(name)
        for name in ("transformers", "datasets", "accelerate", "peft", "bitsandbytes", "safetensors")
    }
    environment["data"] = {
        "manifest": manifest,
        "train_sha256": sha256_file(data_dir / "train.jsonl"),
        "validation_sha256": sha256_file(data_dir / "validation.jsonl"),
        "test_sha256": sha256_file(data_dir / "test.jsonl"),
    }
    if not environment["cuda_available"]:
        raise RuntimeError("未检测到CUDA。请安装CUDA版PyTorch，并确认 torch.cuda.is_available() 为 True")
    if environment["gpu"]["memory_gib"] < 6:
        raise RuntimeError("可用GPU总显存低于6GiB，不满足当前1.5B QLoRA最低建议")
    if not bool(config.get("load_in_4bit")):
        raise RuntimeError("8GB显存配置必须启用 load_in_4bit")

    output = resolve_path(config_path, config["environment_file"])
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(environment, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(environment, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
