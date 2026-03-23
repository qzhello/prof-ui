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
	"regexp"
	"runtime"
	"strings"
	"time"

	"prof-analyzer/prompts"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type AIRequest struct {
	Model    string      `json:"model"`
	Messages []AIMessage `json:"messages"`
	Stream   bool        `json:"stream"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

	// API Key 鉴权中间件
	apiKey := os.Getenv("API_KEY")
	if apiKey != "" {
		authMiddleware := func(c *gin.Context) {
			token := c.GetHeader("X-API-Key")
			if token == "" {
				token = c.Query("api_key")
			}
			if token != apiKey {
				c.JSON(401, gin.H{"error": "Unauthorized: invalid or missing API key"})
				c.Abort()
				return
			}
			c.Next()
		}
		// 对 /api/ 路由应用鉴权，但 /api/health 可选
		r.POST("/api/analyze/stream", authMiddleware, handleAnalyzeStream)
		r.POST("/api/pprof/text", authMiddleware, handlePprofText)
		r.POST("/api/pprof/image", authMiddleware, handlePprofImage)
		log.Println("API Key authentication enabled")
	} else {
		r.POST("/api/analyze/stream", handleAnalyzeStream)
		r.POST("/api/pprof/text", handlePprofText)
		r.POST("/api/pprof/image", handlePprofImage)
		log.Println("WARNING: API Key authentication is DISABLED (API_KEY not set)")
	}
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

// runGoToolPprof saves the profile and runs `go tool pprof -text` to get human-readable output.
func runGoToolPprof(rawData []byte, filename string) (string, bool, error) {
	tmpDir := filepath.Join(".", "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", false, fmt.Errorf("failed to create tmp dir: %w", err)
	}
	profPath := filepath.Join(tmpDir, "pprof_"+filepath.Base(filename))
	if err := os.WriteFile(profPath, rawData, 0644); err != nil {
		return "", false, fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(profPath)

	// Find pprof binary via GOROOT env
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		log.Printf("[ERROR] GOROOT not set")
		hex := formatHexDump(rawData)
		return hex, false, fmt.Errorf("GOROOT not set")
	}
	log.Printf("[INFO] GOROOT: %s", goroot)

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	gotoolDir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
	pprofBin := filepath.Join(gotoolDir, "pprof")
	log.Printf("[INFO] pprof path: %s", pprofBin)

	if _, err := os.Stat(pprofBin); err != nil {
		log.Printf("[ERROR] pprof not found at: %s", pprofBin)
		hex := formatHexDump(rawData)
		return hex, false, fmt.Errorf("pprof not found")
	}

	// Method 1: pprof -raw
	cmd := exec.Command(pprofBin, "-raw", profPath)
	out, cmdErr := cmd.CombinedOutput()
	if cmdErr == nil && len(out) > 0 {
		result := parsePprofRawOutput(string(out))
		if result != "" {
			log.Printf("[INFO] pprof -raw parsed: %d bytes", len(result))
			return result, true, nil
		}
	}
	log.Printf("[WARN] pprof -raw failed: %v, trying -text", cmdErr)

	// Method 2: pprof -text
	cmd = exec.Command(pprofBin, "-text", profPath)
	out, cmdErr = cmd.CombinedOutput()
	if cmdErr == nil && len(out) > 0 {
		log.Printf("[INFO] pprof -text output: %d bytes", len(out))
		return string(out), true, nil
	}
	if cmdErr == nil && len(out) > 0 {
		result := parsePprofRawOutput(string(out))
		if result != "" {
			log.Printf("[INFO] pprof -raw parsed: %d bytes", len(result))
			return result, true, nil
		}
	}

	// Last resort: hex dump
	hex := formatHexDump(rawData)
	log.Printf("[WARN] all pprof methods failed, using hex dump")
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
	}

	if goodLines > 0 && sb.Len() > 50 {
		return sb.String()
	}
	return ""
}

// extractSourceFiles extracts unique file paths from pprof text output.
// It looks for patterns like "file.go:123" or "/path/to/file.go:456".
func extractSourceFiles(pprofText string) []string {
	// Match patterns like "file.go:123" or "/path/to/file.go:456"
	// The file path can contain word characters, dots, slashes, hyphens
	re := regexp.MustCompile(`([\w\-./]+):(\d+)`)
	matches := re.FindAllStringSubmatch(pprofText, -1)

	seen := make(map[string]bool)
	var files []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		filePath := match[1]
		// Filter out obviously non-Go source files and numeric-only paths
		if strings.HasSuffix(filePath, ".go") && !seen[filePath] {
			// Skip if it looks like a standard library path without source
			if !strings.Contains(filePath, "$") && !strings.HasPrefix(filePath, "built-in") {
				seen[filePath] = true
				files = append(files, filePath)
			}
		}
	}

	return files
}

// extractTopFunctions extracts function names from pprof text output.
// It returns the top N functions sorted by cumulative percentage.
func extractTopFunctions(pprofText string, maxFuncs int) []string {
	lines := strings.Split(pprofText, "\n")
	var functions []string
	funcRe := regexp.MustCompile(`^\s*[\d.]+[a-z]*s\s+[\d.]+%\s+[\d.]+%\s+([\w/.@]+)`)

	seen := make(map[string]bool)
	for _, line := range lines {
		matches := funcRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			funcName := matches[1]
			// Skip runtime/internal/standard library functions
			if !seen[funcName] && !strings.HasPrefix(funcName, "internal/runtime") &&
				!strings.HasPrefix(funcName, "runtime.") && !strings.HasPrefix(funcName, "internal/poll") &&
				!strings.HasPrefix(funcName, "internal/syscall") && !strings.HasPrefix(funcName, "syscall.") &&
				!strings.HasPrefix(funcName, "hash/crc32") && !strings.HasPrefix(funcName, "bytes.") &&
				!strings.HasPrefix(funcName, "strings.") && !strings.HasPrefix(funcName, "io.") &&
				!strings.HasPrefix(funcName, "bufio.") && !strings.HasPrefix(funcName, "net/http") &&
				!strings.HasPrefix(funcName, "mime/") && !strings.HasPrefix(funcName, "os.") &&
				!strings.HasPrefix(funcName, "time.") && !strings.HasPrefix(funcName, "fmt.") &&
				!strings.HasPrefix(funcName, "math/") && !strings.HasPrefix(funcName, "strconv.") &&
				!strings.HasPrefix(funcName, "unicode.") && !strings.HasPrefix(funcName, "sort.") &&
				!strings.HasPrefix(funcName, "container/") && !strings.HasPrefix(funcName, "reflect.") &&
				!strings.HasPrefix(funcName, "sync.") && !strings.HasPrefix(funcName, "errors.") &&
				!strings.HasPrefix(funcName, "path.") && !strings.HasPrefix(funcName, "filepath.") &&
				!strings.HasPrefix(funcName, "io/ioutil") && !strings.HasPrefix(funcName, "log.") &&
				!strings.HasPrefix(funcName, "encoding/") && !strings.HasPrefix(funcName, "context.") &&
				!strings.HasPrefix(funcName, "crypto/") && !strings.HasPrefix(funcName, "compress/") &&
				!strings.HasPrefix(funcName, "archive/") && !strings.HasPrefix(funcName, "vendor/") &&
				!strings.Contains(funcName, "$") && funcName != "???" {
				seen[funcName] = true
				functions = append(functions, funcName)
				if len(functions) >= maxFuncs {
					break
				}
			}
		}
	}
	return functions
}

// convertFuncNamesToFiles converts function names to potential file paths.
// e.g., github.com/seaweedfs/seaweedfs/weed/storage/needle.ParseUpload -> weed/storage/needle.go
func convertFuncNamesToFiles(funcNames []string) []string {
	seen := make(map[string]bool)
	var files []string

	for _, funcName := range funcNames {
		// Split by "/" to get package path components
		parts := strings.Split(funcName, "/")
		if len(parts) < 3 {
			continue
		}

		// Look for the last part that looks like a file (contains dot or is followed by func name)
		// Pattern: github.com/org/project/package/file.Package/func or github.com/org/project/package/file.go
		for i := len(parts) - 1; i >= 0; i-- {
			part := parts[i]
			// Skip domain parts (github.com, go.uber.io, etc.)
			if i < 2 {
				break
			}
			// If part ends with .go, it's likely a file
			if strings.HasSuffix(part, ".go") {
				filePath := strings.Join(parts[i:], "/")
				if !seen[filePath] {
					seen[filePath] = true
					files = append(files, filePath)
				}
				break
			}
			// If part contains a dot (like Package.Func), it might be a file without extension
			if strings.Contains(part, ".") && !strings.Contains(part, "/") {
				// Try adding .go extension
				filePath := strings.Join(parts[i:], "/") + ".go"
				if !seen[filePath] {
					seen[filePath] = true
					files = append(files, filePath)
				}
				break
			}
		}
	}

	return files
}

// readSourceCode reads the content of the specified Go source files.
// It returns a map of filename -> content, ignoring files that don't exist.
// Each file is limited to 500 lines to prevent token overflow.
func readSourceCode(sourcePath string, files []string) map[string]string {
	const maxLines = 500
	result := make(map[string]string)

	for _, file := range files {
		// Skip if no source path provided
		if sourcePath == "" {
			continue
		}

		// Construct full path
		fullPath := filepath.Join(sourcePath, file)

		// Check if file exists
		info, err := os.Stat(fullPath)
		if err != nil {
			log.Printf("[WARN] readSourceCode: file not found: %s", fullPath)
			continue
		}
		if info.IsDir() {
			continue
		}

		// Read file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[WARN] readSourceCode: failed to read %s: %v", fullPath, err)
			continue
		}

		// Limit to maxLines
		lines := strings.Split(string(content), "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			result[file] = strings.Join(lines, "\n") + "\n... (内容过长已截断)"
		} else {
			result[file] = string(content)
		}

		log.Printf("[INFO] readSourceCode: read %s (%d lines)", file, len(lines))
	}

	return result
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
		"success": true,
		"text":    text,
		"path":    textPath,
		"message": "已用 go tools 分析，结果已保存",
	})
}

// handlePprofImage generates a PNG flame graph from pprof data
func handlePprofImage(c *gin.Context) {
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

	log.Printf("[INFO] handlePprofImage: processing %s (%d bytes)", file.Filename, len(rawData))

	// Save to tmp dir
	tmpDir := filepath.Join(".", "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(500, gin.H{"error": "failed to create tmp dir"})
		return
	}
	profPath := filepath.Join(tmpDir, "pprof_"+filepath.Base(file.Filename))
	if err := os.WriteFile(profPath, rawData, 0644); err != nil {
		c.JSON(500, gin.H{"error": "failed to write temp file"})
		return
	}
	defer os.Remove(profPath)

	// Find pprof binary
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		log.Printf("[ERROR] GOROOT not set")
		c.JSON(500, gin.H{"error": "GOROOT not set, please configure GOROOT"})
		return
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	gotoolDir := filepath.Join(goroot, "pkg", "tool", goos+"_"+goarch)
	pprofBin := filepath.Join(gotoolDir, "pprof")

	if _, err := os.Stat(pprofBin); err != nil {
		log.Printf("[ERROR] pprof not found at: %s", pprofBin)
		c.JSON(500, gin.H{"error": "pprof binary not found"})
		return
	}

	// Generate PNG using -png -output flags
	pngPath := outputPath("pprof_image", "png")
	cmd := exec.Command(pprofBin, "-png", "-output="+pngPath, profPath)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Printf("[ERROR] pprof -output failed: %v", cmdErr)
		c.JSON(500, gin.H{"error": "failed to generate pprof image: " + cmdErr.Error()})
		return
	}

	log.Printf("[INFO] handlePprofImage: image saved to %s", pngPath)
	c.JSON(200, gin.H{
		"success": true,
		"path":    pngPath,
		"url":     "/" + pngPath,
		"message": "图片已生成",
	})
}

// handleAnalyzeStream: first converts pprof to text, then streams AI response token by token
func handleAnalyzeStream(c *gin.Context) {
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}
	outputLanguage := os.Getenv("OUTPUT_LANGUAGE")
	if outputLanguage == "" {
		outputLanguage = "中文"
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

	sourcePath := c.PostForm("source_path")

	// Extract source files from pprof text and read their content
	var sourceCode map[string]string
	if sourcePath != "" {
		// Find pprof binary path
		goroot := os.Getenv("GOROOT")
		if goroot != "" {
			for _, pprofText := range analysisTexts {
				// First try direct file:line extraction
				files := extractSourceFiles(pprofText)
				if len(files) == 0 {
					// If no files found, extract top functions and use heuristic
					topFuncs := extractTopFunctions(pprofText, 15)
					if len(topFuncs) > 0 {
						log.Printf("[INFO] handleAnalyzeStream: extracted %d top functions, using heuristic for file locations", len(topFuncs))
						files = convertFuncNamesToFiles(topFuncs)
					}
				}
				if len(files) > 0 {
					sourceCode = readSourceCode(sourcePath, files)
					if len(sourceCode) > 0 {
						log.Printf("[INFO] handleAnalyzeStream: successfully read %d source files", len(sourceCode))
						break
					}
				}
			}
		}
	}

	prompt := buildPrompt(fileNames, analysisTexts, sourcePath, sourceCode)

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
			{Role: "system", Content: prompts.StreamingMarkdownSystemPrompt(outputLanguage)},
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

	// Clean accumulated content - remove anything before ```markdown
	cleaned := accumulated
	if idx := strings.Index(cleaned, "```markdown"); idx != -1 {
		cleaned = cleaned[idx:]
	} else if idx := strings.Index(cleaned, "```"); idx != -1 {
		cleaned = cleaned[idx:]
	}

	// Save the complete accumulated response
	send("status", "正在保存结果...")
	if cleaned != "" {
		// Save as text file since it's now markdown
		resultPath := outputPath("analysis_result", "txt")
		os.WriteFile(resultPath, []byte(cleaned), 0644)
		send("saved", resultPath)
		log.Printf("[INFO] handleAnalyzeStream: result saved to %s", resultPath)
	}

	send("done", "")
	log.Printf("[INFO] handleAnalyzeStream: stream finished")
}

