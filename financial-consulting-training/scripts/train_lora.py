from __future__ import annotations

import argparse
import importlib.metadata
import json
from datetime import datetime, timezone
from pathlib import Path

import torch
from datasets import load_dataset
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training
from transformers import (
    AutoModelForCausalLM,
    AutoTokenizer,
    BitsAndBytesConfig,
    DataCollatorForSeq2Seq,
    Trainer,
    TrainingArguments,
    set_seed,
)

from runtime import collect_environment, load_config, require_training_data, resolve_path, sha256_file


def main() -> None:
    parser = argparse.ArgumentParser(description="在8GB NVIDIA显卡上离线微调金融咨询QLoRA")
    parser.add_argument("--config", default="config.json")
    parser.add_argument("--smoke", action="store_true", help="只用2条数据训练1步，验证完整链路")
    parser.add_argument("--resume", action="store_true", help="从输出目录最近的checkpoint继续训练")
    args = parser.parse_args()
    config_path = Path(args.config).resolve()
    config = load_config(config_path)
    data_dir, manifest = require_training_data(config_path, config)
    if not torch.cuda.is_available():
        raise RuntimeError("QLoRA训练需要CUDA；当前 torch.cuda.is_available() 为 False")

    set_seed(int(config["seed"]))
    compute_dtype = torch.bfloat16 if torch.cuda.is_bf16_supported() else torch.float16
    quantization = BitsAndBytesConfig(
        load_in_4bit=bool(config["load_in_4bit"]),
        bnb_4bit_quant_type=str(config["bnb_4bit_quant_type"]),
        bnb_4bit_use_double_quant=bool(config["bnb_4bit_use_double_quant"]),
        bnb_4bit_compute_dtype=compute_dtype,
    )
    tokenizer = AutoTokenizer.from_pretrained(config["base_model"], use_fast=True)
    if tokenizer.pad_token_id is None:
        tokenizer.pad_token = tokenizer.eos_token
    model = AutoModelForCausalLM.from_pretrained(
        config["base_model"],
        quantization_config=quantization,
        torch_dtype=compute_dtype,
        device_map={"": 0},
    )
    model.config.use_cache = False
    model = prepare_model_for_kbit_training(
        model, use_gradient_checkpointing=bool(config["gradient_checkpointing"])
    )
    model = get_peft_model(model, LoraConfig(
        r=int(config["lora_rank"]),
        lora_alpha=int(config["lora_alpha"]),
        lora_dropout=float(config["lora_dropout"]),
        bias="none",
        task_type="CAUSAL_LM",
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj"],
    ))
    model.print_trainable_parameters()

    dataset = load_dataset("json", data_files={
        "train": str(data_dir / "train.jsonl"),
        "validation": str(data_dir / "validation.jsonl"),
    })
    if args.smoke:
        dataset["train"] = dataset["train"].select(range(min(2, len(dataset["train"]))))
        dataset["validation"] = dataset["validation"].select(range(min(1, len(dataset["validation"]))))
    max_length = int(config["max_length"])

    def tokenize(example):
        messages = example["messages"]
        full_ids = tokenizer.apply_chat_template(
            messages, tokenize=True, add_generation_prompt=False, truncation=True, max_length=max_length
        )
        prompt_ids = tokenizer.apply_chat_template(
            messages[:-1], tokenize=True, add_generation_prompt=True, truncation=True, max_length=max_length
        )
        masked_length = min(len(prompt_ids), len(full_ids))
        labels = [-100] * masked_length + full_ids[masked_length:]
        if all(label == -100 for label in labels):
            raise ValueError(f"样本 {example.get('id', 'unknown')} 截断后没有助手答案，请调整 max_length")
        return {"input_ids": full_ids, "attention_mask": [1] * len(full_ids), "labels": labels}

    tokenized = dataset.map(tokenize, remove_columns=dataset["train"].column_names)
    configured_output = resolve_path(config_path, config["output_dir"])
    output_dir = configured_output.parent / "smoke" if args.smoke else configured_output
    output_dir.mkdir(parents=True, exist_ok=True)
    arguments = TrainingArguments(
        output_dir=str(output_dir),
        learning_rate=float(config["learning_rate"]),
        num_train_epochs=float(config["num_train_epochs"]),
        max_steps=1 if args.smoke else -1,
        per_device_train_batch_size=int(config["per_device_train_batch_size"]),
        per_device_eval_batch_size=1,
        gradient_accumulation_steps=1 if args.smoke else int(config["gradient_accumulation_steps"]),
        gradient_checkpointing=bool(config["gradient_checkpointing"]),
        optim=str(config["optimizer"]),
        eval_strategy="steps" if args.smoke else "epoch",
        eval_steps=1 if args.smoke else None,
        save_strategy="steps" if args.smoke else "epoch",
        save_steps=1 if args.smoke else 500,
        save_total_limit=int(config["save_total_limit"]),
        logging_steps=1 if args.smoke else 5,
        load_best_model_at_end=True,
        metric_for_best_model="eval_loss",
        greater_is_better=False,
        report_to="none",
        bf16=compute_dtype == torch.bfloat16,
        fp16=compute_dtype == torch.float16,
        remove_unused_columns=False,
    )
    trainer = Trainer(
        model=model,
        args=arguments,
        train_dataset=tokenized["train"],
        eval_dataset=tokenized["validation"],
        data_collator=DataCollatorForSeq2Seq(tokenizer=tokenizer, padding=True, label_pad_token_id=-100),
    )
    result = trainer.train(resume_from_checkpoint=True if args.resume else None)
    trainer.save_model(str(output_dir))
    tokenizer.save_pretrained(str(output_dir))
    metadata = {
        "created_at": datetime.now(timezone.utc).isoformat(),
        "smoke": args.smoke,
        "base_model": config["base_model"],
        "config": config,
        "data_manifest": manifest,
        "data_sha256": {
            "train": sha256_file(data_dir / "train.jsonl"),
            "validation": sha256_file(data_dir / "validation.jsonl"),
            "test": sha256_file(data_dir / "test.jsonl"),
        },
        "environment": collect_environment(torch),
        "packages": {
            name: importlib.metadata.version(name)
            for name in ("transformers", "datasets", "accelerate", "peft", "bitsandbytes")
        },
        "metrics": result.metrics,
    }
    (output_dir / "training_metadata.json").write_text(
        json.dumps(metadata, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


if __name__ == "__main__":
    main()
