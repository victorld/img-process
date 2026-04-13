import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Col, Descriptions, Empty, Image, Row, Space, Statistic, Table, Tabs, Tag, Timeline, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api'
import type { Job, ScanActionCounts, ScanActionItem, ScanEvent, ScanJobLog } from '../types'

const pendingTypeOptions = [
  { key: 'delete', label: '删除' },
  { key: 'move', label: '移动' },
  { key: 'modify_time', label: '修改时间' },
  { key: 'delete_duplicate', label: '重复项' },
  { key: 'rename', label: '重命名' },
] as const

const historyActionColumns: ColumnsType<ScanActionItem> = [
  { title: '动作类型', dataIndex: 'actionType', width: 140 },
  { title: '源路径', dataIndex: 'sourcePath', ellipsis: true },
  { title: '目标路径', dataIndex: 'targetPath', ellipsis: true },
  { title: '原因', dataIndex: 'reasonText', ellipsis: true, width: 220 },
  { title: '状态', dataIndex: 'status', width: 120, render: (value) => <Tag>{value}</Tag> },
  { title: '发现时间', dataIndex: 'discoveredAt', width: 180 },
  { title: '执行时间', dataIndex: 'executedAt', width: 180 },
  { title: '错误', dataIndex: 'errorMessage', ellipsis: true, width: 240 },
]

