import client from './client'
import type { ScanFile } from './types'

export const fileApi = {
  list(params?: {
    page?: number
    page_size?: number
    device_id?: string
    task_id?: string
  }) {
    return client.get('/files', { params })
  },

  get(id: string) {
    return client.get(`/files/${id}`)
  },

  delete(id: string) {
    return client.delete(`/files/${id}`)
  },

  getDownloadUrl(id: string): string {
    const baseURL = client.defaults.baseURL || '/api/v1'
    return `${baseURL}/files/${id}/download`
  },

  download(id: string) {
    return client.get(`/files/${id}/download`, { responseType: 'blob' })
  },

  batchDownload(fileIds: string[]) {
    return client.post('/files/batch/download', { file_ids: fileIds }, { responseType: 'blob' })
  },

  batchDelete(fileIds: string[]) {
    return client.post('/files/batch/delete', { file_ids: fileIds })
  },
}

export type { ScanFile }
