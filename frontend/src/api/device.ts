import client from './client'
import type { Device, CreateDeviceRequest, DeviceStatus, PaginationData } from './types'

export const deviceApi = {
  // 获取设备列表
  list(params?: {
    page?: number
    page_size?: number
    status?: string
    vendor?: string
  }) {
    return client.get('/devices', { params })
  },

  // 获取设备详情
  get(id: string) {
    return client.get(`/devices/${id}`)
  },

  // 添加设备
  create(data: CreateDeviceRequest) {
    return client.post('/devices', data)
  },

  // 更新设备
  update(id: string, data: Partial<CreateDeviceRequest>) {
    return client.put(`/devices/${id}`, data)
  },

  // 删除设备
  delete(id: string) {
    return client.delete(`/devices/${id}`)
  },

  // 发现设备
  discover() {
    return client.post('/devices/discover')
  },

  // 获取设备状态
  getStatus(id: string) {
    return client.get(`/devices/${id}/status`)
  },

  // 连接设备
  connect(id: string) {
    return client.post(`/devices/${id}/connect`)
  },

  // 断开设备
  disconnect(id: string) {
    return client.post(`/devices/${id}/disconnect`)
  },
}

export type { Device, CreateDeviceRequest, DeviceStatus, PaginationData }
