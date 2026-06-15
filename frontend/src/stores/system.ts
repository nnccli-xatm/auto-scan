import { defineStore } from 'pinia'
import { ref } from 'vue'
import { systemApi } from '@/api/system'
import type { SystemStatus } from '@/api/types'

export const useSystemStore = defineStore('system', () => {
  const status = ref<SystemStatus | null>(null)
  const loading = ref(false)

  async function fetchStatus() {
    loading.value = true
    try {
      const res = await systemApi.getStatus()
      status.value = res.data.data
    } catch (error) {
      console.error('获取系统状态失败:', error)
    } finally {
      loading.value = false
    }
  }

  return {
    status,
    loading,
    fetchStatus,
  }
})
