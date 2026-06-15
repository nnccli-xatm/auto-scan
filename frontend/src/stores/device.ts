import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { deviceApi } from '@/api/device'
import type { Device, CreateDeviceRequest } from '@/api/types'
import { ElMessage } from 'element-plus'

export const useDeviceStore = defineStore('device', () => {
  const devices = ref<Device[]>([])
  const loading = ref(false)
  const total = ref(0)

  const onlineDevices = computed(() =>
    devices.value.filter((d) => d.status === 'online')
  )

  const onlineCount = computed(() => onlineDevices.value.length)

  async function fetchDevices() {
    loading.value = true
    try {
      const res = await deviceApi.list()
      devices.value = res.data.data.list || []
      total.value = res.data.data.pagination?.total || 0
    } catch (error) {
      console.error('获取设备列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  async function addDevice(data: CreateDeviceRequest) {
    try {
      const res = await deviceApi.create(data)
      devices.value.push(res.data.data)
      ElMessage.success('设备添加成功')
      return res.data.data
    } catch (error) {
      console.error('添加设备失败:', error)
      throw error
    }
  }

  async function removeDevice(id: string) {
    try {
      await deviceApi.delete(id)
      devices.value = devices.value.filter((d) => d.id !== id)
      ElMessage.success('设备删除成功')
    } catch (error) {
      console.error('删除设备失败:', error)
      throw error
    }
  }

  async function discoverDevices() {
    try {
      const res = await deviceApi.discover()
      ElMessage.success(`发现 ${res.data.data.found || 0} 台设备`)
      await fetchDevices()
      return res.data.data.devices || []
    } catch (error) {
      console.error('发现设备失败:', error)
      throw error
    }
  }

  function updateDeviceStatus(deviceId: string, status: string) {
    const device = devices.value.find((d) => d.id === deviceId)
    if (device) {
      device.status = status as Device['status']
    }
  }

  return {
    devices,
    loading,
    total,
    onlineDevices,
    onlineCount,
    fetchDevices,
    addDevice,
    removeDevice,
    discoverDevices,
    updateDeviceStatus,
  }
})
