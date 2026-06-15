<template>
  <div class="page-container">
    <el-row :gutter="20">
      <el-col :span="12">
        <el-card>
          <template #header>系统配置</template>
          <el-form :model="config" label-width="140px" v-loading="loading">
            <el-divider content-position="left">服务器</el-divider>
            <el-form-item label="服务端口">
              <el-input-number v-model="config.server.port" :min="1" :max="65535" />
            </el-form-item>

            <el-divider content-position="left">扫描</el-divider>
            <el-form-item label="默认分辨率">
              <el-select v-model="config.scan.default_resolution" style="width: 180px">
                <el-option label="100 DPI" :value="100" />
                <el-option label="150 DPI" :value="150" />
                <el-option label="300 DPI" :value="300" />
                <el-option label="600 DPI" :value="600" />
              </el-select>
            </el-form-item>
            <el-form-item label="默认色彩模式">
              <el-select v-model="config.scan.default_color_mode" style="width: 180px">
                <el-option label="彩色" value="Color" />
                <el-option label="灰度" value="Grayscale" />
                <el-option label="黑白" value="BW" />
              </el-select>
            </el-form-item>
            <el-form-item label="最大并发扫描">
              <el-input-number v-model="config.scan.max_concurrent" :min="1" :max="20" />
            </el-form-item>

            <el-divider content-position="left">设备监控</el-divider>
            <el-form-item label="自动发现设备">
              <el-switch v-model="config.device.auto_discover" />
            </el-form-item>
            <el-form-item label="监控间隔(秒)">
              <el-input-number v-model="monitorIntervalSec" :min="1" :max="60" />
            </el-form-item>

            <el-divider content-position="left">日志</el-divider>
            <el-form-item label="日志级别">
              <el-select v-model="config.log.level" style="width: 180px">
                <el-option label="Debug" value="debug" />
                <el-option label="Info" value="info" />
                <el-option label="Warning" value="warning" />
                <el-option label="Error" value="error" />
              </el-select>
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="handleSave" :loading="saving">保存配置</el-button>
              <el-button @click="fetchConfig">重置</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>系统信息</template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="系统版本">{{ status?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ formatUptime(status?.uptime || 0) }}</el-descriptions-item>
            <el-descriptions-item label="Go版本">{{ status?.go_version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="运行平台">{{ status?.platform || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card style="margin-top: 20px">
          <template #header>存储信息</template>
          <el-descriptions :column="1" border v-if="status">
            <el-descriptions-item label="总容量">{{ formatBytes(status.storage.total) }}</el-descriptions-item>
            <el-descriptions-item label="已使用">{{ formatBytes(status.storage.used) }}</el-descriptions-item>
            <el-descriptions-item label="可用空间">{{ formatBytes(status.storage.free) }}</el-descriptions-item>
            <el-descriptions-item label="文件数量">{{ status.storage.file_count }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { systemApi } from '@/api/system'
import { useSystemStore } from '@/stores/system'

interface SystemConfig {
  server: { port: number }
  scan: { default_resolution: number; default_color_mode: string; max_concurrent: number }
  device: { auto_discover: boolean; monitor_interval: number }
  log: { level: string }
}

const systemStore = useSystemStore()
const loading = ref(false)
const saving = ref(false)

const config = ref<SystemConfig>({
  server: { port: 8080 },
  scan: { default_resolution: 300, default_color_mode: 'Color', max_concurrent: 5 },
  device: { auto_discover: true, monitor_interval: 2000000000 },
  log: { level: 'info' },
})

const status = computed(() => systemStore.status)

const monitorIntervalSec = computed({
  get: () => Math.floor((config.value.device.monitor_interval || 2000000000) / 1000000000),
  set: (val: number) => {
    config.value.device.monitor_interval = val * 1000000000
  },
})

async function fetchConfig() {
  loading.value = true
  try {
    const res = await systemApi.getConfig()
    config.value = res.data.data
  } catch {
    // 错误已处理
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    await systemApi.updateConfig(config.value as unknown as Record<string, unknown>)
    ElMessage.success('配置保存成功')
  } catch {
    // 错误已处理
  } finally {
    saving.value = false
  }
}

function formatBytes(bytes: number) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatUptime(seconds: number) {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${days}天 ${hours}小时 ${mins}分钟`
}

onMounted(() => {
  fetchConfig()
  systemStore.fetchStatus()
})
</script>
