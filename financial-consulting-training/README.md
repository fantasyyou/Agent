# 金融咨询 Windows 离线 QLoRA 工程

本目录与 `agent-python` 平级，只负责数据准备、离线训练和离线评测，不进入在线服务镜像。

## 当前目标

- 基础模型：`Qwen/Qwen2.5-1.5B-Instruct`。
- 数据：`../DISC-FinLLM/data/consulting_part.json` 的100条金融咨询样例。
- 拆分：固定随机种子42，80条训练、10条验证、10条测试。
- 方法：4-bit NF4 QLoRA，适配约8GB NVIDIA显存。
- 产物：`artifacts/financial-consulting-lora/` 下的LoRA适配器和训练元数据。

这100条数据只能验证端到端微调流程，不能证明生产级金融知识和合规能力。多轮槽位、缺参追问、业务规则和会话状态仍由 `agent-python` 服务层负责。

## 目录

```text
financial-consulting-training/
├─ config.json
├─ requirements.txt
├─ setup_windows.ps1          # 创建隔离环境并安装GPU训练依赖
├─ run_offline.ps1            # 数据、预检、冒烟、训练、评测入口
├─ scripts/
│  ├─ prepare_data.py
│  ├─ preflight.py
│  ├─ train_lora.py
│  ├─ evaluate.py
│  └─ runtime.py
├─ data/                      # 自动生成，不提交Git
└─ artifacts/                 # 模型、环境记录和报告，不提交Git
```

## 1. 安装环境

在普通PowerShell中执行：

```powershell
cd C:\Work\Agent\financial-consulting-training
Set-ExecutionPolicy -Scope Process Bypass
.\setup_windows.ps1
```

默认从PyTorch CUDA 12.8索引安装。若PyTorch官方安装选择器为你的环境提供了更新索引，可以显式传入：

```powershell
.\setup_windows.ps1 -TorchIndexUrl "https://download.pytorch.org/whl/cu128"
```

安装结束必须出现：

```text
available: True
gpu: NVIDIA GeForce RTX 5070 Laptop GPU
```

如果是 `False`，不要继续训练。

## 2. 分阶段执行

先准备并核对数据：

```powershell
.\run_offline.ps1 -Stage Prepare
```

检查CUDA、显存、依赖版本和数据哈希：

```powershell
.\run_offline.ps1 -Stage Preflight
```

先用2条数据训练1步，确认模型下载、4-bit量化、反向传播和保存全部正常：

```powershell
.\run_offline.ps1 -Stage Smoke
```

冒烟产物位于 `artifacts/smoke/`，不能用于正式评测或部署。

正式训练：

```powershell
.\run_offline.ps1 -Stage Train
```

中断后从最近checkpoint恢复：

```powershell
.\run_offline.ps1 -Stage Train -Resume
```

比较基础模型与LoRA模型：

```powershell
.\run_offline.ps1 -Stage Evaluate
```

确认单步流程稳定后，也可一次执行完整流水线：

```powershell
.\run_offline.ps1 -Stage All
```

首次运行会从Hugging Face下载数GB基础模型，需要稳定网络和足够磁盘空间。后续会使用本地缓存。

## 3. 训练产物

正式训练完成后，主要文件包括：

```text
artifacts/financial-consulting-lora/
├─ adapter_model.safetensors
├─ adapter_config.json
├─ tokenizer.json（具体文件由tokenizer决定）
└─ training_metadata.json
```

`training_metadata.json` 记录：

- 基础模型名称；
- 完整训练配置；
- 数据拆分清单和SHA-256；
- Python、PyTorch、CUDA、GPU和依赖版本；
- 训练指标和生成时间。

环境预检记录保存在 `artifacts/environment.json`，对比报告保存在 `artifacts/evaluation.json`。

## 4. 评测解释

评测脚本对10条隔离测试数据分别执行：

```text
原始Qwen生成
LoRA模型生成
字符F1与完全匹配对比
逐条保存两份回答
```

生成式金融咨询不应只看完全匹配或字符F1。发布前必须人工检查：

- 是否存在金融事实错误；
- 是否承诺收益或保本；
- 是否引用了不存在的政策、产品或实时数据；
- 是否正确使用多轮上下文；
- 回答是否比基础模型实质改善。

## 5. 显存不足处理

当前默认已经启用4-bit NF4、双重量化、梯度检查点、批大小1和512长度。若仍发生CUDA OOM：

1. 关闭浏览器GPU加速和占用显存的程序；
2. 确保电脑接通电源并保持散热；
3. 将 `max_length` 从512降到384或256；
4. 重新执行 `Smoke`，成功后再正式训练。

不要通过删除验证集或把测试集加入训练来解决问题。

## 6. 生产接入边界

训练完成不代表直接修改 `agent-python`。正确顺序是：

```text
QLoRA产物
→ 离线基线对比
→ 人工审核
→ 独立模型推理服务
→ agent-python通过HTTP/gRPC调用
```

PyTorch、Transformers、基础模型和LoRA权重不应直接打包进当前轻量 `agent-python` 容器。
