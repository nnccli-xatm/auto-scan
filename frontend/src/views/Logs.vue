<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-select v-model="levelFilter" placeholder="日志级别" clearable style="width: 120px">
              <el-option label="Debug" value="debug" />
              <el-option label="Info" value="info" />
              <el-option label="Warning" value="warning" />
              <el-option label="Error" value="error" />
            </el-select>
            <el-date-picker
              v-model="dateRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              style="width: 380px"
            />
          </div>
          <el-button :icon="Refresh" @click="fetchLogs">刷新</el-button>
        </div>
      </template>

      <el-table :data="logs" v-loading="loading" style="width: 100%" max-height="600">
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="levelTagType(row.level)" size="small">
              {{ row.level.toUpperCase() }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="event_type" label="事件类型" width="200" />
        <el-table-column prop="message" label="消息" min-width="300" show-overflow-tooltip />
        <el-table-column label="设备/任务" width="200">
          <template #default="{ row }">
            <span v-if="row.device_id" class="link-id">设备: {{ row.device_id.substring(0, 8) }}...</span>
            <span v-if="row.task_id" class="link-id">任务: {{ row.task_id.substring(0, 8) }}...</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { systemApi } from '@/api/system'

interface LogEntry {
  id: number
  timestamp: string
  level: string
  event_type: string
  message: string
  device_id?: string
  task_id?: string
}

const logs = ref<LogEntry[]>([])
const loading = ref(false)
const levelFilter = ref('')
const dateRange = ref<[Date, Date] | null>(null)

async function fetchLogs() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page_size: 100 }
    if (levelFilter.value) params.level = levelFilter.value
    const res = await systemApi.getLogs(params)
    logs.value = res.data.data.list || []
  } catch {
    // 错误已处理
  } finally {
    loading.value = false
  }
}

function levelTagType(level: string) {
  const map: Record<string, string> = {
    debug: 'info',
    info: '',
    warning: 'warning',
    error: 'danger',
  }
  return map[level] || 'info'
}

function formatTime(time: string) {
  if (!time || time.startsWith('0001')) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchLogs()
})
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-left { display: flex; gap: 10px; }
.link-id { font-family: monospace; font-size: 12px; color: #909399; display: block; }
</style>
