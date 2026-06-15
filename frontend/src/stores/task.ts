import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { taskApi } from '@/api/task'
import type { ScanTask, CreateTaskRequest } from '@/api/types'
import { ElMessage } from 'element-plus'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<ScanTask[]>([])
  const loading = ref(false)
  const total = ref(0)

  const runningTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'running')
  )

  const pendingTasks = computed(() =>
    tasks.value.filter((t) => t.status === 'pending')
  )

  async function fetchTasks() {
    loading.value = true
    try {
      const res = await taskApi.list()
      tasks.value = res.data.data.list || []
      total.value = res.data.data.pagination?.total || 0
    } catch (error) {
      console.error('获取任务列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  async function createTask(data: CreateTaskRequest) {
    try {
      const res = await taskApi.create(data)
      tasks.value.unshift(res.data.data)
      ElMessage.success('任务创建成功')
      return res.data.data
    } catch (error) {
      console.error('创建任务失败:', error)
      throw error
    }
  }

  async function cancelTask(id: string) {
    try {
      await taskApi.cancel(id)
      const task = tasks.value.find((t) => t.id === id)
      if (task) {
        task.status = 'cancelled'
      }
      ElMessage.success('任务已取消')
    } catch (error) {
      console.error('取消任务失败:', error)
      throw error
    }
  }

  function updateTaskProgress(taskId: string, progress: number, scannedPages: number) {
    const task = tasks.value.find((t) => t.id === taskId)
    if (task) {
      task.progress = progress
      task.scanned_pages = scannedPages
    }
  }

  return {
    tasks,
    loading,
    total,
    runningTasks,
    pendingTasks,
    fetchTasks,
    createTask,
    cancelTask,
    updateTaskProgress,
  }
})
