# MySQL 模型

Go 服务启动时使用 `CREATE TABLE IF NOT EXISTS` 初始化以下表。

## users

| 字段 | 说明 |
|---|---|
| `id` | 用户全局唯一 ID |
| `username` | 唯一登录账号 |
| `password_hash` | bcrypt 密码哈希，不保存明文 |
| `status` | 用户状态 |
| `created_at` | 注册时间 |
| `updated_at` | 更新时间 |

## conversation_sessions

| 字段 | 说明 |
|---|---|
| `id` | 会话 ID，与 `user_id` 组成主键 |
| `user_id` | 会话所属用户 |
| `status` | 会话状态 |
| `created_at` | 首次创建时间 |
| `last_active_at` | 最后活跃时间 |

## model_usages

| 字段 | 说明 |
|---|---|
| `id` | 用量记录 ID |
| `user_id` | 发起调用的用户 |
| `session_id` | 所属会话 |
| `request_id` | 跨服务请求追踪 ID |
| `provider` | 模型供应商 |
| `model` | 模型名称 |
| `input_tokens` | 输入 Token |
| `cached_tokens` | 缓存命中 Token |
| `output_tokens` | 输出 Token |
| `total_tokens` | 总 Token |
| `total_cost` | 预留费用字段，当前为 0 |
| `currency` | 费用币种，当前为 CNY |
| `latency_ms` | 模型调用耗时 |
| `status` | 调用状态 |
| `created_at` | 记录时间 |

## action_executions

| 字段 | 说明 |
|---|---|
| `id` | 动作执行记录ID |
| `user_id` | 发起动作的用户 |
| `session_id` | 所属会话 |
| `request_id` | 跨服务请求追踪ID |
| `intent` | Python识别的工作流意图 |
| `action` | Python建议、Go执行的下一动作 |
| `active_slot` | 本轮正在收集的槽位 |
| `status` | 执行状态，可选值为 success、not_configured |
| `result_message` | 不包含敏感信息的执行结果摘要 |
| `created_at` | 执行时间 |

验证：

```bash
docker exec -it mysql mysql -uagent_user -pagent-password agent -e 'SHOW TABLES;'
docker exec -it mysql mysql -uagent_user -pagent-password agent -e 'DESCRIBE model_usages;'
docker exec -it mysql mysql -uagent_user -pagent-password agent -e 'DESCRIBE action_executions;'
```
