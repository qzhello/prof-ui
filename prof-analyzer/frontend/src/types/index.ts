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
  result_url?: string
  result_path?: string
  path?: string
  url?: string
  message?: string
}
