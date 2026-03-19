export interface ChainLink {
  from: string
  to: string
  description: string
  time_cost: string
}

export interface Metrics {
  total_time: string
  memory_usage: string
  cpu_usage: string
  goroutines: number
  gc_count: number
}

export interface Hotspot {
  function: string
  location: string
  time_cost: string
  percentage: number
  calls: number
}

export interface CallNode {
  name: string
  time_cost: string
  calls: number
  children?: CallNode[]
}

export interface ChartData {
  type: string
  name: string
  data: {
    labels?: string[]
    values?: number[]
    [key: string]: any
  }
}

export interface AnalysisResult {
  summary: string
  chain: ChainLink[]
  root_cause: string
  solutions: string[]
  metrics: Metrics
  charts: ChartData[]
  hotspots: Hotspot[]
  call_tree: CallNode[]
}

export interface UploadedFile {
  name: string
  size: number
  type: string
  file: File
}

export interface APIResponse<T = any> {
  success: boolean
  data?: T
  error?: string
}
