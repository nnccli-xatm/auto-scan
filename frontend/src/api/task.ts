import client from './client'
import type { ScanTask, CreateTaskRequest } from './types'

export const taskApi = {
  list(params?: {
    page?: number
    page_size?: number
    status?: string
    device_id?: string
  }) {
    return client.get('/tasks', { params })
  },

  get(id: string) {
    return client.get(`/tasks/${id}`)
  },

  create(data: CreateTaskRequest) {
    return client.post('/tasks', data)
  },

  cancel(id: string) {
    return client.delete(`/tasks/${id}`)
  },

  getProgress(id: string) {
    return client.get(`/tasks/${id}/progress`)
  },
}

export type { ScanTask, CreateTaskRequest }
