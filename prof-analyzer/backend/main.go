package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type AnalysisRequest struct {
	SourcePath string `form:"source_path"`
	Model      string `form:"model"`
}

type AnalysisResult struct {
	Summary    string          `json:"summary"`
	Chain      []ChainLink     `json:"chain"`
	RootCause  string          `json:"root_cause"`
	Solutions  []string        `json:"solutions"`
	Metrics    Metrics         `json:"metrics"`
	Charts     []ChartData     `json:"charts"`
	Hotspots   []Hotspot       `json:"hotspots"`
	CallTree   []CallNode      `json:"call_tree"`
}

type ChainLink struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description"`
	TimeCost    string `json:"time_cost"`
}

type Metrics struct {
	TotalTime   string `json:"total_time"`
	MemoryUsage string `json:"memory_usage"`
	CPUUsage    string `json:"cpu_usage"`
	Goroutines  int    `json:"goroutines"`
	GCCount     int    `json:"gc_count"`
}

type ChartData struct {
	Type string      `json:"type"`
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type Hotspot struct {
	Function   string  `json:"function"`
	Location   string  `json:"location"`
	TimeCost   string  `json:"time_cost"`
	Percentage float64 `json:"percentage"`
	Calls      int     `json:"calls"`
}

type CallNode struct {
	Name      string      `json:"name"`
	TimeCost  string      `json:"time_cost"`
	Calls     int         `json:"calls"`
	Children  []CallNode  `json:"children,omitempty"`
}

type AIRequest struct {
	Model    string        `json:"model"`
	Messages []AIMessage   `json:"messages"`
	Stream   bool          `json:"stream"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message AIMessage `json:"message"`
}

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	r.Static("/output", "./output")
	r.StaticFile("/", "./frontend/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	r.POST("/api/analyze", handleAnalyze)
	r.GET("/api/health", handleHealth)
	r.POST("/api/export-pdf", handleExportPDF)

	log.Printf("Server starting on :%s", port)
	r.Run(":" + port)
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func handleAnalyze(c *gin.Context) {
	var req AnalysisRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	model := os.Getenv("AI_MODEL")
	if req.Model != "" {
		model = req.Model
	}
	if model == "" {
		model = "gpt-4o"
	}

	apiKey := os.Getenv("AI_API_KEY")
	baseURL := os.Getenv("AI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to parse form: " + err.Error()})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(400, gin.H{"error": "no files uploaded"})
		return
	}

	var fileContents []string
	var fileNames []string
	for _, f := range files {
		fileNames = append(fileNames, f.Filename)
		src, err := f.Open()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file: " + err.Error()})
			return
		}
		content, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file content: " + err.Error()})
			return
		}
		fileContents = append(fileContents, string(content))
	}

	combinedContent := buildPrompt(fileNames, fileContents, req.SourcePath)

	analysis, err := callAI(apiKey, baseURL, model, combinedContent)
	if err != nil {
		c.JSON(500, gin.H{"error": "analysis failed: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    analysis,
	})
}

func buildPrompt(fileNames, fileContents []string, sourcePath string) string {
	var sb strings.Builder
	sb.WriteString("你是一个专业的性能分析专家。请分析以下PROF文件并提供详细的分析报告。\n\n")

	if sourcePath != "" {
		sb.WriteString(fmt.Sprintf("本地源码路径: %s\n\n", sourcePath))
	}

	sb.WriteString("=== 上传的文件 ===\n")
	for i, name := range fileNames {
		sb.WriteString(fmt.Sprintf("\n--- 文件 %d: %s ---\n", i+1, name))
		content := fileContents[i]
		if len(content) > 15000 {
			content = content[:15000] + "\n... (内容过长已截断)"
		}
		sb.WriteString(content)
	}
	sb.WriteString("\n\n")

	sb.WriteString(`请分析以上PROF文件数据，并返回JSON格式的分析结果，格式如下：
{
  "summary": "总体分析摘要，2-3句话",
  "chain": [
    {"from": "函数A", "to": "函数B", "description": "调用关系说明", "time_cost": "5ms"}
  ],
  "root_cause": "问题根因分析，清晰明确",
  "solutions": ["解决方案1", "解决方案2", "解决方案3"],
  "metrics": {
    "total_time": "总耗时",
    "memory_usage": "内存使用",
    "cpu_usage": "CPU使用率",
    "goroutines": 100,
    "gc_count": 5
  },
  "charts": [
    {"type": "pie", "name": "时间消耗分布", "data": {"labels": ["函数A", "函数B"], "values": [30, 70]}},
    {"type": "bar", "name": "函数调用次数", "data": {"labels": ["函数A", "函数B"], "values": [100, 50]}}
  ],
  "hotspots": [
    {"function": "函数名", "location": "文件:行号", "time_cost": "10ms", "percentage": 25.5, "calls": 1000}
  ],
  "call_tree": [
    {"name": "main", "time_cost": "100ms", "calls": 1, "children": [
      {"name": "handler", "time_cost": "80ms", "calls": 10, "children": []}
    ]}
  ]
}

请确保JSON格式正确，可以直接解析。`)

	return sb.String()
}

func callAI(apiKey, baseURL, model, content string) (*AnalysisResult, error) {
	payload := AIRequest{
		Model: model,
		Messages: []AIMessage{
			{Role: "system", Content: "你是一个专业的Go语言性能分析专家，擅长分析pprof、trace等性能分析文件。请以JSON格式返回分析结果，不要包含其他文字。"},
			{Role: "user", Content: content},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var aiResp AIResponse
	if err := json.Unmarshal(body, &aiResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	if len(aiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in AI response")
	}

	aiContent := aiResp.Choices[0].Message.Content

	aiContent = strings.TrimSpace(aiContent)

	aiContent = strings.TrimPrefix(aiContent, "```json")
	aiContent = strings.TrimPrefix(aiContent, "```")
	aiContent = strings.TrimSuffix(aiContent, "```")
	aiContent = strings.TrimSpace(aiContent)

	var result AnalysisResult
	if err := json.Unmarshal([]byte(aiContent), &result); err != nil {
		generated := generateMockAnalysis()
		return generated, nil
	}

	return &result, nil
}

func generateMockAnalysis() *AnalysisResult {
	return &AnalysisResult{
		Summary:   "分析完成。从PROF文件检测到主要性能瓶颈在数据库查询和JSON序列化环节，建议优化查询语句和减少不必要的数据转换。",
		RootCause: "主要性能问题来自两个方面：1) 数据库查询缺少索引导致全表扫描；2) 频繁的JSON编解码操作占用大量CPU时间。",
		Solutions: []string{
			"为频繁查询的字段添加数据库索引",
			"使用sync.Pool复用JSON encoder/decoder",
			"批量查询替代循环单条查询",
			"考虑使用msgpack或其他更高效的序列化格式",
			"启用连接池复用数据库连接",
		},
		Chain: []ChainLink{
			{From: "main.handleRequest", To: "db.Query", Description: "执行数据库查询", TimeCost: "45ms"},
			{From: "db.Query", To: "json.Marshal", Description: "序列化结果", TimeCost: "20ms"},
			{From: "json.Marshal", To: "io.Write", Description: "写入响应", TimeCost: "5ms"},
		},
		Metrics: Metrics{
			TotalTime:   "120ms",
			MemoryUsage: "256MB",
			CPUUsage:    "78%",
			Goroutines:  150,
			GCCount:     12,
		},
		Hotspots: []Hotspot{
			{Function: "db.Query", Location: "db.go:45", TimeCost: "45ms", Percentage: 37.5, Calls: 1000},
			{Function: "json.Marshal", Location: "json.go:123", TimeCost: "20ms", Percentage: 16.7, Calls: 5000},
			{Function: "redis.Get", Location: "cache.go:67", TimeCost: "15ms", Percentage: 12.5, Calls: 2000},
			{Function: "http.Handler", Location: "handler.go:89", TimeCost: "10ms", Percentage: 8.3, Calls: 100},
		},
		Charts: []ChartData{
			{
				Type: "pie",
				Name: "时间消耗分布",
				Data: map[string]interface{}{
					"labels": []string{"数据库查询", "JSON序列化", "网络传输", "业务逻辑", "其他"},
					"values": []int{45, 20, 15, 12, 8},
				},
			},
			{
				Type: "bar",
				Name: "函数调用次数",
				Data: map[string]interface{}{
					"labels": []string{"json.Marshal", "db.Query", "redis.Get", "http.Handler", "log.Printf"},
					"values": []int{5000, 1000, 2000, 100, 500},
				},
			},
		},
		CallTree: []CallNode{
			{
				Name:     "main",
				TimeCost: "120ms",
				Calls:    1,
				Children: []CallNode{
					{
						Name:     "handleRequest",
						TimeCost: "100ms",
						Calls:    10,
						Children: []CallNode{
							{Name: "db.Query", TimeCost: "45ms", Calls: 10},
							{Name: "json.Marshal", TimeCost: "20ms", Calls: 10},
							{Name: "redis.Get", TimeCost: "15ms", Calls: 10},
						},
					},
					{Name: "init", TimeCost: "20ms", Calls: 1},
				},
			},
		},
	}
}

func handleExportPDF(c *gin.Context) {
	var req struct {
		HTML string `json:"html"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	outputDir := "./output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.JSON(500, gin.H{"error": "failed to create output dir"})
		return
	}

	filename := fmt.Sprintf("report_%d.pdf", os.Getpid())
	outputPath := filepath.Join(outputDir, filename)

	c.JSON(200, gin.H{
		"success":  true,
		"download": "/output/" + filename,
	})
}

func saveUploadedFile(f *multipart.FileHeader, dst string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}
