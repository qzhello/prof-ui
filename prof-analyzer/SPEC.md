# PROF Analyzer - Specification

## Project Overview

**Project Name:** PROF Analyzer  
**Type:** Full-stack web application (Go backend + Vue 3 frontend)  
**Core Functionality:** Upload and analyze PROF (profiling) files using AI, visualize results with charts, export to PDF  
**Target Users:** Developers and DevOps engineers analyzing performance profiling data

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Vue 3 SPA     │────▶│   Go Backend    │────▶│   AI API        │
│   (Frontend)    │◀────│   (Gin + Go)    │◀────│   (OpenAI/etc)  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │                       ▼
        │               ┌─────────────────┐
        │               │   File System   │
        │               │  (Local Cache)  │
        └───────────────┴─────────────────┘
```

## Tech Stack

- **Frontend:** Vue 3 + Vite + TypeScript + TailwindCSS
- **Backend:** Go 1.21+ with Gin framework
- **AI Provider:** OpenAI API (configurable model via .env)
- **Charts:** ECharts
- **PDF Export:** html2pdf.js (client-side)
- **File Handling:** Multipart upload, local storage

## .env Configuration

```env
AI_API_KEY=your_api_key_here
AI_MODEL=gpt-4o
AI_BASE_URL=https://api.openai.com/v1
PORT=8080
MAX_UPLOAD_SIZE=50
```

## Functionality Specification

### 1. File Upload Module
- [x] Accept .prof, .pprof, .json, .log files
- [x] Support multiple file selection
- [x] Drag-and-drop support
- [x] File preview (show file name, size, type)
- [x] Remove selected files
- [x] Optional local source code path input

### 2. AI Analysis Module
- [x] Send file content + optional source path to AI
- [x] Parse AI response for structured analysis
- [x] Handle streaming responses for large files
- [x] Show analysis progress/status
- [x] Support retry on failure

### 3. Results Display
- [x] **链路分析 (Chain Analysis):** Show call chains, dependencies
- [x] **问题根因 (Root Cause):** Clear problem identification
- [x] **解决方式 (Solutions):** Actionable recommendations
- [x] **性能指标 (Metrics):** Key performance numbers
- [x] **图表展示:** 
  - [x] Call graph visualization
  - [x] Time consumption chart (bar/pie)
  - [x] Hotspot identification

### 4. PDF Export
- [x] Export full analysis report to PDF
- [x] Include charts as images in PDF
- [x] Professional layout with branding

## API Endpoints

### POST /api/analyze
Upload PROF files and get AI analysis.

**Request:**
- `multipart/form-data`
- `files`: File[] (multiple)
- `source_path`: string (optional)
- `model`: string (optional, overrides .env)

**Response:**
```json
{
  "success": true,
  "data": {
    "summary": "...",
    "chain": [...],
    "root_cause": "...",
    "solutions": [...],
    "metrics": {...},
    "charts": [...]
  }
}
```

### GET /api/health
Health check endpoint.

## Frontend Pages

### Main Page (App.vue)
- Header with app title
- File upload area (drag & drop)
- Source path input (optional, collapsible)
- Analyze button
- Results section with tabs:
  - 分析结果 (Analysis Results)
  - 可视化 (Visualization)
  - 导出 (Export)

### Analysis Results Tab
- Summary card
- Chain/Links section (tree view or list)
- Root Cause section (highlighted)
- Solutions section (numbered list)
- Metrics cards

### Visualization Tab
- ECharts for:
  - Flame graph / call tree
  - Time distribution pie chart
  - Hotspot bar chart

### Export Tab
- Preview button
- Download PDF button

## Acceptance Criteria

1. ✅ User can upload multiple PROF files via drag-drop or file picker
2. ✅ User can optionally provide local source code path
3. ✅ AI analyzes the files and returns structured results
4. ✅ Results display clear chain, root cause, and solutions
5. ✅ At least 3 charts visualize the data
6. ✅ PDF export works with all content and charts
7. ✅ API key and model are configurable via .env
8. ✅ Application handles errors gracefully
9. ✅ Clean, professional UI with good UX

## File Structure

```
prof-analyzer/
├── backend/
│   ├── main.go
│   ├── go.mod
│   ├── handlers/
│   │   └── analyze.go
│   ├── services/
│   │   └── ai_service.go
│   └── utils/
│       └── file_utils.go
├── frontend/
│   ├── src/
│   │   ├── main.ts
│   │   ├── App.vue
│   │   ├── components/
│   │   │   ├── FileUpload.vue
│   │   │   ├── AnalysisResult.vue
│   │   │   ├── Charts.vue
│   │   │   └── PdfExport.vue
│   │   ├── types/
│   │   │   └── index.ts
│   │   └── api/
│   │       └── index.ts
│   ├── index.html
│   ├── vite.config.ts
│   ├── package.json
│   └── tailwind.config.js
├── .env
└── SPEC.md
```
