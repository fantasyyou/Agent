# RAG 实战练手路径（求职导向）

> 面向求职的 RAG 工程师实战项目路线，每个项目都能成为简历亮点。

## 练手总览：5 个项目阶梯

```
项目1 → 项目2 → 项目3 → 项目4 → 项目5
基础    进阶    企业级   前沿    全栈
PDF问答  知识库  多源RAG  Agent  产品化
```

---

## 项目1：PDF 智能问答系统（入门，1-2周）

### 目标
做一个能上传 PDF 并问答的 Web 应用

### 技术栈
- 后端：Python + LangChain + FastAPI
- 向量库：Chroma（本地轻量）
- 前端：Streamlit 或 Gradio

### 核心实现
```python
# 核心pipeline（伪代码）
PDF → PyPDFLoader → RecursiveCharacterSplitter 
    → OpenAIEmbeddings → Chroma 
    → Retriever → LLM → 答案
```

### 简历亮点
- 实现文档解析、分块、向量化、检索、生成全流程
- 支持多格式文档（PDF/Word/Markdown）
- 展示检索到的原文片段，增强可解释性

### 难点突破
- 分块策略对效果的影响（chunk_size、overlap 调优）
- 不同 Embedding 模型对比（OpenAI vs BGE vs m3e）

---

## 项目2：企业知识库 RAG（进阶，2-3周）

### 目标
模拟企业内部知识库问答系统

### 技术栈
- 向量库：**Milvus** 或 **Qdrant**（企业级）
- 框架：**LlamaIndex**（比 LangChain 更适合 RAG）
- 部署：Docker Compose

### 新增技能点

#### 1. 混合检索（Hybrid Search）
```
用户查询
   ├─ 向量检索（语义相似）
   ├─ BM25检索（关键词匹配）
   └─ RRF融合排序 → 最终结果
```

#### 2. 重排序（Rerank）
- 使用 BGE Reranker 或 Cohere Rerank
- 先粗排（Top-50）再精排（Top-5）

#### 3. 查询改写
- Query Expansion：扩展同义词
- HyDE：先让 LLM 生成假设答案再检索

### 简历亮点
- 实现混合检索，召回率提升 XX%
- 引入 Rerank，答案准确率提升 XX%
- 支持多文档库管理和权限隔离

---

## 项目3：多源数据 RAG 系统（企业级，3-4周）

### 目标
整合数据库、API、文档、网页等多源数据

### 架构
```
数据源层：PDF文档 + MySQL + Notion API + 网页爬取
   ↓
统一处理层：文档加载器 + 分块 + 元数据标注
   ↓
存储层：向量库（语义） + 关系库（结构化） + 图谱（关系）
   ↓
检索层：路由器 → 选择最佳检索策略
   ↓
生成层：LLM + 引用溯源
```

### 核心技能点

#### 1. 智能路由（Query Routing）
```python
# 根据问题类型路由到不同数据源
用户问题 → LLM分类 → 
  ├─ "财务数据" → SQL查询
  ├─ "产品文档" → 向量检索
  └─ "最新资讯" → 网页搜索
```

#### 2. 父子文档检索（Parent-Child）
- 小块用于精准检索
- 大块用于提供完整上下文

#### 3. 评估体系
```python
# 使用RAGAS评估
from ragas import evaluate
from ragas.metrics import (
    faithfulness,        # 忠实度
    answer_relevancy,    # 答案相关性
    context_precision,   # 上下文精确率
    context_recall       # 上下文召回率
)
```

### 简历亮点
- 设计多源数据统一接入架构
- 实现智能路由，针对不同问题选择最优检索策略
- 建立 RAGAS 评估体系，量化系统效果
- 支持引用溯源，答案可追溯到原文

---

## 项目4：Agentic RAG 系统（前沿，3-4周）

### 目标
让 Agent 自主决定何时检索、检索什么、如何整合

### 技术栈
- 框架：**LangGraph** 或 **LlamaAgents**
- 模型：支持 Function Calling 的 LLM
- 工具：搜索引擎、计算器、代码执行器

