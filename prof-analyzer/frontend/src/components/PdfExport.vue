<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  htmlContent?: string
  markdownContent?: string
  canExport?: boolean
}>()

const isExporting = ref(false)
const exportStatus = ref('')

async function exportToPDF() {
  isExporting.value = true
  exportStatus.value = '正在生成 PDF...'

  try {
    const html2pdf = (await import('html2pdf.js')).default

    if (!props.canExport) {
      throw new Error('分析尚未完成，请等待完整报告生成后再导出')
    }

    if (!props.markdownContent?.trim() || !props.htmlContent?.trim()) {
      throw new Error('分析内容未找到，请先执行分析')
    }

    const opt = {
      margin: [10, 10, 10, 10],
      filename: `prof-analysis-${Date.now()}.pdf`,
      image: { type: 'jpeg', quality: 0.98 },
      html2canvas: {
        scale: 2,
        useCORS: true,
        logging: false,
        backgroundColor: '#ffffff'
      },
      jsPDF: {
        unit: 'mm',
        format: 'a4',
        orientation: 'portrait'
      },
      pagebreak: { mode: ['avoid-all', 'css', 'legacy'] }
    }

    exportStatus.value = '正在渲染...'

    // Create a wrapper with header for the PDF
    const wrapper = document.createElement('div')
    wrapper.style.padding = '20mm'
    wrapper.style.fontFamily = "'Helvetica Neue', Arial, sans-serif"
    wrapper.style.color = '#1f2937'
    wrapper.style.backgroundColor = '#ffffff'

    // Add header
    const header = document.createElement('div')
    header.style.textAlign = 'center'
    header.style.marginBottom = '30px'
    header.style.borderBottom = '2px solid #3b82f6'
    header.style.paddingBottom = '20px'
    header.innerHTML = `
      <h1 style="font-size: 24px; color: #1e40af; margin: 0 0 10px 0;">PROF Analyzer</h1>
      <p style="font-size: 14px; color: #6b7280; margin: 0;">性能分析报告</p>
      <p style="font-size: 12px; color: #9ca3af; margin: 10px 0 0 0;">
        生成时间: ${new Date().toLocaleString('zh-CN')}
      </p>
    `
    wrapper.appendChild(header)

    const contentClone = document.createElement('div')
    contentClone.className = 'markdown-body'
    contentClone.innerHTML = props.htmlContent
    contentClone.style.backgroundColor = '#ffffff'
    contentClone.style.color = '#374151'
    wrapper.appendChild(contentClone)

    // Add footer
    const footer = document.createElement('div')
    footer.style.marginTop = '40px'
    footer.style.paddingTop = '20px'
    footer.style.borderTop = '1px solid #e5e7eb'
    footer.style.textAlign = 'center'
    footer.innerHTML = `<p style="font-size: 10px; color: #9ca3af; margin: 0;">由 PROF Analyzer 自动生成</p>`
    wrapper.appendChild(footer)

    document.body.appendChild(wrapper)
    await html2pdf().set(opt).from(wrapper).save()
    document.body.removeChild(wrapper)

    exportStatus.value = 'PDF 已下载'
    setTimeout(() => {
      exportStatus.value = ''
    }, 3000)
  } catch (err: any) {
    exportStatus.value = '导出失败: ' + (err.message || '未知错误')
    setTimeout(() => {
      exportStatus.value = ''
    }, 5000)
  } finally {
    isExporting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-100 bg-gray-50">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">导出报告</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">将分析结果导出为 PDF 文件</p>
      </div>

      <div class="p-6">
        <div class="bg-gray-50 dark:bg-gray-900 rounded-xl p-4 mb-6">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 bg-primary-100 rounded-lg flex items-center justify-center">
              <svg class="w-5 h-5 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">PDF 报告</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">包含完整的分析结果和 Markdown 内容</p>
            </div>
          </div>
        </div>

        <div v-if="!canExport" class="mb-4 p-3 bg-amber-50 rounded-lg">
          <p class="text-sm text-amber-700">请先等待分析完成，再导出完整 PDF 报告。</p>
        </div>

        <!-- Export Status -->
        <div v-if="exportStatus" class="mb-4 p-3 bg-blue-50 rounded-lg">
          <p class="text-sm text-blue-700">{{ exportStatus }}</p>
        </div>

        <!-- Export Button -->
        <button
          @click="exportToPDF"
          :disabled="isExporting || !canExport"
          :class="[
            'w-full py-4 px-6 rounded-xl text-base font-semibold transition-all duration-200 flex items-center justify-center space-x-2',
            isExporting || !canExport
              ? 'bg-gray-200 text-gray-400 cursor-not-allowed'
              : 'bg-primary-600 text-white hover:bg-primary-700 shadow-lg shadow-primary-200 hover:shadow-xl hover:shadow-primary-300'
          ]"
        >
          <svg v-if="!isExporting" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <svg v-else class="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span>{{ isExporting ? '导出中...' : '导出 PDF 报告' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
