<template>
  <div class="page-container">
    <el-page-header @back="$router.back()" :content="device?.name || '设备详情'" style="margin-bottom: 20px" />

    <el-row :gutter="20" v-if="device">
      <el-col :span="16">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>基本信息</span>
              <el-tag :type="statusTagType(device.status)">{{ statusText(device.status) }}</el-tag>
            </div>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="设备ID">{{ device.id }}</el-descriptions-item>
            <el-descriptions-item label="设备名称">{{ device.name }}</el-descriptions-item>
            <el-descriptions-item label="厂商">{{ device.vendor }}</el-descriptions-item>
            <el-descriptions-item label="型号">{{ device.model }}</el-descriptions-item>
            <el-descriptions-item label="IP地址">{{ device.ip_address }}</el-descriptions-item>
            <el-descriptions-item label="协议">{{ device.protocol.toUpperCase() }}</el-descriptions-item>
            <el-descriptions-item label="最后在线">{{ formatTime(device.last_seen) }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ formatTime(device.created_at) }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card style="margin-top: 20px">
          <template #header>
            <div class="card-header">
              <span>实时状态</span>
              <el-button :icon="Refresh" circle size="small" @click="fetchStatus" />
            </div>
          </template>
          <el-descriptions :column="2" border v-if="deviceStatus">
            <el-descriptions-item label="扫描仪状态">
              <el-tag size="small">{{ deviceStatus.scanner_state || '-' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="ADF状态">
              <el-tag :type="deviceStatus.adf_status === 'ScannerAdfLoaded' ? 'success' : 'info'" size="small">
                {{ adfText(deviceStatus.adf_status) }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
          <el-empty v-else description="点击右上角刷新获取状态" />
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card>
          <template #header>快捷操作</template>
          <div class="actions">
            <el-button type="primary" @click="showScanDialog = true" :disabled="device.status === 'offline'">
              <el-icon><VideoPlay /></el-icon> 开始扫描
            </el-button>
            <el-button @click="$router.push('/tasks')">
              <el-icon><List /></el-icon> 查看任务
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 扫描对话框 -->
    <el-dialog v-model="showScanDialog" title="开始扫描" width="450px">
      <el-form :model="scanForm" label-width="90px">
        <el-form-item label="分辨率">
          <el-select v-model="scanForm.resolution" style="width: 100%">
            <el-option label="100 DPI" :value="100" />
            <el-option label="150 DPI" :value="150" />
            <el-option label="200 DPI" :value="200" />
            <el-option label="300 DPI (推荐)" :value="300" />
            <el-option label="600 DPI" :value="600" />
          </el-select>
        </el-form-item>
        <el-form-item label="色彩模式">
          <el-radio-group v-model="scanForm.color_mode">
            <el-radio value="Color">彩色</el-radio>
            <el-radio value="Grayscale">灰度</el-radio>
            <el-radio value="BW">黑白</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="输出格式">
          <el-radio-group v-model="scanForm.format">
            <el-radio value="JPEG">JPEG</el-radio>
            <el-radio value="PDF">PDF</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="优先级">
          <el-slider v-model="scanForm.priority" :min="1" :max="10" show-input style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showScanDialog = false">取消</el-button>
        <el-button type="primary" @click="handleStartScan" :loading="scanning">开始</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Refresh, VideoPlay, List } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { deviceApi } from '@/api/device'
import { taskApi } from '@/api/task'
import type { Device, DeviceStatus, ScanSettings } from '@/api/types'

const route = useRoute()
const router = useRouter()

const device = ref<Device | null>(null)
const deviceStatus = ref<DeviceStatus | null>(null)
const showScanDialog = ref(false)
const scanning = ref(false)

const scanForm = ref({
  resolution: 300,
  color_mode: 'Color',
  format: 'JPEG',
  priority: 5,
})

async function fetchDevice() {
  try {
    const id = route.params.id as string
    const res = await deviceApi.get(id)
    device.value = res.data.data
  } catch {
    ElMessage.error('获取设备信息失败')
  }
}

async function fetchStatus() {
  try {
    const id = route.params.id as string
    const res = await deviceApi.getStatus(id)
    deviceStatus.value = res.data.data
  } catch {
    // 错误已处理
  }
}

async function handleStartScan() {
  scanning.value = true
  try {
    const settings: ScanSettings = {
      resolution: scanForm.value.resolution,
      color_mode: scanForm.value.color_mode,
      format: scanForm.value.format,
      input_source: 'Feeder',
    }
    await taskApi.create({
      device_id: device.value!.id,
      priority: scanForm.value.priority,
      settings,
    })
    ElMessage.success('扫描任务已创建')
    showScanDialog.value = false
    router.push('/tasks')
  } catch {
    // 错误已处理
  } finally {
    scanning.value = false
  }
}

function statusTagType(status: string) {
  const map: Record<string, string> = {
    online: 'success', offline: 'info', busy: 'warning', error: 'danger',
  }
  return map[status] || 'info'
}

function statusText(status: string) {
  const map: Record<string, string> = {
    online: '在线', offline: '离线', busy: '忙碌', error: '错误',
  }
  return map[status] || status
}

function adfText(status: string) {
  if (status === 'ScannerAdfLoaded' || status === 'loaded') return '已装入纸张'
  if (status === 'ScannerAdfEmpty' || status === 'empty') return '空'
  return status || '-'
}

function formatTime(time: string) {
  if (!time || time.startsWith('0001')) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchDevice()
})
</script>

<style scoped>
.actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.actions .el-button {
  width: 100%;
}
</style>
