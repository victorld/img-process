export function formatJobStatus(value?: string): string {
  switch (value) {
    case 'pending':
      return '待执行'
    case 'running':
      return '运行中'
    case 'succeeded':
      return '成功'
    case 'failed':
      return '失败'
    case 'interrupted':
      return '已中断'
    case 'skipped':
      return '已跳过'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatJobSource(value?: string): string {
  switch (value) {
    case 'manual':
      return '手动'
    case 'schedule':
      return '计划'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatPhase(value?: string): string {
  // These values are persisted/returned as internal enums, only the display string is localized.
  switch (value) {
    case 'queued':
      return '已入队'
    case 'starting':
      return '启动中'
    case 'initializing':
      return '初始化'
    case 'scan_primary':
      return '扫描主目录'
    case 'scan_backup':
      return '扫描备份目录'
    case 'process_actions':
      return '生成待执行动作'
    case 'process_duplicates':
      return '处理重复项'
    case 'build_result':
      return '生成结果'
    case 'post_action':
      return '后置动作'
    case 'artifact':
      return '产物'
    case 'lifecycle':
      return '生命周期'
    case 'completed':
      return '已完成'
    case 'runtime':
      return '运行时'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatActionStage(value?: string): string {
  switch (value) {
    case 'candidate':
      return '候选'
    case 'executed':
      return '已执行'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatActionStatus(value?: string): string {
  switch (value) {
    case 'pending':
      return '待执行'
    case 'succeeded':
      return '成功'
    case 'failed':
      return '失败'
    case 'skipped':
      return '已跳过'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatActionType(value?: string): string {
  switch (value) {
    case 'delete':
      return '删除'
    case 'move':
      return '移动'
    case 'rename':
      return '重命名'
    case 'modify_time':
      return '修改时间'
    case 'delete_empty_dir':
      return '删除空目录'
    case 'delete_duplicate':
      return '重复项删除'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatScheduleMode(value?: string): string {
  switch (value) {
    case 'daily':
      return '每天'
    case 'weekly':
      return '每周'
    case 'monthly':
      return '每月'
    case 'custom':
      return '自定义 Cron'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}

export function formatLogLevel(value?: string): string {
  switch (value) {
    case 'info':
      return '信息'
    case 'error':
      return '错误'
    case 'warn':
      return '警告'
    case 'debug':
      return '调试'
    case '':
    case undefined:
    case null:
      return '未知'
    default:
      return value
  }
}
