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

var httpClient = &http.Client{}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

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
	r.POST("/api/pprof/text", handlePprofText)
	r.GET("/api/health", handleHealth)

	log.Printf("Server starting on :%s", port)
	log.Printf("Output directory: ./output")
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func outputPath(prefix, ext string) string {
	ts := time.Now().Format("20060102_150405")
	return filepath.Join("output", fmt.Sprintf("%s_%s.%s", prefix, ts, ext))
}

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

// runGoToolPprof saves the profile and runs `go tool pprof -text` to get human-readable output.
// It writes to a temp text file and reads back the result to avoid TTY/format issues.
func runGoToolPprof(rawData []byte, filename string) (string, bool, error) {
	tmpDir := filepath.Join(".", "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", false, fmt.Errorf("failed to create tmp dir: %w", err)
	}
	profPath := filepath.Join(tmpDir, "pprof_"+filepath.Base(filename))
	textOutPath := filepath.Join(tmpDir, "pprof_text_"+filepath.Base(filename)+".txt")

	if err := os.WriteFile(profPath, rawData, 0644); err != nil {
		return "", false, fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(profPath)
	defer os.Remove(textOutPath)

	// Find go binary and set up environment
	goBin, err := exec.LookPath("go")
	if err != nil {
		log.Printf("[ERROR] go not found in PATH: %v", err)
		hex := formatHexDump(rawData)
		return hex, false, fmt.Errorf("go not found in PATH")
	}

	gorootCmd := exec.Command(goBin, "env", "GOROOT")
	gorootOut, _ := gorootCmd.Output()
	goroot := strings.TrimSpace(string(gorootOut))
	env := os.Environ()
	if goroot != "" {
		env = append(env, "GOROOT="+goroot)
	}

	// Use -text -output to write directly to file, bypassing TTY issues
	cmd := exec.Command(goBin, "tool", "pprof", "-text", "-output", textOutPath, profPath)
	cmd.Env = env
	cmd.Dir = tmpDir
	outBytes, err := cmd.CombinedOutput()
	outStr := string(outBytes)
	log.Printf("[INFO] go tool pprof -text -output: err=%v output_len=%d output=%q", err, len(outStr), outStr)

	// Read back the text output file
	textOut, err := os.ReadFile(textOutPath)
	if err == nil && len(textOut) > 0 {
		result := string(textOut)
		if isPrintableASCII(result) {
			log.Printf("[INFO] pprof text file read: %d bytes", len(result))
			return result, true, nil
		}
		log.Printf("[WARN] pprof text file not readable ASCII, falling back")
	}

	// Fallback: try -raw which outputs proto in text format
	cmd = exec.Command(goBin, "tool", "pprof", "-raw", profPath)
	cmd.Env = env
	cmd.Dir = tmpDir
	rawOut, err := cmd.CombinedOutput()
	if err == nil && len(rawOut) > 0 {
		result := parsePprofRawOutput(string(rawOut))
		if result != "" {
			log.Printf("[INFO] pprof -raw parsed: %d bytes", len(result))
			return result, true, nil
		}
	}

	// Last resort: hex dump
	hex := formatHexDump(rawData)
	log.Printf("[WARN] pprof conversion failed, using hex dump fallback")
	return hex, false, nil
}

// parsePprofRawOutput parses the raw proto text format output by `go tool pprof -raw`
// and extracts human-readable profile data.
func parsePprofRawOutput(raw string) string {
	var sb strings.Builder
	lines := strings.Split(raw, "\n")

	// Check if it looks like actual profile data (not binary)
	goodLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Accept lines that are printable ASCII and contain useful profile info
		if isPrintableASCII(trimmed) &&
			(strings.Contains(trimmed, ":") || strings.Contains(trimmed, "0x") ||
			 strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "=") ||
			 len(trimmed) > 3) {
			goodLines++
			sb.WriteString(trimmed)
			sb.WriteString("\n")
		}
	}

	if goodLines > 0 && sb.Len() > 50 {
		return sb.String()
	}
	return ""
}

// formatHexDump returns a hex dump of the data with ASCII representation.
func formatHexDump(data []byte) string {
	const linesize = 32
	var sb strings.Builder
	sb.WriteString("=== RAW PROFILE DATA (hex dump) ===\n")
	for i := 0; i < len(data); i += linesize {
		end := i + linesize
		if end > len(data) {
			end = len(data)
		}
		hex := ""
		for j := i; j < end; j++ {
			hex += fmt.Sprintf("%02x ", data[j])
		}
		ascii := ""
		for j := i; j < end; j++ {
			b := data[j]
			if b >= 32 && b < 127 {
				ascii += string(b)
			} else {
				ascii += "."
			}
		}
		sb.WriteString(fmt.Sprintf("%08x  %-96s  %s\n", i, hex, ascii))
	}
	sb.WriteString("=================================\n")
	return sb.String()
}

