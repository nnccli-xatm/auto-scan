<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px">
              <el-option label="等待中" value="pending" />
              <el-option label="执行中" value="running" />
              <el-option label="已完成" value="completed" />
              <el-option label="失败" value="failed" />
              <el-option label="已取消" value="cancelled" />
            </el-select>
          </div>
          <el-button :icon="Refresh" @click="taskStore.fetchTasks()">刷新</el-button>
        </div>
      </template>

      <el-table :data="filteredTasks" v-loading="taskStore.loading" style="width: 100%">
        <el-table-column prop="id" label="任务ID" width="140">
          <template #default="{ row }">
            <span class="task-id">{{ row.id.substring(0, 8) }}...</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="taskStatusTagType(row.status)" size="small">
              {{ taskStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" min-width="200">
          <template #default="{ row }">
            <el-progress
              :percentage="row.progress"
              :status="progressStatus(row.status)"
              :stroke-width="12"
            />
            <span class="page-info">{{ row.scanned_pages }}/{{ row.total_pages }} 页</span>
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="80">
          <template #default="{ row }">
            <el-tag size="small" type="info">P{{ row.priority }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'pending' || row.status === 'running'"
              link
              type="danger"
              @click="handleCancel(row)"
            >
              取消
            </el-button>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { useTaskStore } from '@/stores/task'
import type { ScanTask } from '@/api/types'

const taskStore = useTaskStore()
const statusFilter = ref('')

const filteredTasks = computed(() => {
  if (!statusFilter.value) return taskStore.tasks
  return taskStore.tasks.filter((t) => t.status === statusFilter.value)
})

async function handleCancel(task: ScanTask) {
  await taskStore.cancelTask(task.id)
}

function taskStatusTagType(status: string) {
  const map: Record<string, string> = {
    pending: 'info', running: 'warning', completed: 'success',
    failed: 'danger', cancelled: 'info',
  }
  return map[status] || 'info'
}

function taskStatusText(status: string) {
  const map: Record<string, string> = {
    pending: '等待中', running: '执行中', completed: '已完成',
    failed: '失败', cancelled: '已取消',
  }
  return map[status] || status
}

function progressStatus(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'exception'
  return undefined
}

function formatTime(time: string) {
  if (!time || time.startsWith('0001')) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  taskStore.fetchTasks()
})
</script>

<style scoped>
.task-id { font-family: monospace; color: #909399; }
.page-info { font-size: 12px; color: #909399; margin-left: 8px; }
.text-muted { color: #c0c4cc; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-left { display: flex; gap: 10px; }
</style>
