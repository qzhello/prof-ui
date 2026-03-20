import axios from 'axios'
import type { UploadedFile } from '../types'

const api = axios.create({
  baseURL: '/api',
  timeout: 300000,
})

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

export async function healthCheck(): Promise<boolean> {
  try {
    const response = await api.get('/health')
    return response.data.status === 'ok'
  } catch {
    return false
  }
}

export { api }
