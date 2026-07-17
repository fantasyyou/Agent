# 简单金融客服与 Elasticsearch 记忆

当前链路：

```text
用户 -> Go 客服 -> ES 召回同一 session 的记忆
             -> gRPC -> Python/DeepSeek
             -> ES 写入本轮问题和回答
```

ES 保存的是客服会话记忆，不应保存完整银行卡号、密码、验证码等敏感数据。生产环境应在写入前增加脱敏、保存期限和删除机制。

## 1. 配置

编辑 `config.json`：

- `elasticsearch.addresses`：ES 地址。
- `username/password` 或 `api_key`：ES 鉴权，二选一。
- `index`：记忆索引名称。
- `python_agent.address`：Python gRPC 地址。
- `memory.recall_limit`：每次最多召回多少轮记忆。

## 2. 启动 Python gRPC 服务

PowerShell：

```powershell
cd C:\Work\Agent\agent-python
$env:DEEPSEEK_API_KEY="你的 Key"
python -m pip install -r requirements.txt
python customer_service.py
```

也可以使用 Docker：

```powershell
docker build -t agent-python:latest .
docker run --rm -p 8765:8765 -e DEEPSEEK_API_KEY agent-python:latest
```

## 3. 启动 Go 客服

```powershell
cd C:\Work\Agent\agent-go
go run . -config config.json -session customer-001
```

单次提问：

```powershell
go run . -config config.json -session customer-001 -question "我是稳健型投资者，购买理财产品时更关注本金波动风险，请记住这个偏好"
go run . -config config.json -session customer-001 -question "结合我的风险偏好，介绍理财产品时应该重点提醒我什么？"
```

两次必须使用相同的 `session`。更换为 `customer-002` 后不应召回前一个客户的记忆。

## 4. 验证记忆确实写入 ES

使用账号密码：

```powershell
curl.exe -u elastic:change-me "http://127.0.0.1:9200/agent-conversation-memory-v1/_count?pretty"
curl.exe -u elastic:change-me "http://127.0.0.1:9200/agent-conversation-memory-v1/_search?pretty" -H "Content-Type: application/json" -d '{"query":{"term":{"session_id":"customer-001"}},"sort":[{"created_at":"asc"}]}'
```

每成功问答一次，`_count` 应增加 1；`_search` 应看到 `session_id`、`user_question`、`assistant_answer` 和 `created_at`。随后用同一 session 追问风险偏好，回答应能利用第一轮内容；换 session 后不应知道该偏好。

## 当前记忆能力边界

当前是短期会话记忆：优先召回文本相关的历史，同时保留最近历史作为上下文。它使用 ES 的 BM25，不生成向量。后续增加 embedding 服务和 `dense_vector` 字段后，可升级为 BM25 + kNN + RRF 的语义记忆，但不需要更换 ES。
