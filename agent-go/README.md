# Go 金融客服服务

Go 服务负责 HTTP API、认证、业务编排和所有数据访问；Python 只负责无状态 AI 推理。整体设计见根目录 [ARCHITECTURE.md](../ARCHITECTURE.md)。

## 分层

```text
controller -> service -> dao/model
                    \-> client/python gRPC
```

- `model`：带字段说明的业务模型和接口 DTO。
- `dao/mysql`：用户、会话和模型用量。
- `dao/elasticsearch`：可检索的问答记忆。
- `service`：认证和聊天用例。
- `controller`：外部 HTTP 接口。
- `client`：Python gRPC 客户端。
- `config`：配置加载与校验。

数据字段见 [MYSQL_MODELS.md](MYSQL_MODELS.md) 和 [ES_MODELS.md](ES_MODELS.md)。整套服务通过根目录的 `./start.sh` 启动。
