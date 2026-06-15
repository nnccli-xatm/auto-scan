// API类型定义

export interface Device {
  id: string
  name: string
  ip_address: string
  protocol: string
  model: string
  vendor: string
  status: 'online' | 'offline' | 'busy' | 'error'
  capabilities?: string
  config?: string
  last_seen: string
  created_at: string
  updated_at: string
}

export interface CreateDeviceRequest {
  name: string
  ip_address: string
  protocol: 'escl' | 'wsd'
}

export interface DeviceStatus {
  device_id: string
  status: string
  adf_status: string
  scanner_state: string
  current_task?: string
  last_seen: string
}

export interface ScanTask {
  id: string
  device_id: string
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled'
  priority: number
  settings: string
  result: string
  progress: number
  total_pages: number
  scanned_pages: number
  error_message: string
  started_at: string
  completed_at: string
  created_at: string
}

export interface CreateTaskRequest {
  device_id: string
  priority?: number
  settings: ScanSettings
}

export interface ScanSettings {
  resolution: number
  color_mode: string
  format: string
  input_source: string
}

export interface ScanFile {
  id: string
  task_id: string
  device_id: string
  filename: string
  original_name: string
  file_path: string
  file_size: number
  checksum: string
  page_number: number
  width: number
  height: number
  format: string
  status: string
  created_at: string
}

export interface SystemStatus {
  version: string
  uptime: number
  go_version: string
  platform: string
  devices: DeviceStats
  tasks: TaskStats
  storage: StorageStats
}

export interface DeviceStats {
  total: number
  online: number
  offline: number
  busy: number
  error: number
}

export interface TaskStats {
  pending: number
  running: number
  completed: number
  failed: number
  cancelled: number
}

export interface StorageStats {
  total: number
  used: number
  free: number
  file_count: number
}

export interface Pagination {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PaginationData<T> {
  list: T[]
  pagination: Pagination
}