// handlePprofText runs pprof text analysis on the uploaded file and returns the text.
func handlePprofText(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "no file uploaded"})
		return
	}

	if file.Size > 100*1024*1024 {
		c.JSON(400, gin.H{"error": "file too large"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to open file"})
		return
	}
	rawData, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to read file"})
		return
	}

	log.Printf("[INFO] handlePprofText: processing %s (%d bytes)", file.Filename, len(rawData))
	text, ok, err := runGoToolPprof(rawData, filepath.Base(file.Filename))
	if err != nil {
		log.Printf("[ERROR] handlePprofText: %v", err)
	} else if !ok {
		log.Printf("[WARN] handlePprofText: go tool pprof not available, using hex dump")
	}

	// Save text output
	textPath := outputPath("pprof_text", "txt")
	os.WriteFile(textPath, []byte(text), 0644)
	log.Printf("[INFO] handlePprofText: text saved to %s", textPath)

	c.JSON(200, gin.H{
		"success":  true,
		"text":    text,
		"path":    textPath,
		"message": "已用 go tools 分析，结果已保存",
	})
}

// handleSaveResult saves a complete analysis result JSON to ./output/
func handleSaveResult(c *gin.Context) {
	var result AnalysisResult
	if err := c.ShouldBindJSON(&result); err != nil {
		log.Printf("[ERROR] handleSaveResult: %v", err)
		c.JSON(400, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	resultPath, err := saveJSON(result, "analysis", "json")
	if err != nil {
		log.Printf("[ERROR] handleSaveResult: %v", err)
		c.JSON(500, gin.H{"error": "failed to save: " + err.Error()})
		return
	}

	log.Printf("[INFO] Analysis result saved to: %s", resultPath)
	c.JSON(200, gin.H{
		"success":     true,
		"result_path": resultPath,
		"result_url":  "/" + resultPath,
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

	if apiKey == "" {
		c.JSON(500, gin.H{"error": "AI_API_KEY not configured"})
		return
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

	var analysisTexts []string
	var fileNames []string

	for _, f := range files {
		fileNames = append(fileNames, f.Filename)
		src, err := f.Open()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file"})
			return
		}
		rawData, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file content"})
			return
		}

		// Convert binary pprof to text using go tool pprof
		pprofText, ok, err := runGoToolPprof(rawData, f.Filename)
		if err != nil {
			log.Printf("[WARN] handleAnalyze: pprof conversion failed for %s: %v", f.Filename, err)
		} else if !ok {
			log.Printf("[WARN] handleAnalyze: go tool pprof unavailable for %s, using hex dump", f.Filename)
		}

		// Log preview of what we're sending to AI
		preview := pprofText
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		if ok {
			log.Printf("[INFO] handleAnalyze: pprof text preview for %s:\n%s", f.Filename, preview)
		} else {
			log.Printf("[INFO] handleAnalyze: hex dump preview for %s:\n%s", f.Filename, preview)
		}
		analysisTexts = append(analysisTexts, pprofText)
	}

	prompt := buildPrompt(fileNames, analysisTexts, c.PostForm("source_path"))

	// Log full prompt so user can verify what's being sent to AI
	log.Printf("[INFO] ===== PROMPT SENT TO AI =====")
	log.Printf("%s", prompt)
	log.Printf("[INFO] ===== END OF PROMPT =====")
	log.Printf("[INFO] handleAnalyze: calling AI model=%s files=%v", model, fileNames)
	analysis, err := callAI(apiKey, baseURL, model, prompt)
	if err != nil {
		log.Printf("[ERROR] handleAnalyze: AI call failed: %v", err)
		c.JSON(500, gin.H{"error": "analysis failed: " + err.Error()})
		return
	}

	resultPath, err := saveJSON(analysis, "analysis", "json")
	if err != nil {
		log.Printf("[WARN] handleAnalyze: failed to save result: %v", err)
	} else {
		log.Printf("[INFO] handleAnalyze: result saved to %s", resultPath)
	}

	c.JSON(200, gin.H{
		"success":     true,
		"data":        analysis,
		"result_url":  "/" + resultPath,
		"result_path": resultPath,
	})
}

// handleAnalyzeStream: first converts pprof to text, then streams AI response token by token
func handleAnalyzeStream(c *gin.Context) {
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}
	apiKey := os.Getenv("AI_API_KEY")
	baseURL := os.Getenv("AI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	if apiKey == "" {
		c.JSON(500, gin.H{"error": "AI_API_KEY not configured"})
		return
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

	var analysisTexts []string
	var fileNames []string

	for _, f := range files {
		fileNames = append(fileNames, f.Filename)
		src, err := f.Open()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file"})
			return
		}
		rawData, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to read file content"})
			return
		}

		pprofText, ok, err := runGoToolPprof(rawData, f.Filename)
		if err != nil {
			log.Printf("[WARN] handleAnalyzeStream: pprof conversion failed for %s: %v", f.Filename, err)
		} else if !ok {
			log.Printf("[WARN] handleAnalyzeStream: go tool pprof unavailable for %s, using hex dump", f.Filename)
		}
		preview := pprofText
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("[INFO] handleAnalyzeStream: sending to AI (isPprofText=%v): %s", ok, preview)
		analysisTexts = append(analysisTexts, pprofText)
	}

	prompt := buildPrompt(fileNames, analysisTexts, c.PostForm("source_path"))

	log.Printf("[INFO] ===== STREAMING PROMPT SENT TO AI =====")
	log.Printf("%s", prompt)
	log.Printf("[INFO] ===== END OF PROMPT =====")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	flush := func() {
		if fw, ok := c.Writer.(http.Flusher); ok {
			fw.Flush()
		}
	}

	send := func(event, data string) {
		c.Writer.Write([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)))
		flush()
	}

	send("status", "开始分析...")

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
		send("error", "failed to marshal request: "+err.Error())
		return
	}

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		send("error", "failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	log.Printf("[INFO] handleAnalyzeStream: calling AI model=%s files=%v", model, fileNames)
	resp, err := httpClient.Do(req)
	if err != nil {
		send("error", "AI request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		send("error", fmt.Sprintf("AI API error %d: %s", resp.StatusCode, string(body)))
		return
	}

	send("status", "AI 正在分析...")

	reader := resp.Body
	lineBuf := []byte{}
	accumulated := ""

	for {
		b := make([]byte, 1)
		n, rerr := reader.Read(b)
		if n == 0 || rerr != nil {
			break
		}
		lineBuf = append(lineBuf, b[0])
		if b[0] != '\n' {
			continue
		}
		line := string(lineBuf)
		lineBuf = lineBuf[:0]

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}

		var sseData struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &sseData); err != nil {
			continue
		}
		if len(sseData.Choices) == 0 {
			continue
		}

		token := sseData.Choices[0].Delta.Content
		if token == "" {
			continue
		}

		accumulated += token

		// Escape newlines for SSE transport
		escaped := strings.ReplaceAll(token, "\n", "\\n")
		escaped = strings.ReplaceAll(escaped, "\r", "\\r")
		send("chunk", escaped)
	}

	// Save the complete accumulated response
	send("status", "正在保存结果...")
	if accumulated != "" {
		var result AnalysisResult
		if err := json.Unmarshal([]byte(accumulated), &result); err == nil {
			resultPath, err := saveJSON(result, "analysis_stream", "json")
			if err == nil {
				send("saved", resultPath)
				log.Printf("[INFO] handleAnalyzeStream: result saved to %s", resultPath)
			}
		}
	}

	send("done", "")
	log.Printf("[INFO] handleAnalyzeStream: stream finished")
}

func buildPrompt(fileNames, analysisTexts []string, sourcePath string) string {
	var sb strings.Builder
	sb.WriteString("你是一个专业的性能分析专家。以下是 PROF 文件经过 go tools 分析后的文本数据，请基于这些数据进行详细分析。\n\n")

	if sourcePath != "" {
		sb.WriteString(fmt.Sprintf("本地源码路径: %s\n\n", sourcePath))
	}

	for i, name := range fileNames {
		sb.WriteString(fmt.Sprintf("=== 文件 %d: %s (go tools 分析结果) ===\n", i+1, name))
		text := analysisTexts[i]
		if len(text) > 20000 {
			text = text[:20000] + "\n... (内容过长已截断)"
		}
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`请分析以上 PROF 分析数据，返回以下格式的 JSON（确保 JSON 可直接解析）：
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
}`)

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
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w (preview: %s)", err, aiContent[:min(200, len(aiContent))])
	}

	return &result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
