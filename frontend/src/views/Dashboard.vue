<template>
  <div class="page-container">
    <el-row :gutter="20" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-value">{{ status?.devices?.total || 0 }}</div>
            <div class="stat-label">设备总数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-value" style="color: #67c23a">{{ status?.devices?.online || 0 }}</div>
            <div class="stat-label">在线设备</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-value" style="color: #e6a23c">{{ status?.tasks?.running || 0 }}</div>
            <div class="stat-label">进行中任务</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-value" style="color: #909399">{{ status?.storage?.file_count || 0 }}</div>
            <div class="stat-label">扫描文件数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>设备概览</span>
              <el-button type="primary" link @click="$router.push('/devices')">查看全部</el-button>
            </div>
          </template>
          <el-table :data="recentDevices" style="width: 100%" size="small">
            <el-table-column prop="name" label="设备名称" />
            <el-table-column prop="ip_address" label="IP地址" width="130" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusTagType(row.status)" size="small">
                  {{ statusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>存储使用情况</span>
            </div>
          </template>
          <div v-if="status">
            <el-progress
              :percentage="storagePercentage"
              :color="storageColor"
              :stroke-width="20"
              :text-inside="true"
            />
            <el-descriptions :column="1" border style="margin-top: 20px">
              <el-descriptions-item label="总容量">
                {{ formatBytes(status.storage.total) }}
              </el-descriptions-item>
              <el-descriptions-item label="已使用">
                {{ formatBytes(status.storage.used) }}
              </el-descriptions-item>
              <el-descriptions-item label="可用空间">
                {{ formatBytes(status.storage.free) }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty v-else description="暂无数据" />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近任务</span>
              <el-button type="primary" link @click="$router.push('/tasks')">查看全部</el-button>
            </div>
          </template>
          <el-table :data="recentTasks" style="width: 100%" size="small">
            <el-table-column prop="id" label="任务ID" width="180">
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
            <el-table-column label="进度" width="150">
              <template #default="{ row }">
                <el-progress :percentage="row.progress" :stroke-width="10" />
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useSystemStore } from '@/stores/system'
import { useDeviceStore } from '@/stores/device'
import { useTaskStore } from '@/stores/task'

const systemStore = useSystemStore()
const deviceStore = useDeviceStore()
const taskStore = useTaskStore()

const status = computed(() => systemStore.status)
const recentDevices = computed(() => deviceStore.devices.slice(0, 5))
const recentTasks = computed(() => taskStore.tasks.slice(0, 5))

const storagePercentage = computed(() => {
  if (!status.value || status.value.storage.total === 0) return 0
  return Math.round((status.value.storage.used / status.value.storage.total) * 100)
})

const storageColor = computed(() => {
  const p = storagePercentage.value
  if (p > 90) return '#f56c6c'
  if (p > 70) return '#e6a23c'
  return '#67c23a'
})

function statusTagType(status: string) {
  const map: Record<string, string> = {
    online: 'success',
    offline: 'info',
    busy: 'warning',
    error: 'danger',
  }
  return map[status] || 'info'
}

function statusText(status: string) {
  const map: Record<string, string> = {
    online: '在线',
    offline: '离线',
    busy: '忙碌',
    error: '错误',
  }
  return map[status] || status
}

function taskStatusTagType(status: string) {
  const map: Record<string, string> = {
    pending: 'info',
    running: 'warning',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
  }
  return map[status] || 'info'
}

function taskStatusText(status: string) {
  const map: Record<string, string> = {
    pending: '等待中',
    running: '执行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
  }
  return map[status] || status
}

function formatBytes(bytes: number) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatTime(time: string) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  systemStore.fetchStatus()
  deviceStore.fetchDevices()
  taskStore.fetchTasks()
})
</script>

<style scoped>
.stat-row {
  margin-bottom: 10px;
}
.task-id {
  font-family: monospace;
  color: #909399;
}
</style>