export function JobDetailPage() {
  const { id = '' } = useParams()
  const [messageApi, contextHolder] = message.useMessage()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState('pending')
  const [pendingActionType, setPendingActionType] = useState<string>('delete')
  const [actionPage, setActionPage] = useState({ current: 1, pageSize: 20 })
  const actionTabs = ['pending', 'executed', 'error']

  const jobQuery = useQuery({
    queryKey: ['job', id],
    queryFn: () => api.getJob(id),
    refetchInterval: (query) => {
      const status = (query.state.data?.job.status ?? '') as string
      return status === 'running' || status === 'pending' ? 2000 : false
    },
  })

  const actionQuery = useQuery({
    queryKey: ['job-actions', id, activeTab, pendingActionType, actionPage],
    enabled: actionTabs.includes(activeTab),
    queryFn: () => {
      const params = new URLSearchParams({
        tab: activeTab,
        page: String(actionPage.current),
        pageSize: String(actionPage.pageSize),
      })
      if (activeTab === 'pending') {
        params.set('type', pendingActionType)
      }
      return api.getJobActionItems(id, params)
    },
  })

  const eventQuery = useQuery({
    queryKey: ['job-events', id],
    queryFn: () => api.getJobEvents(id, new URLSearchParams({ page: '1', pageSize: '50' })),
    refetchInterval: 3000,
  })

  const logQuery = useQuery({
    queryKey: ['job-logs', id],
    queryFn: () => api.getJobLogs(id, new URLSearchParams({ page: '1', pageSize: '200' })),
    refetchInterval: () => {
      const status = (queryClient.getQueryData<{ job: Job }>(['job', id])?.job.status ?? '') as string
      return status === 'running' || status === 'pending' ? 2000 : false
    },
  })

  useEffect(() => {
    if (!id) return
    const source = new EventSource(`/api/jobs/${id}/stream`, { withCredentials: true })
    source.addEventListener('job', () => {
      queryClient.invalidateQueries({ queryKey: ['job', id] })
    })
    source.addEventListener('event', () => {
      queryClient.invalidateQueries({ queryKey: ['job-events', id] })
      queryClient.invalidateQueries({ queryKey: ['job-actions', id] })
      queryClient.invalidateQueries({ queryKey: ['job-logs', id] })
    })
    return () => source.close()
  }, [id, queryClient])

  const duplicateMutation = useMutation({
    mutationFn: () => api.executeDuplicateDelete(id),
    onSuccess: () => {
      messageApi.success('重复文件删除已触发')
      queryClient.invalidateQueries({ queryKey: ['job-actions', id] })
      queryClient.invalidateQueries({ queryKey: ['job-events', id] })
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : '执行失败')
    },
  })

  const job = jobQuery.data?.job
  const summary = job?.summary as Record<string, number | string> | undefined
  const events = eventQuery.data?.list ?? []
  const logs = logQuery.data?.list ?? []
  const counts = actionQuery.data?.counts ?? summaryToCounts(summary)
  const tabItems = useMemo(
    () => [
      { key: 'pending', label: '待执行动作' },
      { key: 'executed', label: '已执行动作' },
      { key: 'error', label: '错误' },
      { key: 'events', label: '实时事件' },
      { key: 'logs', label: '运行日志' },
      { key: 'stats', label: '统计' },
      { key: 'raw', label: '原始参数' },
    ],
    [],
  )

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <Row justify="space-between" align="middle">
        <Col><Typography.Title level={3}>扫描详情 #{id}</Typography.Title></Col>
        <Col>
          <Button onClick={() => duplicateMutation.mutate()} loading={duplicateMutation.isPending}>
            执行重复文件删除
          </Button>
        </Col>
      </Row>

      {job?.errorMessage ? (
        <Alert
          type="error"
          showIcon
          message="任务失败"
          description={job.errorMessage}
        />
      ) : null}

      <Card>
        <Descriptions column={3}>
          <Descriptions.Item label="状态"><Tag>{job?.status}</Tag></Descriptions.Item>
          <Descriptions.Item label="来源">{job?.source}</Descriptions.Item>
          <Descriptions.Item label="阶段">{job?.currentPhase}</Descriptions.Item>
          <Descriptions.Item label="任务UUID">{job?.jobUuid}</Descriptions.Item>
          <Descriptions.Item label="扫描UUID">{job?.scanUuid}</Descriptions.Item>
          <Descriptions.Item label="是否执行动作">{job?.hasAction ? '是' : '否'}</Descriptions.Item>
          <Descriptions.Item label="开始时间">{job?.startAt}</Descriptions.Item>
          <Descriptions.Item label="结束时间">{job?.endAt}</Descriptions.Item>
          <Descriptions.Item label="最近心跳">{job?.lastHeartbeatAt}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Row gutter={16}>
        <Col span={6}><Card><Statistic title="已处理文件" value={job?.processedCount ?? 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="总文件数" value={job?.totalCount ?? 0} /></Card></Col>
        <Col span={6}><Card><Statistic title="总待操作数" value={counts.total} /></Card></Col>
        <Col span={6}><Card><Statistic title="重复项数量" value={counts.deleteDuplicate} /></Card></Col>
      </Row>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={(key) => {
            setActiveTab(key)
            setActionPage({ current: 1, pageSize: 20 })
          }}
          items={tabItems.map((item) => ({
            ...item,
            children: renderTab(item.key, {
              id,
              job,
              summary,
              events,
              logs,
              actions: actionQuery.data?.list ?? [],
              counts,
              actionTotal: actionQuery.data?.total ?? 0,
              actionPage,
              pendingActionType,
              setPendingActionType: (value) => {
                setPendingActionType(value)
                setActionPage({ current: 1, pageSize: actionPage.pageSize })
              },
              setActionPage,
            }),
          }))}
        />
      </Card>
    </Space>
  )
}

