import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Col, Descriptions, Row, Space, Statistic, Table, Tabs, Tag, Timeline, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api'
import type { Job, ScanActionItem, ScanEvent, ScanJobLog } from '../types'

const actionColumns: ColumnsType<ScanActionItem> = [
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
  const [actionPage, setActionPage] = useState({ current: 1, pageSize: 20 })
  const actionTabs = ['pending', 'executed', 'error', 'duplicate']

  const jobQuery = useQuery({
    queryKey: ['job', id],
    queryFn: () => api.getJob(id),
    refetchInterval: (query) => {
      const status = (query.state.data?.job.status ?? '') as string
      return status === 'running' || status === 'pending' ? 2000 : false
    },
  })

  const actionQuery = useQuery({
    queryKey: ['job-actions', id, activeTab, actionPage],
    enabled: actionTabs.includes(activeTab),
    queryFn: () =>
      api.getJobActionItems(
        id,
        new URLSearchParams({
          tab: activeTab,
          page: String(actionPage.current),
          pageSize: String(actionPage.pageSize),
        }),
      ),
  })

  const eventQuery = useQuery({
    queryKey: ['job-events', id],
    queryFn: () => api.getJobEvents(id, new URLSearchParams({ page: '1', pageSize: '50' })),
    refetchInterval: 3000,
  })

  const logQuery = useQuery({
    queryKey: ['job-logs', id],
    queryFn: () => api.getJobLogs(id, new URLSearchParams({ page: '1', pageSize: '200' })),
    refetchInterval: (query) => {
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
  const tabItems = useMemo(
    () => [
      { key: 'pending', label: '待执行动作' },
      { key: 'executed', label: '已执行动作' },
      { key: 'error', label: '错误' },
      { key: 'duplicate', label: '重复文件' },
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
        <Col span={6}><Card><Statistic title="待删数量" value={Number(summary?.deleteFileCnt ?? 0)} /></Card></Col>
        <Col span={6}><Card><Statistic title="待移动数量" value={Number(summary?.moveFileCnt ?? 0)} /></Card></Col>
      </Row>

      <Card>
        <Tabs activeKey={activeTab} onChange={(key) => setActiveTab(key)} items={tabItems.map((item) => ({ ...item, children: renderTab(item.key, {
          job,
          summary,
          events,
          logs,
          actions: actionQuery.data?.list ?? [],
          actionTotal: actionQuery.data?.total ?? 0,
          actionPage,
          setActionPage,
        }) }))} />
      </Card>
    </Space>
  )
}

function renderTab(
  key: string,
  context: {
    job?: Job
    summary?: Record<string, number | string>
    events: ScanEvent[]
    logs: ScanJobLog[]
    actions: ScanActionItem[]
    actionTotal: number
    actionPage: { current: number; pageSize: number }
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

  return (
    <Table
      rowKey="id"
      columns={actionColumns}
      dataSource={context.actions}
      pagination={{
        total: context.actionTotal,
        current: context.actionPage.current,
        pageSize: context.actionPage.pageSize,
        onChange: (current, pageSize) => context.setActionPage({ current, pageSize }),
      }}
    />
  )
}
