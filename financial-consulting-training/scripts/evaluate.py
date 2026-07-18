from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path

import torch
from peft import PeftModel
from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

from runtime import load_config, require_training_data, resolve_path


def normalized_chars(text: str) -> list[str]:
    return [char for char in text.strip().lower() if not char.isspace()]


def char_f1(reference: str, prediction: str) -> float:
    expected, actual = Counter(normalized_chars(reference)), Counter(normalized_chars(prediction))
    overlap = sum((expected & actual).values())
    if not expected or not actual or overlap == 0:
        return 0.0
    precision = overlap / sum(actual.values())
    recall = overlap / sum(expected.values())
    return 2 * precision * recall / (precision + recall)


def load_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def generate(model, tokenizer, messages: list[dict], max_new_tokens: int) -> str:
    prompt = tokenizer.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
    device = next(model.parameters()).device
    inputs = tokenizer(prompt, return_tensors="pt").to(device)
    with torch.inference_mode():
        generated = model.generate(
            **inputs,
            max_new_tokens=max_new_tokens,
            do_sample=False,
            pad_token_id=tokenizer.pad_token_id,
            eos_token_id=tokenizer.eos_token_id,
        )
    return tokenizer.decode(
        generated[0][inputs["input_ids"].shape[1]:], skip_special_tokens=True
    ).strip()


def summarize(rows: list[dict], prediction_field: str) -> dict:
    return {
        "count": len(rows),
        "exact_match": sum(row[prediction_field] == row["reference"] for row in rows) / len(rows),
        "average_char_f1": sum(char_f1(row["reference"], row[prediction_field]) for row in rows) / len(rows),
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="对比基础模型与金融咨询QLoRA")
    parser.add_argument("--config", default="config.json")
    parser.add_argument("--limit", type=int, default=0, help="只评测前N条；0表示完整测试集")
    args = parser.parse_args()
    config_path = Path(args.config).resolve()
    config = load_config(config_path)
    data_dir, _ = require_training_data(config_path, config)
    adapter_dir = resolve_path(config_path, config["output_dir"])
    report_file = resolve_path(config_path, config["report_file"])
    if not (adapter_dir / "adapter_config.json").exists():
        raise RuntimeError("没有找到正式LoRA产物，请先执行 train_lora.py（不是 --smoke）")
    if not torch.cuda.is_available():
        raise RuntimeError("4-bit评测需要CUDA")

    compute_dtype = torch.bfloat16 if torch.cuda.is_bf16_supported() else torch.float16
    tokenizer = AutoTokenizer.from_pretrained(adapter_dir, use_fast=True)
    if tokenizer.pad_token_id is None:
        tokenizer.pad_token = tokenizer.eos_token
    quantization = BitsAndBytesConfig(
        load_in_4bit=bool(config["load_in_4bit"]),
        bnb_4bit_quant_type=str(config["bnb_4bit_quant_type"]),
        bnb_4bit_use_double_quant=bool(config["bnb_4bit_use_double_quant"]),
        bnb_4bit_compute_dtype=compute_dtype,
    )
    base = AutoModelForCausalLM.from_pretrained(
        config["base_model"], quantization_config=quantization, torch_dtype=compute_dtype, device_map={"": 0}
    )
    model = PeftModel.from_pretrained(base, adapter_dir).eval()
    test_rows = load_jsonl(data_dir / "test.jsonl")
    if args.limit > 0:
        test_rows = test_rows[:args.limit]

    results = []
    for item in test_rows:
        messages = item["messages"][:-1]
        reference = item["messages"][-1]["content"]
        with model.disable_adapter():
            baseline_prediction = generate(model, tokenizer, messages, int(config["max_new_tokens"]))
        adapter_prediction = generate(model, tokenizer, messages, int(config["max_new_tokens"]))
        results.append({
            "id": item["id"],
            "reference": reference,
            "baseline_prediction": baseline_prediction,
            "adapter_prediction": adapter_prediction,
            "baseline_char_f1": char_f1(reference, baseline_prediction),
            "adapter_char_f1": char_f1(reference, adapter_prediction),
        })
        print(f"evaluated {item['id']}")

    baseline = summarize(results, "baseline_prediction")
    adapter = summarize(results, "adapter_prediction")
    report = {
        "base_model": config["base_model"],
        "adapter": str(adapter_dir),
        "baseline": baseline,
        "fine_tuned": adapter,
        "delta_average_char_f1": adapter["average_char_f1"] - baseline["average_char_f1"],
        "results": results,
        "warning": "字符F1只用于自动回归；金融事实、合规性和回答质量仍需人工审核。",
    }
    report_file.parent.mkdir(parents=True, exist_ok=True)
    report_file.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({key: value for key, value in report.items() if key != "results"}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
