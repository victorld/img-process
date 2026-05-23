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
  totalFolderCount?: number
  totalBackupFileCount?: number
  totalBackupFolderCount?: number
  pendingActionCount?: number
  executedActionCount?: number
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
  detail?: ScanActionDetail
  pair?: ScanActionPair
  duplicatePhotos?: ScanActionPhoto[]
  duplicateMeta?: ScanActionDuplicateMeta
}

export type ScanActionCounts = {
  delete: number
  move: number
  modifyTime: number
  deleteDuplicate: number
  deletePathDuplicate: number
  rename: number
  total: number
}

export type ScanActionGroupedCounts = {
  pending: ScanActionCounts
  executed: ScanActionCounts
  error: ScanActionCounts
}

export type ScanActionDetail = {
  fileName?: string
  currentPath?: string
  targetPath?: string
  targetFileName?: string
  dirDate?: string
  modifyDate?: string
  fileNameDate?: string
  shootDate?: string
  shootDateRaw?: string
  minDate?: string
  previewSlot?: string
}

export type ScanActionPhoto = {
  fileName: string
  path: string
  previewSlot?: string
  sizeBytes?: number
  sizeText?: string
  md5Matched?: boolean
  matchKey?: string
  matchType?: string
  pathSource?: string
  recommendedDelete?: boolean
  executedAction?: boolean
  deleteEligible?: boolean
  candidateIndex?: number
  matchCount?: number
}

export type ScanActionPair = {
  photoA: ScanActionPhoto
  photoB: ScanActionPhoto
}

export type ScanActionDuplicateMeta = {
  sizeMatch: boolean
  deleteEligible: boolean
  deleteIneligibleReason?: string
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
  configSource?: string
  configFile?: string
  readonlySections?: string[]
  config?: {
    database: Record<string, unknown>
    server: Record<string, unknown>
    scanArgs: Record<string, unknown>
    basic: Record<string, unknown>
    cache: Record<string, unknown>
    dump: Record<string, unknown>
    bak: Record<string, unknown>
    gis: Record<string, unknown>
    batch: Record<string, unknown>
  }
  server: {
    httpPort: string
    startPath: string
    startPathBak: string
    poolSize: number
    imgCache: boolean
    sqlDebug: boolean
    scanDefaults?: Record<string, unknown>
  }
}
