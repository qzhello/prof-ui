# PROF Analyzer Browser Extension

Chrome 浏览器插件 - AI 驱动的 pprof/PROF 文件分析工具

## 功能

- 📁 支持上传 .prof, .pprof, .json, .log 文件
- 🤖 调用后端 AI API 进行智能分析
- 📊 结果展示：链路分析、根因、解决方案
- 🔧 可配置后端 API 地址

## 安装

1. 打开 Chrome，访问 `chrome://extensions/`
2. 开启右上角的「开发者模式」
3. 点击「加载已解压的扩展程序」
4. 选择 `browser-extension` 文件夹

## 配置

1. 点击插件图标
2. 在 "Backend API URL" 中输入你的后端地址
   - 例如：`http://localhost:8080`
3. 点击保存

## 使用

1. 拖拽或选择 pprof 文件
2. 点击 "Analyze with AI" 按钮
3. 查看分析结果

## 后端要求

后端需要实现 `/api/analyze/stream` 接口：
- Method: POST
- Content-Type: multipart/form-data
- 参数: `files` (文件数组)
- 返回: SSE 流式响应

参考 [prof-analyzer backend](../prof-analyzer/backend/)

## 文件结构

```
browser-extension/
├── manifest.json      # 插件清单
├── popup.html         # 弹窗界面
├── popup.js           # 弹窗逻辑
├── icons/             # 图标
└── README.md
```