function renderTab(
  key: string,
  context: {
    id: string
    job?: Job
    summary?: Record<string, number | string>
    events: ScanEvent[]
    logs: ScanJobLog[]
    actions: ScanActionItem[]
    counts: ScanActionCounts
    actionTotal: number
    actionPage: { current: number; pageSize: number }
    pendingActionType: string
    setPendingActionType: (value: string) => void
    setActionPage: (page: { current: number; pageSize: number }) => void
  },
) {
  if (key === 'events') {
    return (
      <Timeline
        items={context.events.map((event) => ({
          children: `${event.createdAt} | ${event.title} | ${event.message}${event.relatedPath ? ` | ${event.relatedPath}` : ''}`,
        }))}
      />
    )
  }

  if (key === 'stats') {
    return <pre className="json-block">{JSON.stringify(context.summary ?? {}, null, 2)}</pre>
  }

  if (key === 'logs') {
    return (
      <pre className="json-block log-block">
        {context.logs.length === 0
          ? '暂无日志'
          : context.logs
              .map((log) => {
                const payload = log.payloadJson ? ` ${log.payloadJson}` : ''
                return `[${log.createdAt}] [${log.level.toUpperCase()}] [${log.phase || 'runtime'}] ${log.message}${payload}`
              })
              .join('\n')}
      </pre>
    )
  }

  if (key === 'raw') {
    return <pre className="json-block">{JSON.stringify(context.job?.scanArgs ?? {}, null, 2)}</pre>
  }

  if (key === 'pending') {
    const columns = getPendingColumns(context.id, context.pendingActionType)
    return (
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Row gutter={16}>
          <Col span={8}><Card><Statistic title="待执行总数" value={context.counts.total} /></Card></Col>
          <Col span={8}><Card><Statistic title="删除类" value={context.counts.delete} /></Card></Col>
          <Col span={8}><Card><Statistic title="移动类" value={context.counts.move} /></Card></Col>
        </Row>
        <Tabs
          activeKey={context.pendingActionType}
          onChange={context.setPendingActionType}
          items={pendingTypeOptions.map((item) => ({
            key: item.key,
            label: `${item.label} (${countForType(context.counts, item.key)})`,
            children: (
              <Table
                rowKey="id"
                columns={columns}
                dataSource={context.actions}
                locale={{ emptyText: <Empty description="暂无待执行动作" /> }}
                scroll={{ x: true }}
                pagination={{
                  total: context.actionTotal,
                  current: context.actionPage.current,
                  pageSize: context.actionPage.pageSize,
                  onChange: (current, pageSize) => context.setActionPage({ current, pageSize }),
                }}
              />
            ),
          }))}
        />
      </Space>
    )
  }

  return (
    <Table
      rowKey="id"
      columns={historyActionColumns}
      dataSource={context.actions}
      scroll={{ x: true }}
      pagination={{
        total: context.actionTotal,
        current: context.actionPage.current,
        pageSize: context.actionPage.pageSize,
        onChange: (current, pageSize) => context.setActionPage({ current, pageSize }),
      }}
    />
  )
}

function getPendingColumns(id: string, actionType: string): ColumnsType<ScanActionItem> {
  switch (actionType) {
    case 'move':
      return [
        photoColumn(id, '照片'),
        textColumn('文件名', (record) => record.detail?.fileName),
        textColumn('当前位置', (record) => record.detail?.currentPath),
        textColumn('移动位置', (record) => record.detail?.targetPath),
        textColumn('目录日期', (record) => record.detail?.dirDate),
        textColumn('文件名提取日期', (record) => record.detail?.fileNameDate),
        textColumn('照片拍摄日期', (record) => record.detail?.shootDate),
        textColumn('照片拍摄日志（原始）', (record) => record.detail?.shootDateRaw),
        textColumn('最小日期', (record) => record.detail?.minDate),
      ]
    case 'modify_time':
      return [
        photoColumn(id, '照片'),
        textColumn('文件名', (record) => record.detail?.fileName),
        textColumn('目录日期', (record) => record.detail?.dirDate),
        textColumn('文件名提取日期', (record) => record.detail?.fileNameDate),
        textColumn('照片拍摄日期', (record) => record.detail?.shootDate),
        textColumn('照片拍摄日志（原始）', (record) => record.detail?.shootDateRaw),
        textColumn('最小日期', (record) => record.detail?.minDate),
      ]
    case 'delete_duplicate':
      return [
        pairPhotoColumn(id, '照片A', 'pair_a'),
        textColumn('照片A文件名', (record) => record.pair?.photoA.fileName),
        textColumn('照片A位置', (record) => record.pair?.photoA.path),
        pairPhotoColumn(id, '照片B', 'pair_b'),
        textColumn('照片B文件名', (record) => record.pair?.photoB.fileName),
        textColumn('照片B位置', (record) => record.pair?.photoB.path),
      ]
    case 'rename':
      return [
        photoColumn(id, '照片'),
        textColumn('原文件名', (record) => record.detail?.fileName),
        textColumn('新文件名', (record) => record.detail?.targetFileName),
        textColumn('当前位置', (record) => record.detail?.currentPath),
        textColumn('照片拍摄日期', (record) => record.detail?.shootDate),
        textColumn('照片拍摄日志（原始）', (record) => record.detail?.shootDateRaw),
      ]
    case 'delete':
    default:
      return [
        textColumn('文件名', (record) => record.detail?.fileName ?? record.sourcePath.split('/').pop()),
        textColumn('位置', (record) => record.detail?.currentPath ?? record.sourcePath),
      ]
  }
}

