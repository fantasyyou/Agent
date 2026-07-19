# 跨服务对话契约

`dialogue-turn.schema.json` 是 Go 与 Python 对话状态和理解结果的共享契约。

- Go 负责生成、合并和持久化 `dialogueState`；
- Python 只返回 `dialogueDecision` 和 `slot_updates`；
- `slotDefinition` 与 `extractionResult` 约束双方共同使用的结构化概念；
- gRPC 当前使用 `google.protobuf.Struct` 传输，字段含义和允许值以该 JSON Schema 为准。

修改跨服务字段时，应同步修改 Schema、Go模型、Python模型以及双方序列化测试。
