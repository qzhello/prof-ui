<script setup lang="ts">
import { ref, computed } from 'vue'
import type { UploadedFile } from '../types'

const props = defineProps<{
  files: UploadedFile[]
  sourcePath: string
  isStreaming: boolean
}>()

const emit = defineEmits<{
  (e: 'files-selected', files: UploadedFile[]): void
  (e: 'source-path-change', path: string): void
  (e: 'stream-analyze'): void
  (e: 'generate-pprof'): void
}>()

const isDragging = ref(false)
const showSourcePath = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

const hasFiles = computed(() => props.files.length > 0)

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  isDragging.value = true
}

function handleDragLeave() {
  isDragging.value = false
}

function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (files) {
    processFiles(Array.from(files))
  }
}

function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement
  if (target.files) {
    processFiles(Array.from(target.files))
  }
}

function processFiles(fileList: File[]) {
  const validExtensions = ['.prof', '.pprof', '.json', '.log', '.txt', '.zip']
  const newFiles: UploadedFile[] = []

  for (const file of fileList) {
    const ext = '.' + file.name.split('.').pop()?.toLowerCase()
    if (validExtensions.includes(ext) || file.name.match(/\.(prof|pprof|json|log|txt|zip)$/i)) {
      newFiles.push({
        name: file.name,
        size: file.size,
        type: file.type || ext,
        file: file
      })
    }
  }

  const existingNames = new Set(props.files.map(f => f.name))
  const uniqueNewFiles = newFiles.filter(f => !existingNames.has(f.name))
  emit('files-selected', [...props.files, ...uniqueNewFiles])
}

function removeFile(index: number) {
  const newFiles = [...props.files]
  newFiles.splice(index, 1)
  emit('files-selected', newFiles)
}

function openFilePicker() {
  fileInputRef.value?.click()
}

function toggleSourcePath() {
  showSourcePath.value = !showSourcePath.value
}

function updateSourcePath(e: Event) {
  const target = e.target as HTMLInputElement
  emit('source-path-change', target.value)
}
</script>

<template>
  <div class="space-y-6">
    <!-- File Upload Area -->
    <div
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
      @click="openFilePicker"
      :class="[
        'relative border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-all duration-200',
        isDragging
          ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
          : hasFiles
            ? 'border-gray-300 bg-gray-50 dark:bg-gray-900 dark:bg-gray-900 hover:border-primary-400'
            : 'border-gray-300 bg-gray-50 dark:bg-gray-900 dark:bg-gray-900 hover:border-primary-400'
      ]"
    >
      <input
        ref="fileInputRef"
        type="file"
        multiple
        accept=".prof,.pprof,.json,.log,.txt,.zip"
        @change="handleFileSelect"
        class="hidden"
      />

      <div v-if="!hasFiles" class="space-y-4">
        <div class="w-16 h-16 mx-auto bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center">
          <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
        </div>
        <div>
          <p class="text-lg font-medium text-gray-700 dark:text-gray-200">拖拽 PROF 文件到此处</p>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">或点击选择文件</p>
          <p class="text-xs text-gray-400 mt-2">支持 .prof, .pprof, .json, .log, .txt, .zip 格式</p>
        </div>
      </div>

      <div v-else class="space-y-3">
        <div class="w-12 h-12 mx-auto bg-primary-100 rounded-full flex items-center justify-center">
          <svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <p class="text-primary-600 dark:text-primary-400 font-medium">点击添加更多文件</p>
      </div>
    </div>

    <!-- Selected Files List -->
    <div v-if="hasFiles" class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-200 bg-gray-50 dark:bg-gray-900">
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-200">已选择 {{ files.length }} 个文件</h3>
      </div>
      <ul class="divide-y divide-gray-100">
        <li
          v-for="(file, index) in files"
          :key="file.name"
          class="px-4 py-3 flex items-center justify-between hover:bg-gray-50 dark:bg-gray-900 transition-colors"
        >
          <div class="flex items-center space-x-3 min-w-0">
            <div class="w-10 h-10 bg-gray-100 dark:bg-gray-700 rounded-lg flex items-center justify-center flex-shrink-0">
              <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div class="min-w-0">
              <p class="text-sm font-medium text-gray-900 dark:text-white truncate">{{ file.name }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ formatFileSize(file.size) }}</p>
            </div>
          </div>
          <button
            @click.stop="removeFile(index)"
            class="ml-4 p-2 text-gray-400 hover:text-red-500 transition-colors"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </li>
      </ul>
    </div>

    <!-- Source Path (Optional) -->
    <div class="bg-white rounded-xl shadow-sm border border-gray-200">
      <button
        @click="toggleSourcePath"
        class="w-full px-4 py-3 flex items-center justify-between hover:bg-gray-50 dark:bg-gray-900 transition-colors"
      >
        <div class="flex items-center space-x-3">
          <div class="w-10 h-10 bg-gray-100 dark:bg-gray-700 rounded-lg flex items-center justify-center">
            <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </div>
          <div class="text-left">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">本地源码路径</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">可选，用于更精准的分析</p>
          </div>
        </div>
        <svg
          :class="['w-5 h-5 text-gray-400 transition-transform', showSourcePath ? 'rotate-180' : '']"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      <div v-show="showSourcePath" class="px-4 pb-4">
        <input
          type="text"
          :value="sourcePath"
          @input="updateSourcePath"
          placeholder="/path/to/your/source/code"
          class="w-full px-4 py-3 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
        <p class="mt-2 text-xs text-gray-400">指定本地源码路径可以帮助 AI 更准确地定位问题</p>
      </div>
    </div>

    <!-- Action Buttons -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <button
        @click="$emit('stream-analyze')"
        :disabled="!hasFiles || isStreaming"
        :class="[
          'py-3 px-4 rounded-xl text-sm font-semibold transition-all duration-200 flex items-center justify-center space-x-2',
          hasFiles && !isStreaming
            ? 'bg-primary-600 text-white hover:bg-primary-700 shadow-sm hover:shadow-md'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-400 cursor-not-allowed'
        ]"
      >
        <span v-if="isStreaming" class="flex items-center space-x-2">
          <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span>分析中...</span>
        </span>
        <span v-else class="flex items-center space-x-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          <span>开始分析</span>
        </span>
      </button>
      <button
        @click="$emit('generate-pprof')"
        :disabled="!hasFiles || isStreaming"
        :class="[
          'py-3 px-4 rounded-xl text-sm font-semibold transition-all duration-200 flex items-center justify-center space-x-2',
          hasFiles && !isStreaming
            ? 'bg-orange-500 text-white hover:bg-orange-600 shadow-sm hover:shadow-md'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-400 cursor-not-allowed'
        ]"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>生成 Pprof 图片</span>
      </button>
    </div>
  </div>
</template>
