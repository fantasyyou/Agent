# 容器日志说明

Go 和 Python 服务都将日志写入标准输出，Docker 可以直接采集。日志采用单行 JSON，便于后续接入 Elasticsearch、Loki 或其他日志平台。

## 查看日志

```bash
docker logs -f --tail 200 agent-go
docker logs -f --tail 200 agent-app
```

只查看最近十分钟：

```bash
docker logs --since 10m agent-go
docker logs --since 10m agent-app
```

安装 `jq` 后可以按事件筛选：

```bash
docker logs agent-go 2>&1 | jq 'select(.event == "chat_processing_completed")'
docker logs agent-app 2>&1 | jq 'select(.event == "llm_request_failed")'
```

## 日志级别

在 `agent-python/.env` 中配置：

```bash
GO_LOG_LEVEL=INFO
PYTHON_LOG_LEVEL=INFO
```

支持 `DEBUG`、`INFO`、`WARN/WARNING`、`ERROR`。修改后需要重新执行 `./start.sh` 创建应用容器。

## 主要事件

Go：

- `service_starting`：Go 服务启动。
- `dependency_ready`：MySQL 或 ES 已就绪。
- `http_request_completed`：HTTP 状态码和耗时。
- `user_registered`、`user_logged_in`：认证事件。
- `chat_processing_started`、`chat_processing_completed`：聊天请求和 Token 用量。
- `chat_stage_failed`：会话、记忆、Python 或计量阶段失败。

Python：

- `service_starting`、`grpc_server_started`：Python 和 gRPC 服务启动。
- `grpc_request_started`、`grpc_request_completed`：内部推理请求。
- `llm_request_started`、`llm_request_completed`：DeepSeek 调用、耗时和 Token。
- `grpc_request_failed`、`llm_request_failed`：带异常堆栈的失败事件。

日志不会输出密码、API Key、完整问题、Prompt 或模型回答正文，只记录内部 ID、字符数、耗时、状态和 Token 统计。
