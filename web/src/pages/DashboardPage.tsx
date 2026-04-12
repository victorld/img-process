import { useQuery } from '@tanstack/react-query'
import { Card, Col, List, Row, Space, Statistic, Tag, Typography } from 'antd'
import { Link } from 'react-router-dom'
import { api } from '../api'

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
            <List
              dataSource={jobs}
              renderItem={(item) => (
                <List.Item actions={[<Link key="view" to={`/jobs/${item.id}`}>查看详情</Link>]}>
                  <List.Item.Meta title={item.jobUuid} description={item.currentPhase || 'queued'} />
                  <Tag>{item.status}</Tag>
                </List.Item>
              )}
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
                    description={`${item.mode} / ${item.cronExpr}`}
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
