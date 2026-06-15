<template>
  <el-container class="layout">
    <el-aside width="220px" class="sidebar">
      <div class="logo">
        <el-icon size="24"><Printer /></el-icon>
        <span>Auto Scan</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        class="menu"
      >
        <el-menu-item
          v-for="route in menuRoutes"
          :key="route.path"
          :index="route.path"
        >
          <el-icon><component :is="route.meta?.icon" /></el-icon>
          <span>{{ route.meta?.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-title">
          {{ currentTitle }}
        </div>
        <div class="header-actions">
          <el-badge :value="onlineCount" :hidden="onlineCount === 0" type="success">
            <el-icon size="20"><Monitor /></el-icon>
          </el-badge>
          <el-tooltip content="刷新数据" placement="bottom">
            <el-button :icon="Refresh" circle @click="refreshData" />
          </el-tooltip>
        </div>
      </el-header>

      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Refresh, Monitor, Printer } from '@element-plus/icons-vue'
import router from '@/router'
import { useDeviceStore } from '@/stores/device'
import { useSystemStore } from '@/stores/system'

const route = useRoute()
const deviceStore = useDeviceStore()
const systemStore = useSystemStore()

const menuRoutes = computed(() =>
  router.options.routes[0].children?.filter((r) => !r.meta?.hidden) || []
)

const activeMenu = computed(() => '/' + (route.path.split('/')[1] || 'dashboard'))

const currentTitle = computed(() => (route.meta?.title as string) || 'Auto Scan')

const onlineCount = computed(() => deviceStore.onlineCount)

function refreshData() {
  deviceStore.fetchDevices()
  systemStore.fetchStatus()
}

onMounted(() => {
  deviceStore.fetchDevices()
  systemStore.fetchStatus()
})
</script>

<style scoped>
.layout {
  height: 100vh;
}

.sidebar {
  background-color: #304156;
  overflow: hidden;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  gap: 10px;
  background-color: #2b3a4d;
}

.menu {
  border-right: none;
  background-color: #304156;
}

:deep(.el-menu-item) {
  color: #bfcbd9;
}

:deep(.el-menu-item:hover) {
  background-color: #263445;
}

:deep(.el-menu-item.is-active) {
  color: #409eff;
  background-color: #263445;
}

.header {
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
}

.header-title {
  font-size: 18px;
  font-weight: 500;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 15px;
}

.main {
  background-color: #f0f2f5;
  overflow-y: auto;
}
</style>
