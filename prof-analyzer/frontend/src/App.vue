<script setup lang="ts">
import { ref, computed } from 'vue'
import FileUpload from './components/FileUpload.vue'
import AnalysisResult from './components/AnalysisResult.vue'
import Charts from './components/Charts.vue'
import PdfExport from './components/PdfExport.vue'
import type { UploadedFile, AnalysisResult as AnalysisResultType } from './types'

const activeTab = ref<'upload' | 'result' | 'charts' | 'export'>('upload')
const uploadedFiles = ref<UploadedFile[]>([])
const sourcePath = ref('')
const isAnalyzing = ref(false)
const analysisResult = ref<AnalysisResultType | null>(null)
const errorMessage = ref('')

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
  activeTab.value = 'upload'

  try {
    const { analyzeFiles } = await import('./api')
    const result = await analyzeFiles(uploadedFiles.value, sourcePath.value)
    analysisResult.value = result
    activeTab.value = 'result'
  } catch (err: any) {
    errorMessage.value = err.message || '分析过程中发生错误'
  } finally {
    isAnalyzing.value = false
  }
}

function clearResults() {
  analysisResult.value = null
  uploadedFiles.value = []
  sourcePath.value = ''
  errorMessage.value = ''
  activeTab.value = 'upload'
}
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Header -->
    <header class="bg-white shadow-sm border-b border-gray-200">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 bg-primary-600 rounded-lg flex items-center justify-center">
              <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h1 class="text-xl font-bold text-gray-900">PROF Analyzer</h1>
              <p class="text-sm text-gray-500">智能性能分析工具</p>
            </div>
          </div>
          <button
            v-if="hasResult"
            @click="clearResults"
            class="px-4 py-2 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
          >
            新建分析
          </button>
        </div>
      </div>
    </header>

    <!-- Navigation Tabs -->
    <nav class="bg-white border-b border-gray-200">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex space-x-8">
          <button
            @click="activeTab = 'upload'"
            :class="[
              'py-4 px-1 border-b-2 text-sm font-medium transition-colors',
              activeTab === 'upload'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            ]"
          >
            上传文件
          </button>
          <button
            @click="activeTab = 'result'"
            :class="[
              'py-4 px-1 border-b-2 text-sm font-medium transition-colors',
              activeTab === 'result'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            ]"
            :disabled="!hasResult"
            :style="{ opacity: hasResult ? 1 : 0.5, cursor: hasResult ? 'pointer' : 'not-allowed' }"
          >
            分析结果
          </button>
          <button
            @click="activeTab = 'charts'"
            :class="[
              'py-4 px-1 border-b-2 text-sm font-medium transition-colors',
              activeTab === 'charts'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            ]"
            :disabled="!hasResult"
            :style="{ opacity: hasResult ? 1 : 0.5, cursor: hasResult ? 'pointer' : 'not-allowed' }"
          >
            可视化
          </button>
          <button
            @click="activeTab = 'export'"
            :class="[
              'py-4 px-1 border-b-2 text-sm font-medium transition-colors',
              activeTab === 'export'
                ? 'border-primary-500 text-primary-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            ]"
            :disabled="!hasResult"
            :style="{ opacity: hasResult ? 1 : 0.5, cursor: hasResult ? 'pointer' : 'not-allowed' }"
          >
            导出PDF
          </button>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Error Message -->
      <div v-if="errorMessage" class="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
        <div class="flex">
          <svg class="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="ml-3 text-sm text-red-700">{{ errorMessage }}</p>
        </div>
      </div>

      <!-- Upload Tab -->
      <div v-show="activeTab === 'upload'">
        <FileUpload
          :files="uploadedFiles"
          :source-path="sourcePath"
          :is-analyzing="isAnalyzing"
          @files-selected="handleFilesSelected"
          @source-path-change="handleSourcePathChange"
          @analyze="startAnalysis"
        />
      </div>

      <!-- Result Tab -->
      <div v-show="activeTab === 'result' && hasResult">
        <AnalysisResult :result="analysisResult!" />
      </div>

      <!-- Charts Tab -->
      <div v-show="activeTab === 'charts' && hasResult">
        <Charts :result="analysisResult!" />
      </div>

      <!-- Export Tab -->
      <div v-show="activeTab === 'export' && hasResult">
        <PdfExport :result="analysisResult!" />
      </div>
    </main>

    <!-- Analyzing Overlay -->
    <div
      v-if="isAnalyzing"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <div class="bg-white rounded-xl shadow-xl p-8 max-w-md w-full mx-4 text-center">
        <div class="animate-spin w-16 h-16 mx-auto mb-4 border-4 border-primary-200 border-t-primary-600 rounded-full"></div>
        <h3 class="text-lg font-semibold text-gray-900 mb-2">AI 分析中...</h3>
        <p class="text-sm text-gray-500">正在分析 PROF 文件，请稍候</p>
      </div>
    </div>
  </div>
</template>
