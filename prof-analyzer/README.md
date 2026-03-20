# PROF Analyzer

智能性能分析工具。导入 PROF 文件后，后端调用 `go tool pprof` 提取文本，再由 AI 生成流式 Markdown 性能分析报告，并支持导出 PDF 与生成 pprof 火焰图。

## 功能特性

- 📁 支持上传多个 PROF 文件 (pprof, json, log 等格式)
- 🤖 AI 流式分析，实时返回 Markdown 报告
- 🔗 调用链路、问题根因、优化建议一体化输出
- 🧠 可选读取本地源码，辅助定位热点代码
- 🖼️ 支持生成 pprof 火焰图图片
- 📄 PDF 报告导出
- 💾 分析结果自动保存到 `output/`

## 技术栈

- **后端**: Go + Gin
- **前端**: Vue 3 + Vite + TypeScript + TailwindCSS
- **可视化**: pprof 图片 + Markdown 渲染
- **AI**: OpenAI API (GPT-4o)

## 环境要求

- Go 1.21+
- Node.js 18+ 建议
- 已正确设置 `GOROOT`
- 可用的 AI API Key

## 快速开始

### 1. 配置 API Key

编辑 `.env` 文件，设置你的 AI API Key:

```env
AI_API_KEY=your_api_key_here
AI_MODEL=gpt-4o
OUTPUT_LANGUAGE=中文
PORT=8181
GOROOT=/path/to/your/go
```

说明：

- `PORT=8181` 是为了和当前前端开发代理配置保持一致
- `OUTPUT_LANGUAGE` 用于指定分析报告语言，默认值为 `中文`
- 如果你改了后端端口，也要同步修改 [frontend/vite.config.ts](/Users/quzhihao/GolandProjects/prof-ui/prof-analyzer/frontend/vite.config.ts)
- 未设置 `GOROOT` 时，`go tool pprof` 和 pprof 图片生成功能会失败

### 2. 启动后端

```bash
cd backend
go mod tidy
go run .
```

后端服务将在 `http://localhost:8181` 启动。

说明：

- 推荐使用 `go run .`
- 如果你习惯执行 `go run main.go`，当前也可以正常工作，因为系统提示词已移动到独立子包

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端开发服务器将在 `http://localhost:5173` 启动。

### 4. 构建前端生产版本

```bash
cd frontend
npm run build
```

构建产物在 `frontend/dist`，可被后端直接服务。

## 生产部署

```bash
cd frontend
npm run build

cd ../backend
go build -o prof-analyzer
./prof-analyzer
```

生产模式下，后端会直接服务 `frontend/dist`。

## API 接口

### POST /api/analyze/stream

上传 PROF 文件并流式返回 Markdown 分析结果。

**Form Data:**
- `files`: 文件 (多个)
- `source_path`: 本地源码路径 (可选)

**SSE 事件:**
- `chunk`: 增量 Markdown 内容
- `saved`: 最终保存的结果文件路径
- `error`: 错误信息

### POST /api/pprof/image

上传单个 profile 文件，生成 pprof 火焰图 PNG。

**Form Data:**
- `file`: 单个 profile 文件

**响应字段:**
- `success`: 是否成功
- `path`: 本地保存路径
- `url`: 前端可访问地址

### POST /api/pprof/text

上传单个 profile 文件，返回 `go tool pprof` 文本分析结果。

### GET /api/health

健康检查。

## 使用说明

1. 在“上传文件”页选择一个或多个 profile 文件。
2. 如需结合源码定位，可展开并填写“本地源码路径”。
3. 点击“开始分析”，查看流式 Markdown 报告。
4. 如需火焰图，点击“生成 Pprof 图片”。
5. 在“导出PDF”页导出当前分析报告。

说明：

- PDF 导出依赖当前内存中的分析结果，先执行一次分析再导出
- “可视化”页当前展示的是 pprof 火焰图，不是结构化 ECharts 图表

## 示例文件

PROF 分析支持以下格式，以下是示例内容：

### 1. Go pprof CPU profile (text 格式)

```
--- cpu ---
Duration: 30s, Total samples: 1200

flat  flat%   sum%     cum     cum%  function
45ms  3.75%  3.75%    120ms   10%   database.Query
30ms  2.50%  6.25%    80ms    6.67% json.Marshal
25ms  2.08%  8.33%    25ms    2.08% redis.Get
20ms  1.67%  10.00%   45ms    3.75% handler.ServeHTTP
15ms  1.25%  11.25%   15ms    1.25% log.Printf
```

### 2. JSON 格式性能数据

```json
{
  "type": "pprof",
  "duration_ms": 30000,
  "samples": [
    {
      "function": "database.Query",
      "location": "db.go:45",
      "time_ms": 45,
      "count": 100
    },
    {
      "function": "json.Marshal",
      "location": "json.go:123",
      "time_ms": 30,
      "count": 500
    },
    {
      "function": "redis.Get",
      "location": "cache.go:67",
      "time_ms": 25,
      "count": 200
    }
  ],
  "metrics": {
    "goroutines": 150,
    "memory_mb": 256,
    "gc_count": 12,
    "gc_pause_ms": 8
  }
}
```

### 3. Trace 格式日志

```
[0.000s] HTTP request received: /api/users
[0.015s] Middleware: auth validation
[0.020s] Database query started
[0.065s] Database query completed (45ms)
[0.070s] JSON serialization started
[0.090s] JSON serialization completed (20ms)
[0.092s] Response sent (92ms)
```

上传任意上述格式文件，AI 将自动分析并输出包含**调用链路**、**问题根因**、**性能热点**、**解决建议**的完整 Markdown 报告。
