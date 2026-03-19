import axios from 'axios'
import type { AnalysisResult, UploadedFile, APIResponse } from '../types'

const api = axios.create({
  baseURL: '/api',
  timeout: 300000,
})

export async function analyzeFiles(
  files: UploadedFile[],
  sourcePath: string,
  model?: string
): Promise<AnalysisResult> {
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

  const response = await api.post<APIResponse<AnalysisResult>>('/analyze', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })

  if (!response.data.success || !response.data.data) {
    throw new Error(response.data.error || 'Analysis failed')
  }

  return response.data.data
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
