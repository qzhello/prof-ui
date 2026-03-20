<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import FileUpload from './components/FileUpload.vue'
import AnalysisResult from './components/AnalysisResult.vue'
import Charts from './components/Charts.vue'
import PdfExport from './components/PdfExport.vue'
import type { UploadedFile, AnalysisResult as AnalysisResultType } from './types'

// Dark mode
const isDark = ref(false)
watch(isDark, (val) => {
  if (val) {
    document.documentElement.classList.add('dark')
    localStorage.setItem('theme', 'dark')
  } else {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('theme', 'light')
  }
}, { immediate: true })

const activeTab = ref<'upload' | 'result' | 'charts' | 'export'>('upload')
const uploadedFiles = ref<UploadedFile[]>([])
const sourcePath = ref('')
const isAnalyzing = ref(false)
const analysisResult = ref<AnalysisResultType | null>(null)
const errorMessage = ref('')

// Saved file paths
const savedResultPath = ref('')
const savedPprofPath = ref('')
const pprofImageUrl = ref('')
const isStreaming = ref(false)
const streamingOutput = ref('')

// Upload progress
const uploadProgress = ref(0)

// Streaming abort controller
let streamAbortController: AbortController | null = null

const hasResult = computed(() => analysisResult.value !== null)

function handleFilesSelected(files: UploadedFile[]) {
  uploadedFiles.value = files
}

function handleSourcePathChange(path: string) {
  sourcePath.value = path
}

async function startAnalysis() {
  if (uploadedFiles.value.length === 0) {
    errorMessage.value = '请先选择要分析的文件'
    return
  }

  isAnalyzing.value = true
  errorMessage.value = ''
  savedResultPath.value = ''
  uploadProgress.value = 0
  activeTab.value = 'upload'

  try {
    const { analyzeFiles } = await import('./api')
    const { result, resultPath } = await analyzeFiles(
      uploadedFiles.value,
      sourcePath.value,
      undefined,
      (pct) => { uploadProgress.value = pct }
    )
    analysisResult.value = result
    savedResultPath.value = resultPath || ''
    activeTab.value = 'result'
  } catch (err: any) {
    errorMessage.value = err.message || '分析过程中发生错误'
  } finally {
    isAnalyzing.value = false
    uploadProgress.value = 0
  }
}

async function startStreamingAnalysis() {
  if (uploadedFiles.value.length === 0) {
    errorMessage.value = '请先选择要分析的文件'
    return
  }

  isStreaming.value = true
  errorMessage.value = ''
  streamingOutput.value = ''
  uploadProgress.value = 0
  activeTab.value = 'result'
  streamAbortController = new AbortController()

  try {
    const formData = new FormData()
    uploadedFiles.value.forEach((f) => formData.append('files', f.file))
    if (sourcePath.value) formData.append('source_path', sourcePath.value)

    const response = await fetch('/api/analyze/stream', {
      method: 'POST',
      body: formData,
      signal: streamAbortController.signal,
    })

    if (!response.ok) {
      const err = await response.text()
      throw new Error(`Server error: ${response.status} - ${err}`)
    }

    const reader = response.body?.getReader()
    if (!reader) throw new Error('No response body')

    const decoder = new TextDecoder()
    let buffer = ''
    let currentEvent = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('event: ')) {
          currentEvent = line.slice(7).trim()
        } else if (line.startsWith('data: ')) {
          let data = line.slice(6).trim()
          if (!data || data === '[DONE]') continue

          if (currentEvent === 'status') {
            // ignore
          } else if (currentEvent === 'chunk') {
            data = data.replace(/\\n/g, '\n').replace(/\\r/g, '\r')
            streamingOutput.value += data
          } else if (currentEvent === 'error') {
            errorMessage.value = data
          } else if (currentEvent === 'saved') {
            savedResultPath.value = data
          }
        } else if (line === '') {
          currentEvent = ''
        }
      }
    }

    const jsonStart = streamingOutput.value.indexOf('{')
    if (jsonStart !== -1) {
      try {
        const jsonStr = streamingOutput.value.slice(jsonStart)
        const parsed = JSON.parse(jsonStr)
        analysisResult.value = parsed
        const { saveResultJSON } = await import('./api')
        const path = await saveResultJSON(parsed)
        savedResultPath.value = path || ''
        activeTab.value = 'result'
      } catch {
        errorMessage.value = '流式输出解析失败，请尝试普通分析模式'
        activeTab.value = 'upload'
      }
    }
  } catch (err: any) {
    if (err.name === 'AbortError') {
      errorMessage.value = '流式分析已停止'
    } else {
      errorMessage.value = err.message || '流式分析失败'
    }
    activeTab.value = 'upload'
  } finally {
    isStreaming.value = false
    streamAbortController = null
    uploadProgress.value = 0
  }
}