function photoColumn(id: string, title: string): ColumnsType<ScanActionItem>[number] {
  return {
    title,
    width: 112,
    render: (_, record) => <PreviewImage id={id} itemId={record.id} slot={record.detail?.previewSlot ?? 'source'} alt={record.detail?.fileName ?? '照片'} />,
  }
}

function pairPhotoColumn(id: string, title: string, slot: 'pair_a' | 'pair_b'): ColumnsType<ScanActionItem>[number] {
  return {
    title,
    width: 112,
    render: (_, record) => {
      const label = slot === 'pair_a' ? record.pair?.photoA.fileName : record.pair?.photoB.fileName
      return <PreviewImage id={id} itemId={record.id} slot={slot} alt={label ?? '照片'} />
    },
  }
}

function textColumn(title: string, getter: (record: ScanActionItem) => string | undefined): ColumnsType<ScanActionItem>[number] {
  return {
    title,
    ellipsis: true,
    render: (_, record) => getter(record) || '-',
  }
}

function PreviewImage({ id, itemId, slot, alt }: { id: string; itemId: number; slot: string; alt: string }) {
  return (
    <Image
      className="action-preview"
      width={72}
      height={72}
      src={api.getJobActionPreviewUrl(id, itemId, slot)}
      alt={alt}
      fallback="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='72' height='72'%3E%3Crect width='72' height='72' fill='%23eef2ef'/%3E%3Ctext x='36' y='40' text-anchor='middle' fill='%23839590' font-size='12'%3E%E6%97%A0%E5%9B%BE%3C/text%3E%3C/svg%3E"
    />
  )
}

function countForType(counts: ScanActionCounts, type: string) {
  switch (type) {
    case 'delete':
      return counts.delete
    case 'move':
      return counts.move
    case 'modify_time':
      return counts.modifyTime
    case 'delete_duplicate':
      return counts.deleteDuplicate
    case 'rename':
      return counts.rename
    default:
      return 0
  }
}

function summaryToCounts(summary?: Record<string, number | string>): ScanActionCounts {
  const deleteCount = numberFromSummary(summary?.deleteFileCnt)
  const moveCount = numberFromSummary(summary?.moveFileCnt)
  const modifyCount = numberFromSummary(summary?.modifyDateFileCnt)
  const duplicateCount = numberFromSummary(summary?.dumpFileCnt)
  const renameCount = numberFromSummary(summary?.renameFileCnt)
  return {
    delete: deleteCount,
    move: moveCount,
    modifyTime: modifyCount,
    deleteDuplicate: duplicateCount,
    rename: renameCount,
    total: deleteCount + moveCount + modifyCount + duplicateCount + renameCount,
  }
}

function numberFromSummary(value: number | string | undefined) {
  if (typeof value === 'number') return value
  if (typeof value === 'string') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}
