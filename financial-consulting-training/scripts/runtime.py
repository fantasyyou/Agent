from __future__ import annotations

import hashlib
import json
import platform
from pathlib import Path
from typing import Any


def load_config(config_path: Path) -> dict[str, Any]:
    return json.loads(config_path.read_text(encoding="utf-8"))


def resolve_path(config_path: Path, value: str) -> Path:
    return (config_path.parent / value).resolve()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def require_training_data(config_path: Path, config: dict[str, Any]) -> tuple[Path, dict[str, Any]]:
    data_dir = resolve_path(config_path, config["data_dir"])
    manifest_file = data_dir / "manifest.json"
    required = [manifest_file, data_dir / "train.jsonl", data_dir / "validation.jsonl", data_dir / "test.jsonl"]
    missing = [str(path) for path in required if not path.exists()]
    if missing:
        raise RuntimeError("缺少训练数据，请先运行 prepare_data.py：" + ", ".join(missing))
    return data_dir, json.loads(manifest_file.read_text(encoding="utf-8"))


def collect_environment(torch_module) -> dict[str, Any]:
    cuda = bool(torch_module.cuda.is_available())
    environment: dict[str, Any] = {
        "platform": platform.platform(),
        "python": platform.python_version(),
        "torch": torch_module.__version__,
        "torch_cuda": torch_module.version.cuda,
        "cuda_available": cuda,
    }
    if cuda:
        properties = torch_module.cuda.get_device_properties(0)
        environment["gpu"] = {
            "name": properties.name,
            "memory_bytes": properties.total_memory,
            "memory_gib": round(properties.total_memory / 1024**3, 2),
            "compute_capability": f"{properties.major}.{properties.minor}",
        }
    return environment
