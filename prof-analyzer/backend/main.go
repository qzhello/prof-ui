package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type AnalysisRequest struct {
	SourcePath string `form:"source_path"`
	Model      string `form:"model"`
}

type AnalysisResult struct {
	Summary    string      `json:"summary"`
	Chain      []ChainLink `json:"chain"`
	RootCause  string      `json:"root_cause"`
	Solutions  []string    `json:"solutions"`
	Metrics    Metrics     `json:"metrics"`
	Charts     []ChartData `json:"charts"`
	Hotspots   []Hotspot   `json:"hotspots"`
	CallTree   []CallNode  `json:"call_tree"`
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
	Name     string     `json:"name"`
	TimeCost string     `json:"time_cost"`
	Calls    int        `json:"calls"`
	Children []CallNode `json:"children,omitempty"`
}

type AIRequest struct {
	Model    string      `json:"model"`
	Messages []AIMessage `json:"messages"`
	Stream   bool        `json:"stream"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message AIMessage `json:"message"`
}

// SSE event wrapper
type SSEResult struct {
	Event string `json:"event,omitempty"`
	Data  string `json:"data,omitempty"`
}

var httpClient = &http.Client{}

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Ensure output directory exists
	os.MkdirAll("./output", 0755)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	r.Static("/output", "./output")
	r.StaticFile("/", "./frontend/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	r.POST("/api/analyze", handleAnalyze)
	r.POST("/api/analyze/stream", handleAnalyzeStream)
	r.POST("/api/save-result", handleSaveResult)
	r.POST("/api/pprof/image", handlePprofImage)
	r.GET("/api/health", handleHealth)

	log.Printf("Server starting on :%s", port)
	r.Run(":" + port)
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

// outputPath generates a timestamped file path under ./output
func outputPath(prefix, ext string) string {
	ts := time.Now().Format("20060102_150405")
	return filepath.Join("output", fmt.Sprintf("%s_%s.%s", prefix, ts, ext))
}

// saveJSON saves data to a file and returns the file path
func saveJSON(v interface{}, prefix, ext string) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	path := outputPath(prefix, ext)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// handlePprofImage runs `go tool pprof -png/-svg` and saves the output to ./output/
func handlePprofImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "no file uploaded"})
		return
	}

	if file.Size > 100*1024*1024 {
		c.JSON(400, gin.H{"error": "file too large, max 100MB"})
		return
	}

	tmpDir := os.TempDir()
	profPath := filepath.Join(tmpDir, "pprof_"+filepath.Base(file.Filename))
	pngPath := outputPath("pprof", "png")
	svgPath := outputPath("pprof", "svg")

	f, err := file.Open()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to open file"})
		return
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to read file"})
		return
	}
	if err := os.WriteFile(profPath, data, 0644); err != nil {
		c.JSON(500, gin.H{"error": "failed to save temp file"})
		return
	}
	defer os.Remove(profPath)

	// Try PNG first, then SVG
	var out []byte
	savedPath := ""

	for _, format := range []string{"-png", "-svg"} {
		cmd := exec.Command("go", "tool", "pprof", format, profPath)
		cmd.Dir = tmpDir
		out, err = cmd.Output()
		if err == nil && len(out) > 0 {
			if format == "-png" {
				savedPath = pngPath
			} else {
				savedPath = svgPath
			}
			break
		}
	}

	if savedPath == "" {
		errMsg := strings.TrimSpace(string(out))
		if errMsg == "" {
			errMsg = err.Error()
		}
		c.JSON(400, gin.H{"error": "go tool pprof failed: " + errMsg})
		return
	}

	if err := os.WriteFile(savedPath, out, 0644); err != nil {
		c.JSON(500, gin.H{"error": "failed to save pprof image: " + err.Error()})
		return
	}

	// Return download URL relative to server root
	downloadURL := "/" + savedPath
	c.JSON(200, gin.H{
		"success": true,
		"path":    savedPath,
		"url":     downloadURL,
		"message": "图片已保存，可直接访问 " + downloadURL + " 下载",
	})
}

func handleAnalyze(c *gin.Context) {
	model := os.Getenv("AI_MODEL")
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
			c.JSON(500, gin.H{"error": "failed to read file"})
			return
		}
		content, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file content"})
			return
		}
		fileContents = append(fileContents, string(content))
	}

	sourcePath := c.PostForm("source_path")
	combinedContent := buildPrompt(fileNames, fileContents, sourcePath)

	analysis, err := callAI(apiKey, baseURL, model, combinedContent)
	if err != nil {
		c.JSON(500, gin.H{"error": "analysis failed: " + err.Error()})
		return
	}

	// Save result to ./output/
	resultPath, err := saveJSON(analysis, "analysis", "json")
	if err != nil {
		log.Printf("warning: failed to save analysis result: %v", err)
	}

	c.JSON(200, gin.H{
		"success":    true,
		"data":       analysis,
		"result_url": "/" + resultPath,
		"result_path": resultPath,
	})
}

// handleSaveResult saves a complete analysis result JSON to ./output/
func handleSaveResult(c *gin.Context) {
	var result AnalysisResult
	if err := c.ShouldBindJSON(&result); err != nil {
		c.JSON(400, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	resultPath, err := saveJSON(result, "analysis_stream", "json")
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to save result: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"success":     true,
		"result_path": resultPath,
		"result_url":  "/" + resultPath,
	})
}

func handleAnalyzeStream(c *gin.Context) {
	model := os.Getenv("AI_MODEL")
	baseURL := os.Getenv("AI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	apiKey := os.Getenv("AI_API_KEY")

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
			c.JSON(500, gin.H{"error": "failed to read file"})
			return
		}
		content, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file content"})
			return
		}
		fileContents = append(fileContents, string(content))
	}

	sourcePath := c.PostForm("source_path")
	prompt := buildPrompt(fileNames, fileContents, sourcePath)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	payload := AIRequest{
		Model: model,
		Messages: []AIMessage{
			{Role: "system", Content: "你是一个专业的Go语言性能分析专家，擅长分析pprof、trace等性能分析文件。请以JSON格式返回分析结果。"},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		c.SSEvent("error", "AI request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		c.SSEvent("error", fmt.Sprintf("AI API error %d: %s", resp.StatusCode, string(body)))
		return
	}

	reader := resp.Body
	buf := make([]byte, 0, 1024)
	lineBuf := []byte{}

	for {
		b := make([]byte, 1)
		n, err := reader.Read(b)
		if n == 0 || err != nil {
			break
		}
		lineBuf = append(lineBuf, b[0])
		if b[0] == '\n' {
			line := string(lineBuf)
			lineBuf = lineBuf[:0]
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				data = strings.TrimSpace(data)
				if data == "[DONE]" || data == "" {
					continue
				}
				var sseData struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(data), &sseData); err == nil {
					if len(sseData.Choices) > 0 && sseData.Choices[0].Delta.Content != "" {
						c.SSEvent("chunk", sseData.Choices[0].Delta.Content)
						c.Writer.Flush()
					}
				}
			}
		}
	}

	c.Writer.Flush()
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

	resp, err := httpClient.Do(req)
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
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w (content preview: %s)", err, aiContent[:min(200, len(aiContent))])
	}

	return &result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