function stopStreaming() {
  streamAbortController?.abort()
}

async function generatePprofImage() {
  if (uploadedFiles.value.length === 0) {
    errorMessage.value = '请先选择要分析的文件'
    return
  }
  errorMessage.value = ''
  try {
    const { generatePprofImage: callPprof } = await import('./api')
    const resp = await callPprof(uploadedFiles.value[0].file)
    if (!resp.success) {
      errorMessage.value = resp.error || '生成 pprof 图片失败'
      return
    }
    savedPprofPath.value = resp.path || ''
    pprofImageUrl.value = resp.url || ''
    activeTab.value = 'result'
  } catch (err: any) {
    errorMessage.value = err.message || '生成图片失败'
  }
}

function clearResults() {
  analysisResult.value = null
  streamingOutput.value = ''
  savedResultPath.value = ''
  savedPprofPath.value = ''
  pprofImageUrl.value = ''
  uploadedFiles.value = []
  sourcePath.value = ''
  errorMessage.value = ''
  activeTab.value = 'upload'
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900 transition-colors duration-200">
    <!-- Header -->
    <header class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 bg-primary-600 rounded-lg flex items-center justify-center">
              <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h1 class="text-xl font-bold text-gray-900 dark:text-white">PROF Analyzer</h1>
              <p class="text-sm text-gray-500 dark:text-gray-400">智能性能分析工具</p>
            </div>
          </div>
          <div class="flex items-center space-x-3">
            <!-- Dark mode toggle -->
            <button
              @click="isDark = !isDark"
              class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              :title="isDark ? '切换到浅色模式' : '切换到深色模式'"
            >
              <!-- Sun icon (shown in dark mode to switch to light) -->
              <svg v-if="isDark" class="w-5 h-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
              <!-- Moon icon (shown in light mode to switch to dark) -->
              <svg v-else class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
              </svg>
            </button>
            <button
              v-if="hasResult"
              @click="clearResults"
              class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            >
              新建分析
            </button>
          </div>
        </div>
      </div>
    </header>

    <!-- Navigation Tabs -->
    <nav class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex space-x-8">
          <button
            v-for="tab in [
              { key: 'upload', label: '上传文件' },
              { key: 'result', label: '分析结果', disabled: !hasResult },
              { key: 'charts', label: '可视化', disabled: !hasResult },
              { key: 'export', label: '导出PDF', disabled: !hasResult }
            ]"
            :key="tab.key"
            @click="tab.disabled ? null : (activeTab = tab.key as any)"
            :class="[
              'py-4 px-1 border-b-2 text-sm font-medium transition-colors',
              activeTab === tab.key
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : tab.disabled
                  ? 'border-transparent text-gray-300 dark:text-gray-600 cursor-not-allowed'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:border-gray-300 dark:hover:border-gray-600'
            ]"
          >
            {{ tab.label }}
          </button>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Error Message -->
      <div v-if="errorMessage" class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <div class="flex">
          <svg class="w-5 h-5 text-red-400 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="ml-3 text-sm text-red-700 dark:text-red-300">{{ errorMessage }}</p>
        </div>
      </div>

      <!-- Saved Paths Notice -->
      <div v-if="savedResultPath || savedPprofPath" class="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
        <div class="flex items-start space-x-3">
          <svg class="w-5 h-5 text-green-500 dark:text-green-400 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div>
            <p class="text-sm font-medium text-green-800 dark:text-green-300 mb-1">文件已保存到本地：</p>
            <ul class="text-xs text-green-700 dark:text-green-400 space-y-1">
              <li v-if="savedResultPath">
                📄 分析结果：
                <a :href="'/' + savedResultPath" target="_blank" class="underline hover:text-green-900 dark:hover:text-green-200">{{ savedResultPath }}</a>
              </li>
              <li v-if="savedPprofPath">
                🖼️ Pprof 图片：
                <a :href="'/' + savedPprofPath" target="_blank" class="underline hover:text-green-900 dark:hover:text-green-200">{{ savedPprofPath }}</a>
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Pprof Image Display -->
      <div v-if="pprofImageUrl && activeTab === 'result'" class="mb-6 bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-200 flex items-center space-x-2">
            <svg class="w-4 h-4 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>Pprof 可视化图片</span>
          </h3>
          <a :href="pprofImageUrl" download class="text-xs text-primary-600 dark:text-primary-400 hover:text-primary-800 dark:hover:text-primary-300 flex items-center space-x-1">
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            <span>下载</span>
          </a>
        </div>
        <div class="p-4 bg-gray-50 dark:bg-gray-900 flex justify-center">
          <img :src="pprofImageUrl" alt="Pprof Flame Graph" class="max-w-full h-auto rounded-lg shadow-sm" />
        </div>
      </div>

      <!-- Upload Tab -->
      <div v-if="activeTab === 'upload'">
        <FileUpload
          :files="uploadedFiles"
          :source-path="sourcePath"
          :is-analyzing="isAnalyzing"
          :is-streaming="isStreaming"
          :upload-progress="uploadProgress"
          @files-selected="handleFilesSelected"
          @source-path-change="handleSourcePathChange"
          @analyze="startAnalysis"
          @stream-analyze="startStreamingAnalysis"
          @generate-pprof="generatePprofImage"
        />
      </div>

      <!-- Streaming Panel -->
      <div v-if="activeTab === 'result' && isStreaming" class="bg-gray-900 rounded-2xl p-6 text-white shadow-xl">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center space-x-3">
            <div class="animate-pulse w-3 h-3 bg-green-400 rounded-full"></div>
            <h3 class="text-lg font-semibold text-green-400">AI 正在分析...</h3>
          </div>
          <button
            @click="stopStreaming"
            class="px-4 py-1.5 bg-red-600 hover:bg-red-700 text-white text-sm rounded-lg transition-colors flex items-center space-x-2"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
              <path d="M6 6h12v12H6z" />
            </svg>
            <span>停止</span>
          </button>
        </div>
        <pre class="text-sm text-green-300 font-mono whitespace-pre-wrap overflow-x-auto max-h-[60vh] leading-relaxed bg-gray-900 p-4 rounded-lg border border-gray-700">{{ streamingOutput || '等待响应...' }}</pre>
      </div>

      <!-- Result Tab -->
      <div v-if="activeTab === 'result' && hasResult && !isStreaming">
        <AnalysisResult :result="analysisResult!" />
      </div>

      <!-- Charts Tab -->
      <div v-if="activeTab === 'charts' && hasResult">
        <Charts :result="analysisResult!" />
      </div>

      <!-- Export Tab -->
      <div v-if="activeTab === 'export' && hasResult">
        <PdfExport :result="analysisResult!" />
      </div>
    </main>

    <!-- Analyzing Overlay -->
    <div
      v-if="isAnalyzing"
      class="fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center z-50"
    >
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl p-8 max-w-md w-full mx-4 text-center">
        <div class="animate-spin w-16 h-16 mx-auto mb-4 border-4 border-primary-200 dark:border-primary-700 border-t-primary-600 dark:border-t-primary-400 rounded-full"></div>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">AI 分析中...</h3>
        <!-- Upload progress bar -->
        <div v-if="uploadProgress > 0" class="mb-3">
          <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              class="bg-primary-500 h-2 rounded-full transition-all duration-300"
              :style="{ width: uploadProgress + '%' }"
            ></div>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ uploadProgress }}%</p>
        </div>
        <p class="text-sm text-gray-500 dark:text-gray-400">正在分析 PROF 文件，请稍候</p>
      </div>
    </div>
  </div>
</template>
