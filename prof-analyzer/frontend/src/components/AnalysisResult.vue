<script setup lang="ts">
import type { AnalysisResult } from '../types'

defineProps<{
  result: AnalysisResult
}>()

function formatPercentage(value: number): string {
  return value.toFixed(1) + '%'
}
</script>

<template>
  <div class="space-y-6">
    <!-- Summary Card -->
    <div class="bg-gradient-to-br from-primary-500 to-primary-700 rounded-2xl p-6 text-white shadow-lg">
      <h2 class="text-lg font-semibold mb-3 flex items-center">
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        分析摘要
      </h2>
      <p class="text-primary-100 text-sm leading-relaxed">{{ result.summary }}</p>
    </div>

    <!-- Metrics Grid -->
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
      <div class="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">总耗时</p>
        <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.metrics.total_time }}</p>
      </div>
      <div class="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">内存使用</p>
        <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.metrics.memory_usage }}</p>
      </div>
      <div class="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">CPU 使用</p>
        <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.metrics.cpu_usage }}</p>
      </div>
      <div class="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">Goroutines</p>
        <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.metrics.goroutines }}</p>
      </div>
      <div class="bg-white rounded-xl p-4 shadow-sm border border-gray-100">
        <p class="text-xs text-gray-500 dark:text-gray-400 mb-1">GC 次数</p>
        <p class="text-lg font-bold text-gray-900 dark:text-white">{{ result.metrics.gc_count }}</p>
      </div>
    </div>

    <!-- Root Cause -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
        <span class="w-8 h-8 bg-red-100 rounded-lg flex items-center justify-center mr-3">
          <svg class="w-5 h-5 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </span>
        问题根因
      </h2>
      <div class="bg-red-50 dark:bg-red-900/20 rounded-xl p-4 border border-red-100 dark:border-red-900/30">
        <p class="text-gray-700 dark:text-gray-300 leading-relaxed">{{ result.root_cause }}</p>
      </div>
    </div>

    <!-- Call Chain -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
        <span class="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center mr-3">
          <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
        </span>
        调用链路
      </h2>
      <div class="space-y-3">
        <div
          v-for="(link, index) in result.chain"
          :key="index"
          class="flex items-center space-x-3 p-3 bg-gray-50 dark:bg-gray-900 rounded-lg"
        >
          <div class="flex-1 flex items-center space-x-2 min-w-0">
            <span class="px-2 py-1 bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300 text-xs font-medium rounded truncate">
              {{ link.from }}
            </span>
            <svg class="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3" />
            </svg>
            <span class="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-xs font-medium rounded truncate">
              {{ link.to }}
            </span>
          </div>
          <span class="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">{{ link.time_cost }}</span>
        </div>
      </div>
    </div>

    <!-- Solutions -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
        <span class="w-8 h-8 bg-green-100 dark:bg-green-900/30 rounded-lg flex items-center justify-center mr-3">
          <svg class="w-5 h-5 text-green-600 dark:text-green-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </span>
        解决建议
      </h2>
      <ul class="space-y-3">
        <li
          v-for="(solution, index) in result.solutions"
          :key="index"
          class="flex items-start space-x-3"
        >
          <span class="w-6 h-6 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-300 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0">
            {{ index + 1 }}
          </span>
          <p class="text-gray-700 dark:text-gray-300 leading-relaxed pt-0.5">{{ solution }}</p>
        </li>
      </ul>
    </div>

    <!-- Hotspots -->
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 shadow-sm border border-gray-100 dark:border-gray-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
        <span class="w-8 h-8 bg-orange-100 dark:bg-orange-900/30 rounded-lg flex items-center justify-center mr-3">
          <svg class="w-5 h-5 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z" />
          </svg>
        </span>
        性能热点
      </h2>
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="border-b border-gray-100">
              <th class="text-left py-3 px-2 text-xs font-medium text-gray-500 dark:text-gray-400">函数</th>
              <th class="text-left py-3 px-2 text-xs font-medium text-gray-500 dark:text-gray-400">位置</th>
              <th class="text-right py-3 px-2 text-xs font-medium text-gray-500 dark:text-gray-400">耗时</th>
              <th class="text-right py-3 px-2 text-xs font-medium text-gray-500 dark:text-gray-400">占比</th>
              <th class="text-right py-3 px-2 text-xs font-medium text-gray-500 dark:text-gray-400">调用次数</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-50">
            <tr v-for="(spot, index) in result.hotspots" :key="index" class="hover:bg-gray-50 dark:bg-gray-900">
              <td class="py-3 px-2">
                <span class="text-sm font-medium text-gray-900 dark:text-white">{{ spot.function }}</span>
              </td>
              <td class="py-3 px-2">
                <code class="text-xs text-gray-500 dark:text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">{{ spot.location }}</code>
              </td>
              <td class="py-3 px-2 text-right">
                <span class="text-sm text-gray-900 dark:text-white">{{ spot.time_cost }}</span>
              </td>
              <td class="py-3 px-2 text-right">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300">
                  {{ formatPercentage(spot.percentage) }}
                </span>
              </td>
              <td class="py-3 px-2 text-right">
                <span class="text-sm text-gray-600 dark:text-gray-300">{{ spot.calls.toLocaleString() }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