### 核心架构
```
用户问题
   ↓
Agent决策：需要检索吗？检索几次？
   ↓
┌─ 检索1：向量库 ──────────┐
├─ 检索2：重新改写query ──┤
├─ 检索3：网页搜索 ────────┤
│                          ↓
│        结果评估：够了吗？
│           ↓ 不够          ↓ 够
│        再检索          综合答案
└──────────────────────────┘
```

### 关键实现

#### 1. Self-RAG（自反思检索）
```python
def agent_node(state):
    # 1. LLM判断是否需要检索
    if need_retrieval(query):
        docs = retrieve(query)
        # 2. 评估检索质量
        if quality_low(docs):
            # 3. 改写query重新检索
            new_query = rewrite(query)
            docs = retrieve(new_query)
    # 4. 生成答案
    return generate(query, docs)
```

#### 2. 多轮检索与推理
- Agent 可以多步检索，逐步细化
- 支持复杂问题分解（如"对比 A 和 B 的优势"）

#### 3. CRAG（纠错 RAG）
- 检索质量低 → 触发网页搜索补充
- 检索质量高 → 直接使用

### 简历亮点
- 实现 Self-RAG，Agent 自主决策检索时机
- 支持多轮迭代检索，复杂问题分解能力
- 引入 CRAG 机制，低质量检索自动纠错
- 相比 Naive RAG，准确率提升 XX%

---

## 项目5：完整产品化系统（全栈，4-6周）

### 目标
做一个可上线、可演示的完整产品

### 完整技术栈
```
前端：Next.js / React
后端：FastAPI + Celery（异步任务）
存储：PostgreSQL + Redis + Milvus
部署：Docker + Nginx + GPU服务器
监控：LangSmith / Prometheus
```

### 产品功能
1. 用户系统：注册登录、API Key 管理
2. 知识库管理：上传、分类、权限
3. 对话系统：多轮对话、历史记录
4. 管理后台：数据统计、模型配置
5. 评估看板：RAGAS 指标可视化

### 简历亮点（最重量级）
- 独立设计并实现完整 RAG 产品，支持 XX 用户
- 微服务架构，支持水平扩展
- 集成监控告警，线上问题可追踪
- A/B 测试框架，支持模型和策略迭代

---

## 求职加分项清单

### 必做
- [ ] 每个项目都有 **GitHub 仓库**（README、架构图、部署文档）
- [ ] 至少 1 个项目有 **在线 Demo**（可用 HuggingFace Spaces 免费部署）
- [ ] 用 **RAGAS** 出具评估报告，有数据支撑
- [ ] 技术博客 2-3 篇，讲清楚踩坑和优化过程

### 加分
- [ ] 对比实验：不同 Embedding 模型、不同分块策略的效果对比
- [ ] 性能优化：缓存、异步、批处理
- [ ] 成本优化：Token 使用分析、模型选择策略
- [ ] 安全考虑：Prompt 注入防护、数据脱敏

---

## 推荐时间线

```
Week 1-2:   项目1（基础RAG）         ← 快速出成果，建立信心
Week 3-5:   项目2（混合检索+Rerank）  ← 简历核心项目
Week 6-9:   项目3（多源+评估）        ← 展示工程能力
Week 10-13: 项目4（Agentic RAG）      ← 展示前沿技术
Week 14-19: 项目5（产品化）           ← 展示全栈能力
```

> **求职核心建议**：不需要做完 5 个项目。**项目2 + 项目3** 已经足够应对大多数 RAG 岗位。项目4 是差异化竞争优势。项目5 是高级岗位的敲门砖。

---

## 学习资源

### 官方文档
- LangChain 文档：https://python.langchain.com
- LlamaIndex 文档：https://docs.llamaindex.ai
- RAGAS 文档：https://docs.ragas.io

### 关键论文
- "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks"（RAG 原论文）
- "Self-RAG: Learning to Retrieve, Generate, and Critique through Self-Reflection"
- "Searching for Best Practices in Retrieval-Augmented Generation"

---

> **核心心法**：RAG 的本质是"给 LLM 外挂知识库"，难点不在生成，而在**如何精准找到相关内容**。检索质量决定了 RAG 的上限。
