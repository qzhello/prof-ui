<script setup lang="ts">
defineProps<{
  pprofImageUrl?: string
}>()
</script>

<template>
  <div class="space-y-6">
    <!-- Pprof Image -->
    <div v-if="pprofImageUrl" class="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-lg font-semibold text-gray-700 flex items-center space-x-2">
          <svg class="w-5 h-5 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span>Pprof 火焰图</span>
        </h3>
        <a
          :href="pprofImageUrl"
          download="pprof.png"
          class="text-sm text-primary-600 hover:text-primary-800 flex items-center space-x-1"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>下载</span>
        </a>
      </div>
      <div class="flex justify-center bg-gray-50 rounded-lg p-4">
        <img :src="pprofImageUrl" alt="Pprof Flame Graph" class="max-w-full h-auto rounded shadow-sm" />
      </div>
    </div>

    <!-- No Image Placeholder -->
    <div v-else class="bg-white rounded-2xl p-12 shadow-sm border border-gray-100 text-center">
      <svg class="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
      <p class="text-gray-500">点击「生成 Pprof 图片」后可在此查看火焰图</p>
    </div>
  </div>
</template>
