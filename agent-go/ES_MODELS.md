# Elasticsearch 模型

ES 现在只保存需要全文检索的会话记忆。用户、会话元数据和模型用量已迁移到 MySQL。

## agent-conversation-memory-v2

对应 Go 模型：`model.ConversationMemory`。

| 字段 | ES 类型 | 说明 |
|---|---|---|
| `user_id` | keyword | 记忆所属用户，也是数据隔离条件 |
| `session_id` | keyword | 所属会话 |
| `user_question` | text | 用户原始问题 |
| `assistant_answer` | text | 客服回答 |
| `created_at` | date | 问答完成时间 |

```bash
curl -u elastic:elastic 'http://localhost:9200/_cat/indices/agent-*?v'
curl -u elastic:elastic 'http://localhost:9200/agent-conversation-memory-v2/_mapping?pretty'
```

旧的 `agent-users-v1`、`agent-model-usage-v1` 和 `agent-conversation-memory-v1` 不再由新代码读取，也不会自动删除。
