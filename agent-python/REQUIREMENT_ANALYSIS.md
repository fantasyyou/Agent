# 在线需求拆解与动态追问

该模块已经接入 `CustomerService` 主链路，不需要本地模型微调。DeepSeek负责语义理解，Python代码负责约束和下一问策略，Go负责Redis状态持久化。

## 责任边界

```text
DeepSeekRequirementExtractor
    └── 识别 intent、confidence 和用户本轮明确表达的 slots

RequirementAnalysisService
    ├── 意图置信度阈值
    ├── 参数白名单、类型、范围和枚举校验
    ├── 与上一轮 slots 合并
    ├── 必填参数检查
    └── 返回追问或工作流路由

QuestionPolicy
    └── 根据基础优先级、当前措辞和上一问动态选择缺失参数
```

模型不决定是否执行交易，也不能绕过必填参数和合规规则。

## 当前工作流

- `product_recommendation`：必填风险偏好、投资金额、投资期限；流动性要求可选。
- `fee_query`：必填业务类型。
- `human_service`：固定返回转人工说明，演示环境不创建真实工单。
- `financial_consultation`：普通金融知识问答，继续走金融客服Prompt。
- 低置信度或未知意图：提示用户选择需求方向。

## 状态流转

```text
Go从Redis读取状态
→ gRPC传给Python
→ Python提取本轮参数并与状态合并
→ 缺参：返回下一问和新状态
→ 完整：返回工作流回答并标记清除状态
→ Go写回或删除Redis Key
```

Redis Key格式：

```text
agent:dialogue:{user_id}:{session_id}
```

默认TTL为30分钟。该状态是短期任务进度，不属于ES长期记忆。

## 示例

```text
用户：我想买稳健型产品
客服：这笔资金预计多长时间内不会使用？
用户：半年
客服：您计划投入的大致金额是多少？
用户：十万元
客服：总结已确认需求，并说明需查询产品库和完成适当性校验
```

下一问不是由大模型自由发挥，而是 `QuestionPolicy` 在当前缺失参数中评分选择；表达文本来自工作流配置。

## 测试

```bash
cd agent-python
python -m unittest discover -s tests -v
```

单元测试使用假提取器，不调用DeepSeek，不消耗Token，并覆盖三轮产品需求收集。
