<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-input
              v-model="searchQuery"
              placeholder="搜索设备名称或IP"
              style="width: 250px"
              clearable
              :prefix-icon="Search"
            />
            <el-select v-model="statusFilter" placeholder="状态" clearable style="width: 120px">
              <el-option label="在线" value="online" />
              <el-option label="离线" value="offline" />
              <el-option label="忙碌" value="busy" />
              <el-option label="错误" value="error" />
            </el-select>
            <el-select v-model="vendorFilter" placeholder="厂商" clearable style="width: 120px">
              <el-option v-for="v in vendors" :key="v" :label="v" :value="v" />
            </el-select>
          </div>
          <div class="header-right">
            <el-button type="primary" :icon="Search" @click="handleDiscover" :loading="discovering">
              发现设备
            </el-button>
            <el-button type="success" :icon="Plus" @click="showAddDialog = true">
              添加设备
            </el-button>
          </div>
        </div>
      </template>

      <el-table
        :data="filteredDevices"
        v-loading="deviceStore.loading"
        style="width: 100%"
      >
        <el-table-column prop="name" label="设备名称" min-width="150" />
        <el-table-column prop="vendor" label="厂商" width="100" />
        <el-table-column prop="model" label="型号" min-width="150" />
        <el-table-column prop="ip_address" label="IP地址" width="130" />
        <el-table-column prop="protocol" label="协议" width="80">
          <template #default="{ row }">
            <el-tag size="small">{{ row.protocol.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后在线" width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_seen) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/devices/${row.id}`)">
              详情
            </el-button>
            <el-button
              v-if="row.status !== 'online'"
              link
              type="success"
              @click="handleConnect(row)"
            >
              连接
            </el-button>
            <el-button
              v-else
              link
              type="warning"
              @click="handleDisconnect(row)"
            >
              断开
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加设备对话框 -->
    <el-dialog v-model="showAddDialog" title="添加设备" width="500px">
      <el-form :model="addForm" label-width="80px" ref="addFormRef" :rules="addRules">
        <el-form-item label="名称" prop="name">
          <el-input v-model="addForm.name" placeholder="请输入设备名称" />
        </el-form-item>
        <el-form-item label="IP地址" prop="ip_address">
          <el-input v-model="addForm.ip_address" placeholder="例如: 192.168.1.100" />
        </el-form-item>
        <el-form-item label="协议" prop="protocol">
          <el-radio-group v-model="addForm.protocol">
            <el-radio value="escl">eSCL (推荐)</el-radio>
            <el-radio value="wsd">WSD</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAdd" :loading="adding">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useDeviceStore } from '@/stores/device'
import { deviceApi } from '@/api/device'
import type { Device, CreateDeviceRequest } from '@/api/types'

const deviceStore = useDeviceStore()

const searchQuery = ref('')
const statusFilter = ref('')
const vendorFilter = ref('')
const discovering = ref(false)
const showAddDialog = ref(false)
const adding = ref(false)
const addFormRef = ref()

const vendors = ['HP', 'Canon', 'Ricoh', 'Fujitsu', 'Brother', 'Epson']

const addForm = ref<CreateDeviceRequest>({
  name: '',
  ip_address: '',
  protocol: 'escl',
})

const addRules = {
  name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  ip_address: [
    { required: true, message: '请输入IP地址', trigger: 'blur' },
    {
      pattern: /^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$/,
      message: 'IP地址格式不正确',
      trigger: 'blur',
    },
  ],
}

const filteredDevices = computed(() => {
  let list = deviceStore.devices
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(
      (d) =>
        d.name.toLowerCase().includes(q) ||
        d.ip_address.includes(q) ||
        (d.model || '').toLowerCase().includes(q)
    )
  }
  if (statusFilter.value) {
    list = list.filter((d) => d.status === statusFilter.value)
  }
  if (vendorFilter.value) {
    list = list.filter((d) => d.vendor === vendorFilter.value)
  }
  return list
})

async function handleDiscover() {
  discovering.value = true
  try {
    await deviceStore.discoverDevices()
  } catch {
    // 错误已在拦截器处理
  } finally {
    discovering.value = false
  }
}

async function handleAdd() {
  await addFormRef.value?.validate(async (valid: boolean) => {
    if (!valid) return
    adding.value = true
    try {
      await deviceStore.addDevice(addForm.value)
      showAddDialog.value = false
      addForm.value = { name: '', ip_address: '', protocol: 'escl' }
    } catch {
      // 错误已处理
    } finally {
      adding.value = false
    }
  })
}

async function handleConnect(device: Device) {
  try {
    await deviceApi.connect(device.id)
    deviceStore.updateDeviceStatus(device.id, 'online')
    ElMessage.success('设备已连接')
  } catch {
    // 错误已处理
  }
}

async function handleDisconnect(device: Device) {
  try {
    await deviceApi.disconnect(device.id)
    deviceStore.updateDeviceStatus(device.id, 'offline')
    ElMessage.success('设备已断开')
  } catch {
    // 错误已处理
  }
}

async function handleDelete(device: Device) {
  try {
    await ElMessageBox.confirm(
      `确定要删除设备 "${device.name}" 吗？`,
      '提示',
      { type: 'warning' }
    )
    await deviceStore.removeDevice(device.id)
  } catch {
    // 用户取消或错误
  }
}

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

function formatTime(time: string) {
  if (!time || time.startsWith('0001')) return '-'
  return new Date(time).toLocaleString('zh-CN')
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-left {
  display: flex;
  gap: 10px;
}
.header-right {
  display: flex;
  gap: 10px;
}
</style>
