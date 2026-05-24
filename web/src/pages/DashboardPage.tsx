import { useQuery } from '@tanstack/react-query'
import { Card, Col, List, Row, Space, Statistic, Table, Tag, Tooltip, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { Job } from '../types'
import { formatDurationBetween } from '../utils/dateTime'
import { formatJobSource, formatJobStatus, formatPhase, formatScheduleMode } from '../utils/displayText'

function renderJobIdWithUuid(record: Job) {
  return (
    <Tooltip title={record.jobUuid || '无任务UUID'}>
      <Typography.Text>{record.id}</Typography.Text>
    </Tooltip>
  )
}

export function DashboardPage() {
  const jobsQuery = useQuery({
    queryKey: ['dashboard-jobs'],
    queryFn: () => api.getJobs(new URLSearchParams({ page: '1', pageSize: '5' })),
  })
  const schedulesQuery = useQuery({
    queryKey: ['dashboard-schedules'],
    queryFn: () => api.getSchedules(new URLSearchParams({ page: '1', pageSize: '5' })),
  })

  const jobs = jobsQuery.data?.list ?? []
  const schedules = schedulesQuery.data?.list ?? []
  const runningCount = jobs.filter((item) => item.status === 'running').length
  const failedCount = jobs.filter((item) => item.status === 'failed').length
  const jobColumns: ColumnsType<Job> = useMemo(
    () => [
      { title: '任务ID', dataIndex: 'id', width: 96, render: (_: unknown, record: Job) => renderJobIdWithUuid(record) },
      { title: '任务类型', dataIndex: 'source', width: 88, render: (value) => formatJobSource(String(value ?? '')) },
      { title: '阶段', dataIndex: 'currentPhase', width: 108, render: (value) => formatPhase(String(value || 'queued')) },
      { title: '状态', dataIndex: 'status', width: 84, render: (value) => <Tag>{formatJobStatus(String(value ?? ''))}</Tag> },
      {
        title: '运行时间',
        dataIndex: 'endAt',
        width: 100,
        render: (_: unknown, record: Job) => formatDurationBetween(record.startAt, record.endAt ?? record.lastHeartbeatAt),
      },
      {
        title: '操作',
        dataIndex: 'id',
        width: 88,
        render: (_: unknown, record: Job) => <Link to={`/jobs/${record.id}`}>查看详情</Link>,
      },
    ],
    [],
  )

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Typography.Title level={3}>仪表盘</Typography.Title>
      <Row gutter={16}>
        <Col span={8}>
          <Card><Statistic title="最近任务数" value={jobsQuery.data?.total ?? 0} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="运行中任务" value={runningCount} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="最近失败任务" value={failedCount} /></Card>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col span={14}>
          <Card title="最近任务">
            <Table
              columns={jobColumns}
              dataSource={jobs}
              loading={jobsQuery.isLoading}
              pagination={false}
              rowKey="id"
              size="small"
            />
          </Card>
        </Col>
        <Col span={10}>
          <Card title="最近计划任务">
            <List
              dataSource={schedules}
              renderItem={(item) => (
                <List.Item>
                  <List.Item.Meta
                    title={item.name}
                    description={`${formatScheduleMode(item.mode)} / ${item.cronExpr}`}
                  />
                  <Tag color={item.enabled ? 'green' : 'default'}>{item.enabled ? '启用' : '停用'}</Tag>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </Space>
  )
}
