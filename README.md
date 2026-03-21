# prof-ui

这个仓库当前主要包含一个项目：[prof-analyzer](./prof-analyzer)。

## 项目说明

`prof-analyzer` 是一个性能分析工具。上传 PROF 文件后，后端调用 `go tool pprof` 提取文本，再由 AI 生成流式 Markdown 性能分析报告，并支持：

- pprof 火焰图生成
- PDF 报告导出
- 可选读取本地源码辅助分析

## 目录结构

- `prof-analyzer/`: 主项目目录
- `prof-analyzer/backend/`: Go + Gin 后端
- `prof-analyzer/frontend/`: Vue 3 + Vite 前端
- `prof-analyzer/SPEC.md`: 项目规格说明

## 快速入口

详细运行方式、环境变量和接口说明见：

- [prof-analyzer/README.md](./prof-analyzer/README.md)

如果你是第一次进仓库，建议先看内层 README 的这几部分：

1. 环境要求
2. 快速开始
3. API 接口
4. 使用说明