func buildPrompt(fileNames, analysisTexts []string, sourcePath string, sourceCode map[string]string) string {
	var sb strings.Builder
	sb.WriteString("你是一个专业的性能分析专家。以下是 PROF 文件经过 go tools 分析后的文本数据，请基于这些数据进行详细分析。\n\n")

	if sourcePath != "" {
		sb.WriteString(fmt.Sprintf("本地源码路径: %s\n\n", sourcePath))
	}

	// Add source code content
	if len(sourceCode) > 0 {
		sb.WriteString("=== 源码文件 ===\n\n")
		for filename, content := range sourceCode {
			sb.WriteString(fmt.Sprintf("=== 源码文件: %s ===\n", filename))
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
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

	sb.WriteString(`请分析以上 PROF 数据，返回规范的 Markdown 格式报告。

【格式要求】
- 使用标题、表格、列表等 Markdown 元素
- 性能热点和排名数据只返回 Top 10
- 表格要有清晰的表头
- 保持格式整洁美观

【必须包含的章节】
1. 总览：简要总结主要性能问题
2. 问题根因：清晰描述导致性能问题的根本原因
3. 性能热点：表格展示 Top 10 函数（排名、函数名、位置、耗时、占比、调用次数）
4. 调用链路：表格展示关键调用关系
5. 解决建议：编号列表形式
6. 指标汇总：表格展示关键指标

【注意事项】
- 只返回 Markdown 格式，不要 JSON
- 不要使用代码块包裹整个输出
- 表格使用标准 Markdown 表格语法`)

	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
