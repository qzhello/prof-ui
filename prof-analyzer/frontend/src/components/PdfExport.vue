<script setup lang="ts">
import { ref } from 'vue'
import type { AnalysisResult } from '../types'

defineProps<{
  result: AnalysisResult
}>()

const isExporting = ref(false)
const exportStatus = ref('')

async function exportToPDF() {
  isExporting.value = true
  exportStatus.value = '正在生成 PDF...'

  try {
    const html2pdf = (await import('html2pdf.js')).default
    const html2canvas = (await import('html2canvas')).default

    // Capture chart images first
    const chartContainers = document.querySelectorAll('.chart-container')
    const chartImages: string[] = []
    for (const container of Array.from(chartContainers)) {
      try {
        const canvas = await html2canvas(container as HTMLElement, {
          scale: 2,
          useCORS: true,
          backgroundColor: '#ffffff'
        })
        chartImages.push(canvas.toDataURL('image/png'))
      } catch {
        chartImages.push('')
      }
    }

    const element = document.getElementById('pdf-content')
    if (!element) {
      throw new Error('PDF content element not found')
    }

    const opt = {
      margin: [10, 10, 10, 10],
      filename: `prof-analysis-${Date.now()}.pdf`,
      image: { type: 'jpeg', quality: 0.98 },
      html2canvas: {
        scale: 2,
        useCORS: true,
        logging: false
      },
      jsPDF: {
        unit: 'mm',
        format: 'a4',
        orientation: 'portrait'
      },
      pagebreak: { mode: ['avoid-all', 'css', 'legacy'] }
    }

    exportStatus.value = '正在渲染...'
    await html2pdf().set(opt).from(element).save()

    // If we captured charts, also save them separately
    if (chartImages.length > 0) {
      chartImages.forEach((img, i) => {
        if (img) {
          const a = document.createElement('a')
          a.href = img
          a.download = `chart-${i + 1}.png`
          a.click()
        }
      })
    }

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

function formatPercentage(value: number): string {
  return value.toFixed(1) + '%'
}
</script>

<template>
  <div class="space-y-6">
    <!-- Preview Card -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-100 bg-gray-50">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">PDF 报告预览</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">点击下方按钮导出完整报告</p>
      </div>

      <!-- Hidden content for PDF generation -->
      <div id="pdf-content" class="hidden">
        <div style="padding: 20mm; font-family: 'Helvetica Neue', Arial, sans-serif; color: #1f2937;">
          <!-- Header -->
          <div style="text-align: center; margin-bottom: 30px; border-bottom: 2px solid #3b82f6; padding-bottom: 20px;">
            <h1 style="font-size: 24px; color: #1e40af; margin: 0 0 10px 0;">PROF Analyzer</h1>
            <p style="font-size: 14px; color: #6b7280; margin: 0;">性能分析报告</p>
            <p style="font-size: 12px; color: #9ca3af; margin: 10px 0 0 0;">
              生成时间: {{ new Date().toLocaleString('zh-CN') }}
            </p>
          </div>

          <!-- Summary -->
          <div style="background: linear-gradient(135deg, #3b82f6, #1d4ed8); color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
            <h2 style="font-size: 16px; margin: 0 0 10px 0;">分析摘要</h2>
            <p style="font-size: 13px; line-height: 1.6; margin: 0;">{{ result.summary }}</p>
          </div>

          <!-- Metrics -->
          <div style="margin-bottom: 20px;">
            <h2 style="font-size: 16px; color: #1f2937; border-bottom: 1px solid #e5e7eb; padding-bottom: 8px;">性能指标</h2>
            <div style="display: grid; grid-template-columns: repeat(5, 1fr); gap: 10px; margin-top: 15px;">
              <div style="background: #f9fafb; padding: 15px; border-radius: 6px; text-align: center;">
                <p style="font-size: 10px; color: #6b7280; margin: 0 0 5px 0;">总耗时</p>
                <p style="font-size: 18px; font-weight: bold; color: #1f2937; margin: 0;">{{ result.metrics.total_time }}</p>
              </div>
              <div style="background: #f9fafb; padding: 15px; border-radius: 6px; text-align: center;">
                <p style="font-size: 10px; color: #6b7280; margin: 0 0 5px 0;">内存</p>
                <p style="font-size: 18px; font-weight: bold; color: #1f2937; margin: 0;">{{ result.metrics.memory_usage }}</p>
              </div>
              <div style="background: #f9fafb; padding: 15px; border-radius: 6px; text-align: center;">
                <p style="font-size: 10px; color: #6b7280; margin: 0 0 5px 0;">CPU</p>
                <p style="font-size: 18px; font-weight: bold; color: #1f2937; margin: 0;">{{ result.metrics.cpu_usage }}</p>
              </div>
              <div style="background: #f9fafb; padding: 15px; border-radius: 6px; text-align: center;">
                <p style="font-size: 10px; color: #6b7280; margin: 0 0 5px 0;">Goroutines</p>
                <p style="font-size: 18px; font-weight: bold; color: #1f2937; margin: 0;">{{ result.metrics.goroutines }}</p>
              </div>
              <div style="background: #f9fafb; padding: 15px; border-radius: 6px; text-align: center;">
                <p style="font-size: 10px; color: #6b7280; margin: 0 0 5px 0;">GC次数</p>
                <p style="font-size: 18px; font-weight: bold; color: #1f2937; margin: 0;">{{ result.metrics.gc_count }}</p>
              </div>
            </div>
          </div>

          <!-- Root Cause -->
          <div style="margin-bottom: 20px;">
            <h2 style="font-size: 16px; color: #1f2937; border-bottom: 1px solid #e5e7eb; padding-bottom: 8px;">问题根因</h2>
            <div style="background: #fef2f2; border: 1px solid #fecaca; padding: 15px; border-radius: 6px; margin-top: 10px;">
              <p style="font-size: 13px; line-height: 1.6; color: #991b1b; margin: 0;">{{ result.root_cause }}</p>
            </div>
          </div>

          <!-- Solutions -->
          <div style="margin-bottom: 20px;">
            <h2 style="font-size: 16px; color: #1f2937; border-bottom: 1px solid #e5e7eb; padding-bottom: 8px;">解决建议</h2>
            <ol style="margin: 10px 0 0 0; padding-left: 20px;">
              <li v-for="(solution, index) in result.solutions" :key="index" style="font-size: 13px; line-height: 1.8; color: #374151;">
                {{ solution }}
              </li>
            </ol>
          </div>

          <!-- Call Chain -->
          <div style="margin-bottom: 20px;">
            <h2 style="font-size: 16px; color: #1f2937; border-bottom: 1px solid #e5e7eb; padding-bottom: 8px;">调用链路</h2>
            <div style="margin-top: 10px;">
              <div v-for="(link, index) in result.chain" :key="index" style="display: flex; align-items: center; margin-bottom: 8px; font-size: 12px;">
                <span style="background: #dbeafe; color: #1d4ed8; padding: 4px 8px; border-radius: 4px;">{{ link.from }}</span>
                <span style="margin: 0 10px; color: #9ca3af;">→</span>
                <span style="background: #dcfce7; color: #166534; padding: 4px 8px; border-radius: 4px;">{{ link.to }}</span>
                <span style="margin-left: auto; color: #6b7280;">{{ link.time_cost }}</span>
              </div>
            </div>
          </div>

          <!-- Hotspots -->
          <div>
            <h2 style="font-size: 16px; color: #1f2937; border-bottom: 1px solid #e5e7eb; padding-bottom: 8px;">性能热点</h2>
            <table style="width: 100%; margin-top: 10px; border-collapse: collapse; font-size: 12px;">
              <thead>
                <tr style="background: #f9fafb;">
                  <th style="padding: 10px; text-align: left; border-bottom: 1px solid #e5e7eb;">函数</th>
                  <th style="padding: 10px; text-align: left; border-bottom: 1px solid #e5e7eb;">位置</th>
                  <th style="padding: 10px; text-align: right; border-bottom: 1px solid #e5e7eb;">耗时</th>
                  <th style="padding: 10px; text-align: right; border-bottom: 1px solid #e5e7eb;">占比</th>
                  <th style="padding: 10px; text-align: right; border-bottom: 1px solid #e5e7eb;">调用次数</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(spot, index) in result.hotspots" :key="index">
                  <td style="padding: 10px; border-bottom: 1px solid #e5e7eb;">{{ spot.function }}</td>
                  <td style="padding: 10px; border-bottom: 1px solid #e5e7eb; color: #6b7280;">{{ spot.location }}</td>
                  <td style="padding: 10px; text-align: right; border-bottom: 1px solid #e5e7eb;">{{ spot.time_cost }}</td>
                  <td style="padding: 10px; text-align: right; border-bottom: 1px solid #e5e7eb; color: #ea580c;">{{ formatPercentage(spot.percentage) }}</td>
                  <td style="padding: 10px; text-align: right; border-bottom: 1px solid #e5e7eb;">{{ spot.calls.toLocaleString() }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Footer -->
          <div style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #e5e7eb; text-align: center;">
            <p style="font-size: 10px; color: #9ca3af; margin: 0;">由 PROF Analyzer 自动生成</p>
          </div>
        </div>
      </div>

      <!-- Export Preview -->
      <div class="p-6">
        <div class="bg-gray-50 dark:bg-gray-900 rounded-xl p-4 mb-6">
          <div class="flex items-center space-x-3 mb-4">
            <div class="w-10 h-10 bg-primary-100 rounded-lg flex items-center justify-center">
              <svg class="w-5 h-5 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">性能分析报告</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">包含分析摘要、指标、根因、解决方案和热点</p>
            </div>
          </div>
          <div class="grid grid-cols-3 gap-3 text-center">
            <div class="bg-white rounded-lg p-3">
              <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.chain.length }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">调用链路</p>
            </div>
            <div class="bg-white rounded-lg p-3">
              <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.solutions.length }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">解决建议</p>
            </div>
            <div class="bg-white rounded-lg p-3">
              <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.hotspots.length }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">性能热点</p>
            </div>
          </div>
        </div>

        <!-- Export Status -->
        <div v-if="exportStatus" class="mb-4 p-3 bg-blue-50 rounded-lg">
          <p class="text-sm text-blue-700">{{ exportStatus }}</p>
        </div>

        <!-- Export Button -->
        <button
          @click="exportToPDF"
          :disabled="isExporting"
          :class="[
            'w-full py-4 px-6 rounded-xl text-base font-semibold transition-all duration-200 flex items-center justify-center space-x-2',
            isExporting
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
