# AIReview

AIReview 是一个 Web 化的 AI Pull Request 代码评审系统。Reviewer 在前端创建 GitHub PR 评审任务后，后端会自动获取 PR 元信息和代码变更，构建评审上下文，执行规则风险扫描，并通过 CloudWeGo Eino 编排 OpenAI-compatible 模型（例如 DeepSeek）生成评审报告。分析结果、上下文片段、风险项、事件和模型调用指标会保存到 MySQL，供 Web 工作台查看和管理。

本项目围绕提升 PR Review 效率与质量设计，核心能力包括：

- PR 变更总结
- 风险代码识别
- Review 建议生成
- 上下文构建与压缩
- 基于置信度、规则校验和行号校验的误报控制
- Reviewer 会话管理
- Web 化的历史记录、Findings、上下文、事件和 Markdown 报告查看

## 当前状态

已实现：

- Vue 3 Review 会话管理前端
- 基于 Gin 的 Go HTTP 后端
- 基于 MySQL/GORM 的会话、Finding、上下文、事件和 LLM 指标存储
- GitHub REST API 拉取 PR 元信息、文件变更和 commits
- diff 解析和多语言文件分类
- 规则风险扫描
- CloudWeGo Eino Review Agent 工作流
- OpenAI-compatible / DeepSeek 模型调用
- Review 进度事件
- Finding 反馈标记
- Markdown 报告导出


## 系统架构

```text
Vue 3 Web UI
  -> Gin HTTP API
    -> MySQL 会话存储
    -> Review 后台任务队列
      -> GitHub Client
      -> Diff Parser / File Classifier
      -> Rule Analyzer
      -> CloudWeGo Eino Review Agent
        -> 上下文构建
        -> 上下文检索
        -> 上下文压缩
        -> Prompt 构建
        -> DeepSeek/OpenAI-compatible 模型调用
        -> JSON 解析
        -> Finding 校验
        -> 规则结果与 AI 结果合并
      -> Markdown 报告
```

## 目录结构

```text
cmd/aireview/                 后端服务启动入口
configs/                      TOML 配置示例
internal/agent/               Eino Review Agent 和工作流节点
internal/app/                 Review 编排服务
internal/config/              TOML 与环境变量配置加载
internal/diff/                Patch 解析和文件分类
internal/github/              GitHub REST API Client
internal/jobs/                内存 Review 任务队列和 Worker
internal/report/              Markdown 报告渲染
internal/review/              核心 Review 模型、规则分析、合并/过滤逻辑
internal/reviewcontext/       上下文 chunk 构建、检索、压缩
internal/server/              Gin router、中间件、API handlers
internal/session/             会话模型和存储接口
internal/storage/             MySQL/GORM 持久化和迁移
web/                          Vue 3 前端
```

## 环境要求

- Go 1.25+
- Node.js 18+
- MySQL 8+
- 可访问目标 GitHub PR
- OpenAI-compatible LLM API Key，例如 DeepSeek

## 配置

默认配置文件：

```text
configs/aireview.toml
```

示例：

```toml
language = "zh-CN"
review_focus = ["correctness", "security", "test risk"]

[agent]
max_context_tokens = 12000
max_file_patch_bytes = 20000
max_files = 50
timeout_seconds = 180

[llm]
base_url = "https://api.deepseek.com"
api_key = ""
model = "deepseek-chat"

[mysql]
dsn = ""

[publish]
enabled = false
```

推荐通过环境变量提供敏感配置：

```powershell
$env:MYSQL_DSN="root:123456@tcp(127.0.0.1:3306)/aireview?parseTime=true&charset=utf8mb4&loc=Local"
$env:LLM_API_KEY="your-llm-api-key"
$env:LLM_BASE_URL="https://api.deepseek.com"
$env:LLM_MODEL="deepseek-chat"
$env:GITHUB_TOKEN="your-github-token"
```


## 快速启动

### 1. 启动 MySQL

使用 Docker：

```bash
docker run --name aireview-mysql ^
  -e MYSQL_ROOT_PASSWORD=123456 ^
  -e MYSQL_DATABASE=aireview ^
  -p 3306:3306 ^
  -d mysql:8.4
```

PowerShell 环境变量：

```powershell
$env:MYSQL_DSN="root:123456@tcp(127.0.0.1:3306)/aireview?parseTime=true&charset=utf8mb4&loc=Local"
$env:LLM_API_KEY="your-llm-api-key"
$env:LLM_MODEL="deepseek-chat"
```

### 2. 启动后端

```bash
go run ./cmd/aireview --port 8080 --config configs/aireview.toml
```

### 3. 启动前端

```bash
cd web
npm install
npm run dev
```

浏览器打开：

```text
http://localhost:5173
```


## API 概览

创建 Review 会话：

```http
POST /api/v1/reviews
```

请求示例：

```json
{
  "pr_url": "https://github.com/org/repo/pull/123",
  "reviewer_id": "alice",
  "config": {
    "model": "deepseek-chat",
    "review_focus": ["correctness", "security"],
    "max_files": 50
  }
}
```

主要接口：

```text
GET    /health
GET    /api/v1/reviews
POST   /api/v1/reviews
GET    /api/v1/reviews/:id
GET    /api/v1/reviews/:id/contexts
GET    /api/v1/reviews/:id/events
GET    /api/v1/reviews/:id/report.md
PATCH  /api/v1/findings/:id
```

Review 状态：

```text
created
fetching_pr
building_context
analyzing
completed
failed
cancelled
```

## Review Agent

默认 Review Agent 基于 CloudWeGo Eino 实现。工作流是确定性流程，而不是完全交给模型自由调用工具：

```text
ContextBuildNode
  -> ContextRetrieveNode
  -> ContextCompressNode
  -> PromptBuildNode
  -> ChatModelNode
  -> JSONParseNode
  -> FindingValidateNode
  -> MergeNode
```

这样设计是为了控制风险：

- GitHub 数据获取由本地 GitHub Client 负责。
- Patch 解析和文件分类由本地确定性代码负责。
- Rule findings 在模型调用前生成。
- 模型输出必须解析为 JSON。
- 返回的 findings 会根据 changed files 和 patch 行号进行校验。
- 规则 findings 和 AI findings 会合并、去重、排序。

## 准确性与风险控制

当前机制：

- 支持 Go、TypeScript、JavaScript、Python、Java 的文件语言识别
- 支持 dependency、config、test、source、generated 等文件类型识别
- 规则扫描覆盖大 PR、配置变更、依赖变更、测试缺失、敏感关键词、异常处理风险
- Agent 有上下文 token 预算和单文件 patch 字节限制
- Finding 包含 severity 和 confidence
- Finding 校验用于降低模型编造文件或行号的风险
- Reviewer 可以对 Finding 进行反馈标记

