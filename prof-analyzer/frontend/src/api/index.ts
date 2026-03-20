import axios from 'axios'
import type { AnalysisResult as AnalysisResultType, UploadedFile } from '../types'

const api = axios.create({
  baseURL: '/api',
  timeout: 300000,
})

export interface AnalyzeResponse {
  success: boolean
  data?: AnalysisResultType
  error?: string
  result_url?: string
  result_path?: string
}

export async function analyzeFiles(
  files: UploadedFile[],
  sourcePath: string,
  model?: string,
  onProgress?: (pct: number) => void
): Promise<{ result: AnalysisResultType; resultUrl?: string; resultPath?: string }> {
  return new Promise((resolve, reject) => {
    const formData = new FormData()
    files.forEach((f) => formData.append('files', f.file))
    if (sourcePath) formData.append('source_path', sourcePath)
    if (model) formData.append('model', model)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/analyze')
    xhr.responseType = 'json'

    if (onProgress) {
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          onProgress(Math.round((e.loaded / e.total) * 80))
        }
      }
    }

    xhr.onload = () => {
      if (onProgress) onProgress(100)
      const resp = xhr.response as AnalyzeResponse
      if (!resp.success || !resp.data) {
        reject(new Error(resp.error || 'Analysis failed'))
      } else {
        resolve({
          result: resp.data,
          resultUrl: resp.result_url,
          resultPath: resp.result_path,
        })
      }
    }

    xhr.onerror = () => reject(new Error('Network error'))
    xhr.onabort = () => reject(new Error('Upload aborted'))
    xhr.send(formData)
  })
}

export interface PprofResponse {
  success?: boolean
  path?: string
  url?: string
  message?: string
  error?: string
}

export async function generatePprofImage(file: File): Promise<PprofResponse> {
  const formData = new FormData()
  formData.append('file', file)
  const response = await api.post<PprofResponse>('/pprof/image', formData)
  return response.data || {}
}

export async function saveResultJSON(result: AnalysisResultType): Promise<string> {
  const response = await api.post<{ success: boolean; result_path?: string; error?: string }>('/save-result', result)
  if (!response.data?.success) {
    throw new Error(response.data?.error || 'Failed to save result')
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
