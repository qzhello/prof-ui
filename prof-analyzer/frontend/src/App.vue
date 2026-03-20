<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { marked } from 'marked'
import FileUpload from './components/FileUpload.vue'
import Charts from './components/Charts.vue'
import PdfExport from './components/PdfExport.vue'
import type { UploadedFile } from './types'

marked.setOptions({ breaks: true, gfm: true })

const streamingOutput = ref('')
const streamingContainerRef = ref<HTMLDivElement | null>(null)
const streamingFinished = ref(false)

const streamingHtml = computed(() => {
  if (!streamingOutput.value) return ''
  return marked.parse(streamingOutput.value)
})

watch(streamingOutput, async () => {
  await nextTick()
  if (streamingContainerRef.value) {
    streamingContainerRef.value.scrollTop = streamingContainerRef.value.scrollHeight
  }
})

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
const isStreaming = ref(false)
const errorMessage = ref('')
const savedResultPath = ref('')
const savedPprofPath = ref('')
const pprofImageUrl = ref('')
let streamAbortController: AbortController | null = null

const hasAnalysisContent = computed(() => streamingOutput.value.trim().length > 0)
const hasChartContent = computed(() => pprofImageUrl.value.trim().length > 0)
const canOpenResultTab = computed(() => isStreaming.value || hasAnalysisContent.value || streamingFinished.value)
const canOpenChartsTab = computed(() => hasChartContent.value)
const canOpenExportTab = computed(() => streamingFinished.value && hasAnalysisContent.value)

function handleFilesSelected(files: UploadedFile[]) { uploadedFiles.value = files }
function handleSourcePathChange(path: string) { sourcePath.value = path }

async function startStreamingAnalysis() {
  if (uploadedFiles.value.length === 0) {
    errorMessage.value = '请先选择要分析的文件'
    return
  }
  isStreaming.value = true
  errorMessage.value = ''
  streamingOutput.value = ''
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
    if (!response.ok) throw new Error(`Server error: ${response.status}`)
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
          if (currentEvent === 'chunk') {
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
    streamingFinished.value = true
  } catch (err: any) {
    if (err.name === 'AbortError') {
      errorMessage.value = '分析已停止'
    } else {
      errorMessage.value = err.message || '分析失败'
    }
    streamingFinished.value = true
  } finally {
    isStreaming.value = false
    streamAbortController = null
  }
}

function stopStreaming() { streamAbortController?.abort() }

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
  } catch (err: any) {
    errorMessage.value = err.message || '生成图片失败'
  }
}

function clearResults() {
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
            <button @click="isDark = !isDark" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
              <svg v-if="isDark" class="w-5 h-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
              <svg v-else class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
              </svg>
            </button>
            <button v-if="streamingFinished" @click="clearResults" class="px-4 py-2 text-sm text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors">新建分析</button>
          </div>
        </div>
      </div>
    </header>
    <nav class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex space-x-8">
          <button v-for="tab in [
              { key: 'upload', label: '上传文件', disabled: false },
              { key: 'result', label: '分析', disabled: !canOpenResultTab },
              { key: 'charts', label: '可视化', disabled: !canOpenChartsTab },
              { key: 'export', label: '导出PDF', disabled: !canOpenExportTab }
            ]" :key="tab.key" @click="tab.disabled ? null : (activeTab = tab.key as any)" :class="[
              'py-4 px-1 border-b-2 text-sm font-medium transition-colors',
              activeTab === tab.key
                ? 'border-primary-500 text-primary-600 dark:text-primary-400'
                : tab.disabled
                  ? 'border-transparent text-gray-300 dark:text-gray-600 cursor-not-allowed'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:border-gray-300 dark:hover:border-gray-600'
            ]">{{ tab.label }}</button>
        </div>
      </div>
    </nav>
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div v-if="errorMessage" class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
        <div class="flex">
          <svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          <p class="ml-3 text-sm text-red-700 dark:text-red-300">{{ errorMessage }}</p>
        </div>
      </div>
      <div v-if="savedResultPath || savedPprofPath" class="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
        <div class="flex items-start space-x-3">
          <svg class="w-5 h-5 text-green-500 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          <div>
            <p class="text-sm font-medium text-green-800 dark:text-green-300 mb-1">文件已保存到本地：</p>
            <ul class="text-xs text-green-700 dark:text-green-400 space-y-1">
              <li v-if="savedResultPath">📄 分析结果：<a :href="'/' + savedResultPath" target="_blank" class="underline">{{ savedResultPath }}</a></li>
              <li v-if="savedPprofPath">🖼️ Pprof 图片：<a :href="'/' + savedPprofPath" target="_blank" class="underline">{{ savedPprofPath }}</a></li>
            </ul>
          </div>
        </div>
      </div>
      <div v-if="activeTab === 'upload'">
        <FileUpload
          :files="uploadedFiles"
          :source-path="sourcePath"
          :is-streaming="isStreaming"
          @files-selected="handleFilesSelected"
          @source-path-change="handleSourcePathChange"
          @stream-analyze="startStreamingAnalysis"
          @generate-pprof="generatePprofImage"
        />
      </div>
      <div v-if="activeTab === 'result' && (isStreaming || streamingFinished)" class="min-h-[calc(100vh-200px)]">
        <div class="sticky top-0 z-10 bg-white border-b border-gray-200 py-4 mb-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-3">
              <div v-if="isStreaming" class="animate-pulse w-3 h-3 bg-green-500 rounded-full"></div>
              <svg v-else class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
              <h3 class="text-lg font-semibold text-gray-700">{{ isStreaming ? 'AI 正在分析...' : '分析完成' }}</h3>
            </div>
            <button v-if="isStreaming" @click="stopStreaming" class="px-4 py-1.5 bg-red-600 hover:bg-red-700 text-white text-sm rounded-lg transition-colors flex items-center space-x-2">
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M6 6h12v12H6z" /></svg>
              <span>停止</span>
            </button>
          </div>
        </div>
        <div class="bg-white rounded-xl p-6 shadow-sm border border-gray-100">
          <div ref="streamingContainerRef" class="markdown-body leading-relaxed text-sm" v-html="streamingHtml || '等待响应...'"></div>
        </div>
        <div v-if="errorMessage" class="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">{{ errorMessage }}</div>
      </div>
      <div v-if="activeTab === 'charts'">
        <Charts :pprof-image-url="pprofImageUrl" />
      </div>
      <div v-if="activeTab === 'export'">
        <PdfExport :html-content="streamingHtml" :markdown-content="streamingOutput" :can-export="streamingFinished && hasAnalysisContent" />
      </div>
    </main>
  </div>
</template>
