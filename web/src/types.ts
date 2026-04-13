export type ApiEnvelope<T> = {
  code: number
  data: T
  msg: string
}

export type Job = {
  id: number
  jobUuid: string
  scanUuid?: string
  source: string
  status: string
  scheduleId?: number
  queueAt?: string
  startAt?: string
  endAt?: string
  currentPhase?: string
  lastHeartbeatAt?: string
  processedCount: number
  totalCount: number
  hasAction: boolean
  scanArgs?: Record<string, unknown>
  summary?: Record<string, unknown>
  artifactPath?: string
  errorMessage?: string
  createdAt: string
  updatedAt: string
}

export type ScanActionItem = {
  id: number
  actionType: string
  objectType: string
  sourcePath: string
  targetPath?: string
  reasonCode?: string
  reasonText?: string
  stage: string
  status: string
  discoveredAt?: string
  executedAt?: string
  errorMessage?: string
  metadataJson?: string
  duplicateGroup?: string
}

export type ScanEvent = {
  id: number
  eventType: string
  phase: string
  level: string
  title: string
  message: string
  relatedPath?: string
  payloadJson?: string
  createdAt: string
}

export type ScanJobLog = {
  id: number
  jobId: number
  level: string
  phase: string
  message: string
  payloadJson?: string
  createdAt: string
}

export type Schedule = {
  id: number
  name: string
  enabled: boolean
  timezone: string
  mode: string
  cronExpr: string
  scheduleConfig: string
  scanArgs: string
  lastRunAt?: string
  nextRunAt?: string
  lastJobId?: number
  lastJobStatus?: string
}

export type SystemStatus = {
  server: {
    httpPort: string
    startPath: string
    startPathBak: string
    bakStatEnable: boolean
    poolSize: number
    imgCache: boolean
    sqlDebug: boolean
    scanDefaults?: Record<string, unknown>
  }
}
