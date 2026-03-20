<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'
import type { AnalysisResult } from '../types'

const props = defineProps<{
  result: AnalysisResult
}>()

const pieChartRef = ref<HTMLDivElement | null>(null)
const barChartRef = ref<HTMLDivElement | null>(null)
const hotspotChartRef = ref<HTMLDivElement | null>(null)
const treeChartRef = ref<HTMLDivElement | null>(null)

let pieChart: echarts.ECharts | null = null
let barChart: echarts.ECharts | null = null
let hotspotChart: echarts.ECharts | null = null
let treeChart: echarts.ECharts | null = null

function getPieChartOption() {
  const chartData = props.result.charts.find(c => c.type === 'pie')
  if (!chartData) {
    return {
      title: { text: '时间消耗分布', left: 'center' },
      tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        data: [
          { value: 45, name: '数据库查询' },
          { value: 20, name: 'JSON序列化' },
          { value: 15, name: '网络传输' },
          { value: 12, name: '业务逻辑' },
          { value: 8, name: '其他' }
        ]
      }]
    }
  }

  const labels = chartData.data.labels || []
  const values = chartData.data.values || []

  return {
    title: {
      text: chartData.name,
      left: 'center',
      textStyle: { fontSize: 16, fontWeight: 'normal' }
    },
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center'
    },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      center: ['40%', '50%'],
      data: labels.map((label, i) => ({
        name: label,
        value: values[i]
      }))
    }]
  }
}

function getBarChartOption() {
  const chartData = props.result.charts.find(c => c.type === 'bar')
  if (!chartData) {
    return {
      title: { text: '函数调用次数', left: 'center' },
      tooltip: { trigger: 'axis' },
      xAxis: {
        type: 'category',
        data: ['json.Marshal', 'db.Query', 'redis.Get', 'http.Handler', 'log.Printf'],
        axisLabel: { rotate: 30, fontSize: 10 }
      },
      yAxis: { type: 'value' },
      series: [{
        type: 'bar',
        data: [5000, 1000, 2000, 100, 500],
        itemStyle: { color: '#3b82f6' }
      }]
    }
  }

  const labels = chartData.data.labels || []
  const values = chartData.data.values || []

  return {
    title: {
      text: chartData.name,
      left: 'center',
      textStyle: { fontSize: 16, fontWeight: 'normal' }
    },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '15%', containLabel: true },
    xAxis: {
      type: 'category',
      data: labels,
      axisLabel: { rotate: 30, fontSize: 10 }
    },
    yAxis: { type: 'value' },
    series: [{
      type: 'bar',
      data: values,
      itemStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: '#60a5fa' },
          { offset: 1, color: '#2563eb' }
        ])
      },
      barWidth: '50%'
    }]
  }
}

function getHotspotChartOption() {
  const hotspots = props.result.hotspots.slice(0, 5)

  return {
    title: {
      text: 'Top 5 性能热点',
      left: 'center',
      textStyle: { fontSize: 16, fontWeight: 'normal' }
    },
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'value',
      name: '占比 (%)'
    },
    yAxis: {
      type: 'category',
      data: hotspots.map(h => h.function).reverse(),
      axisLabel: { fontSize: 11 }
    },
    series: [{
      type: 'bar',
      data: hotspots.map(h => h.percentage).reverse(),
      itemStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
          { offset: 0, color: '#f97316' },
          { offset: 1, color: '#fbbf24' }
        ])
      },
      barWidth: '60%'
    }]
  }
}

function getTreeChartOption() {
  function buildTree(nodes: any[]): any {
    if (!nodes || nodes.length === 0) {
      return {
        name: 'main',
        children: []
      }
    }

    return {
      name: nodes[0].name,
      children: (nodes[0].children || []).map((child: any) => buildTree([child]))
    }
  }

  const treeData = buildTree(props.result.call_tree)

  return {
    title: {
      text: '调用树',
      left: 'center',
      textStyle: { fontSize: 16, fontWeight: 'normal' }
    },
    tooltip: { trigger: 'item', formatter: '{b}' },
    series: [{
      type: 'tree',
      data: [treeData],
      orient: 'TB',
      symbol: 'rect',
      symbolSize: [80, 30],
      itemStyle: {
        color: '#dbeafe',
        borderColor: '#3b82f6',
        borderWidth: 1
      },
      label: {
        fontSize: 11,
        position: 'inside'
      },
      leaves: {
        itemStyle: {
          color: '#bbf7d0',
          borderColor: '#22c55e'
        }
      },
      expandAndCollapse: true,
      initialTreeDepth: 2
    }]
  }
}

function initCharts() {
  if (pieChartRef.value) {
    pieChart = echarts.init(pieChartRef.value)
    pieChart.setOption(getPieChartOption())
  }

  if (barChartRef.value) {
    barChart = echarts.init(barChartRef.value)
    barChart.setOption(getBarChartOption())
  }

  if (hotspotChartRef.value) {
    hotspotChart = echarts.init(hotspotChartRef.value)
    hotspotChart.setOption(getHotspotChartOption())
  }

  if (treeChartRef.value) {
    treeChart = echarts.init(treeChartRef.value)
    treeChart.setOption(getTreeChartOption())
  }
}

function handleResize() {
  pieChart?.resize()
  barChart?.resize()
  hotspotChart?.resize()
  treeChart?.resize()
}

onMounted(() => {
  initCharts()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  pieChart?.dispose()
  barChart?.dispose()
  hotspotChart?.dispose()
  treeChart?.dispose()
})

watch(() => props.result, () => {
  pieChart?.setOption(getPieChartOption())
  barChart?.setOption(getBarChartOption())
  hotspotChart?.setOption(getHotspotChartOption())
  treeChart?.setOption(getTreeChartOption())
}, { deep: true })
</script>

<template>
  <div class="space-y-6">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Pie Chart -->
      <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
        <div ref="pieChartRef" class="w-full h-80"></div>
      </div>

      <!-- Bar Chart -->
      <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
        <div ref="barChartRef" class="w-full h-80"></div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Hotspot Chart -->
      <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
        <div ref="hotspotChartRef" class="w-full h-80"></div>
      </div>

      <!-- Tree Chart -->
      <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
        <div ref="treeChartRef" class="w-full h-80"></div>
      </div>
    </div>
  </div>
</template>
