# Python AI 推理服务

Python 服务是无状态的内部 gRPC 服务，不直接连接 MySQL 或 Elasticsearch。

```text
controller/grpc_controller.py
        -> service/customer_service.py
             -> service/requirement_analysis_service.py
             -> service/question_policy.py
             -> service/prompt_service.py
             -> client/deepseek_client.py
```

- `model`：Go 与 Python 层之间使用的类型化请求、记忆、回答和 Token Usage。
- `controller`：接收并校验 gRPC 请求。
- `service`：Prompt 和推理流程。
- `client`：DeepSeek HTTP 协议适配。

Go 传递当前问题、Redis短期工作流状态和已完成权限过滤的 Top-N 记忆。Python返回回答、新状态或清理标记以及供应商真实Usage，由Go负责持久化。

多轮需求拆解、动态追问策略和状态边界见 [REQUIREMENT_ANALYSIS.md](REQUIREMENT_ANALYSIS.md)。
