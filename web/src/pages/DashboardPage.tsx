import { useQuery } from '@tanstack/react-query'
import { Card, Col, List, Row, Space, Statistic, Table, Tag, Tooltip, Typography } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { Job } from '../types'
import { formatDurationBetween } from '../utils/dateTime'
import { formatJobSource, formatJobStatus, formatPhase, formatScheduleMode } from '../utils/displayText'
import { NO_SCROLL_TABLE_CLASS } from '../utils/table'

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
    <Space className="page-stack dashboard-page" direction="vertical" size={16} style={{ width: '100%' }}>
      <div className="page-head">
        <div>
          <div className="page-kicker">Overview</div>
          <Typography.Title level={2}>工作台</Typography.Title>
          <Typography.Paragraph>聚合最近扫描、待处理动作、计划运行和备份差异，帮助先处理风险最高的任务。</Typography.Paragraph>
        </div>
      </div>
      <Row className="stats compact" gutter={16}>
        <Col span={8}>
          <Card className="stat info"><Statistic title="总任务数" value={jobsQuery.data?.total ?? 0} /><div className="hint">全部扫描任务记录</div></Card>
        </Col>
        <Col span={8}>
          <Card className="stat warn"><Statistic title="运行中任务" value={runningCount} /><div className="hint">正在处理的扫描</div></Card>
        </Col>
        <Col span={8}>
          <Card className="stat danger"><Statistic title="失败任务" value={failedCount} /><div className="hint">需查看错误日志</div></Card>
        </Col>
      </Row>
      <Row className="grid-two-row" gutter={[16, 16]}>
        <Col span={14}>
          <Card title="最近任务">
            <div className={NO_SCROLL_TABLE_CLASS}>
              <Table
                columns={jobColumns}
                dataSource={jobs}
                loading={jobsQuery.isLoading}
                pagination={false}
                rowKey="id"
                size="small"
                tableLayout="fixed"
              />
            </div>
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
