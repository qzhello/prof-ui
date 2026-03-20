import axios from 'axios'
import type { AnalysisResult as AnalysisResultType, UploadedFile, APIResponse } from '../types'

const api = axios.create({
  baseURL: '/api',
  timeout: 300000,
})

export interface AnalyzeResponse extends APIResponse {
  result_url?: string
  result_path?: string
}

export async function analyzeFiles(
  files: UploadedFile[],
  sourcePath: string,
  model?: string
): Promise<{ result: AnalysisResultType; resultUrl?: string; resultPath?: string }> {
  const formData = new FormData()

  files.forEach((f) => {
    formData.append('files', f.file)
  })

  if (sourcePath) {
    formData.append('source_path', sourcePath)
  }

  if (model) {
    formData.append('model', model)
  }

  const response = await api.post<AnalyzeResponse>('/analyze', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })

  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.error || 'Analysis failed')
  }

  return {
    result: response.data.data,
    resultUrl: response.data.result_url,
    resultPath: response.data.result_path,
  }
}

export interface PprofResponse extends APIResponse {
  path?: string
  url?: string
  message?: string
}

export async function generatePprofImage(file: File): Promise<PprofResponse> {
  const formData = new FormData()
  formData.append('file', file)

  const response = await api.post<PprofResponse>('/pprof/image', formData)
  return response.data
}

export async function saveResultJSON(result: AnalysisResultType): Promise<string> {
  const response = await api.post<{ success: boolean; result_path?: string; error?: string }>('/save-result', result)
  if (!response.data.success) {
    throw new Error(response.data.error || 'Failed to save result')
  }
  return response.data.result_path || ''
}

export async function healthCheck(): Promise<boolean> {
  try {
    const response = await api.get('/health')
    return response.data.status === 'ok'
  } catch {
    return false
  }
}

export { api }
