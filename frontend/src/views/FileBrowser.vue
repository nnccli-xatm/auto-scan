<template>
  <div class="page-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>扫描文件 ({{ files.length }})</span>
          <div class="header-right">
            <el-button
              v-if="selectedFiles.length > 0"
              type="danger"
              :icon="Delete"
              @click="handleBatchDelete"
            >
              批量删除 ({{ selectedFiles.length }})
            </el-button>
            <el-button
              v-if="selectedFiles.length > 0"
              type="primary"
              :icon="Download"
              @click="handleBatchDownload"
            >
              批量下载
            </el-button>
            <el-button :icon="Refresh" circle @click="fetchFiles" />
          </div>
        </div>
      </template>

      <el-table
        :data="files"
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column label="预览" width="100">
          <template #default="{ row }">
            <el-image
              :src="getThumbnailUrl(row.id)"
              :preview-src-list="[getDownloadUrl(row.id)]"
              fit="cover"
              style="width: 60px; height: 80px"
              lazy
            >
              <template #error>
                <div class="image-placeholder">
                  <el-icon><Picture /></el-icon>
                </div>
              </template>
            </el-image>
          </template>
        </el-table-column>
        <el-table-column prop="filename" label="文件名" min-width="180" />
        <el-table-column label="大小" width="100">
          <template #default="{ row }">
            {{ formatBytes(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column prop="page_number" label="页码" width="80" />
        <el-table-column prop="format" label="格式" width="80">
          <template #default="{ row }">
            <el-tag size="small">{{ row.format }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="扫描时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleDownload(row)">
              <el-icon><Download /></el-icon> 下载
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh, Download, Delete, Picture } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fileApi } from '@/api/file'
import type { ScanFile } from '@/api/types'

const files = ref<ScanFile[]>([])
const loading = ref(false)
const selectedFiles = ref<ScanFile[]>([])

async function fetchFiles() {
  loading.value = true
  try {
    const res = await fileApi.list({ page_size: 100 })
    files.value = res.data.data.list || []
  } catch {
    // 错误已处理
  } finally {
    loading.value = false
  }
}

function handleSelectionChange(selection: ScanFile[]) {
  selectedFiles.value = selection
}

function getDownloadUrl(id: string): string {
  return fileApi.getDownloadUrl(id)
}

function getThumbnailUrl(id: string): string {
  return fileApi.getDownloadUrl(id)
}

async function handleDownload(file: ScanFile) {
  try {
    const res = await fileApi.download(file.id)
    const blob = new Blob([res.data])
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.filename
    link.click()
    window.URL.revokeObjectURL(url)
  } catch {
    // 错误已处理
  }
}

async function handleDelete(file: ScanFile) {
  try {
    await ElMessageBox.confirm(`确定要删除文件 "${file.filename}" 吗？`, '提示', {
      type: 'warning',
    })
    await fileApi.delete(file.id)
    files.value = files.value.filter((f) => f.id !== file.id)
    ElMessage.success('文件已删除')
  } catch {
    // 用户取消或错误
  }
}

async function handleBatchDelete() {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedFiles.value.length} 个文件吗？`,
      '批量删除',
      { type: 'warning' }
    )
    const ids = selectedFiles.value.map((f) => f.id)
    await fileApi.batchDelete(ids)
    files.value = files.value.filter((f) => !ids.includes(f.id))
    selectedFiles.value = []
    ElMessage.success('批量删除成功')
  } catch {
    // 用户取消或错误
  }
}

async function handleBatchDownload() {
  try {
    const ids = selectedFiles.value.map((f) => f.id)
    const res = await fileApi.batchDownload(ids)
    const blob = new Blob([res.data], { type: 'application/zip' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `scans_${Date.now()}.zip`
    link.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('批量下载成功')
  } catch {
    // 错误已处理
  }
}

function formatBytes(bytes: number) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatTime(time: string) {
  if (!time || time.startsWith('0001')) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchFiles()
})
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.header-right { display: flex; gap: 10px; align-items: center; }
.image-placeholder {
  width: 60px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  color: #c0c4cc;
  font-size: 24px;
}
</style>
