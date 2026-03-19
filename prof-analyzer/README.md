# PROF Analyzer

智能性能分析工具 - 导入 PROF 文件，AI 分析性能瓶颈，输出可视化报告

## 功能特性

- 📁 支持上传多个 PROF 文件 (pprof, json, log 等格式)
- 🤖 AI 智能分析，识别性能问题
- 🔗 清晰的调用链路展示
- ⚠️ 问题根因分析
- 💡 解决方案建议
- 📊 可视化图表 (饼图、柱状图、调用树)
- 📄 PDF 报告导出

## 技术栈

- **后端**: Go + Gin
- **前端**: Vue 3 + Vite + TypeScript + TailwindCSS
- **图表**: ECharts
- **AI**: OpenAI API (GPT-4o)

## 快速开始

### 1. 配置 API Key

编辑 `.env` 文件，设置你的 AI API Key:

```env
AI_API_KEY=your_api_key_here
AI_MODEL=gpt-4o
```

### 2. 启动后端

```bash
cd backend
go mod tidy
go run main.go
```

后端服务将在 `http://localhost:8080` 启动。

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
# 构建前端
cd frontend && npm run build && cd ..

# 启动后端 (会服务 frontend/dist)
cd backend && go build -o prof-analyzer && ./prof-analyzer
```

## API 接口

### POST /api/analyze

上传 PROF 文件进行分析

**Form Data:**
- `files`: 文件 (多个)
- `source_path`: 本地源码路径 (可选)
- `model`: AI 模型名称 (可选)

**响应:**
```json
{
  "success": true,
  "data": {
    "summary": "分析摘要",
    "chain": [...],
    "root_cause": "问题根因",
    "solutions": [...],
    "metrics": {...},
    "charts": [...],
    "hotspots": [...],
    "call_tree": [...]
  }
}
```

### GET /api/health

健康检查
