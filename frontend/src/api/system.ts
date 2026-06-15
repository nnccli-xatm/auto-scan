import client from './client'
import type { SystemStatus } from './types'

export const systemApi = {
  getStatus() {
    return client.get('/system/status')
  },

  getLogs(params?: {
    page?: number
    page_size?: number
    level?: string
  }) {
    return client.get('/system/logs', { params })
  },

  getConfig() {
    return client.get('/system/config')
  },

  updateConfig(data: Record<string, unknown>) {
    return client.put('/system/config', data)
  },
}

export type { SystemStatus }
