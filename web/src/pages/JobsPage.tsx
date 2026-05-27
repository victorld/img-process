import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Button,
  Card,
  Checkbox,
  Col,
  Drawer,
  Form,
  Input,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { Job } from '../types'
import { formatDateTime, formatDurationBetween } from '../utils/dateTime'
import { formatJobSource, formatJobStatus } from '../utils/displayText'
import { NO_SCROLL_TABLE_CLASS } from '../utils/table'

type ScanFormValues = {
  startPath?: string
  startPathBak?: string
  deleteShow?: boolean
  moveFileShow?: boolean
  modifyDateShow?: boolean
  renameFileShow?: boolean
  md5Show?: boolean
  deleteAction?: boolean
  moveFileAction?: boolean
  modifyDateAction?: boolean
  renameFileAction?: boolean
}

function renderJobIdWithUuid(record: Job) {
  return (
    <Tooltip title={record.jobUuid || '无任务UUID'}>
      <Typography.Text>{record.id}</Typography.Text>
    </Tooltip>
  )
}

export function JobsPage() {
  const [messageApi, contextHolder] = message.useMessage()
  const queryClient = useQueryClient()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [filters, setFilters] = useState({ status: '', source: '', keyword: '', page: 1, pageSize: 10 })
  const [form] = Form.useForm<ScanFormValues>()

  const systemStatusQuery = useQuery({
    queryKey: ['system-status'],
    queryFn: api.getSystemStatus,
  })

  const jobsQuery = useQuery({
    queryKey: ['jobs', filters],
    queryFn: () =>
      api.getJobs(
        new URLSearchParams({
          page: String(filters.page),
          pageSize: String(filters.pageSize),
          status: filters.status,
          source: filters.source,
          keyword: filters.keyword,
        }),
      ),
  })

  const createMutation = useMutation({
    mutationFn: (values: ScanFormValues) => api.createJob(values as Record<string, unknown>),
    onSuccess: () => {
      messageApi.success('扫描任务已创建')
      setDrawerOpen(false)
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : '创建任务失败')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.deleteJob(id),
    onSuccess: () => {
      messageApi.success('扫描任务已删除')
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : '删除任务失败')
    },
  })

  const columns: ColumnsType<Job> = useMemo(
    () => [
      { title: '任务ID', dataIndex: 'id', width: 76, render: (_: unknown, record: Job) => renderJobIdWithUuid(record) },
      { title: '状态', dataIndex: 'status', width: 88, render: (value) => <Tag>{formatJobStatus(String(value ?? ''))}</Tag> },
      { title: '来源', dataIndex: 'source', width: 72, render: (value) => formatJobSource(String(value ?? '')) },
      { title: '总文件夹数', dataIndex: 'totalFolderCount', width: 96, render: (value) => Number(value ?? 0) },
      { title: '总文件数', dataIndex: 'totalCount', width: 88, render: (value) => Number(value ?? 0) },
      { title: '备份总文件夹数', dataIndex: 'totalBackupFolderCount', width: 120, render: (value) => Number(value ?? 0) },
      { title: '备份总文件数', dataIndex: 'totalBackupFileCount', width: 108, render: (value) => Number(value ?? 0) },
      { title: '待执行动作', dataIndex: 'pendingActionCount', width: 96, render: (value) => Number(value ?? 0) },
      { title: '已执行动作', dataIndex: 'executedActionCount', width: 96, render: (value) => Number(value ?? 0) },
      { title: '备份多', dataIndex: 'backupExtraCount', width: 80, render: (value) => Number(value ?? 0) },
      { title: '备份缺', dataIndex: 'backupMissingCount', width: 80, render: (value) => Number(value ?? 0) },
      { title: '开始时间', dataIndex: 'startAt', width: 132, render: (value) => formatDateTime(value as string | null | undefined) },
      { title: '执行时长', dataIndex: 'endAt', width: 96, render: (_: unknown, record: Job) => formatDurationBetween(record.startAt, record.endAt ?? record.lastHeartbeatAt) },
      {
        title: '操作',
        dataIndex: 'id',
        width: 132,
        render: (_: unknown, record: Job) => (
          <Space size={8}>
            <Link to={`/jobs/${record.id}`}>查看详情</Link>
            <Popconfirm
              title="确认删除任务"
              description="会删除该任务及其事件、日志和动作明细，是否继续？"
              okText="删除"
              cancelText="取消"
              okButtonProps={{ danger: true, loading: deleteMutation.isPending }}
              onConfirm={() => deleteMutation.mutate(record.id)}
            >
              <Button
                danger
                size="small"
                type="link"
                disabled={record.status === 'pending' || record.status === 'running'}
              >
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deleteMutation],
  )

  return (
    <Space className="page-stack jobs-page" direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <div className="page-head">
        <div>
          <div className="page-kicker">Scan Jobs</div>
          <Typography.Title level={2}>扫描历史</Typography.Title>
          <Typography.Paragraph>查看扫描来源、文件统计、备份差异和可执行动作。危险删除会进入二次确认。</Typography.Paragraph>
        </div>
        <div className="row">
          <Button
            type="primary"
            onClick={async () => {
              const result = await systemStatusQuery.refetch()
              const defaults = result.data?.server.scanDefaults as ScanFormValues | undefined
              form.resetFields()
              form.setFieldsValue({ ...defaults, modifyDateShow: false })
              setDrawerOpen(true)
            }}
          >
            新建扫描
          </Button>
        </div>
      </div>

      <Card className="job-filter-card">
        <Row className="filters job-filters" gutter={12}>
          <Col xs={24} md={6}>
            <Select
              placeholder="状态"
              allowClear
              style={{ width: '100%' }}
              onChange={(value) => setFilters((prev) => ({ ...prev, status: value ?? '', page: 1 }))}
              options={['pending', 'running', 'succeeded', 'failed', 'interrupted', 'skipped'].map((value) => ({ label: formatJobStatus(value), value }))}
            />
          </Col>
          <Col xs={24} md={6}>
            <Select
              placeholder="来源"
              allowClear
              style={{ width: '100%' }}
              onChange={(value) => setFilters((prev) => ({ ...prev, source: value ?? '', page: 1 }))}
              options={['manual', 'schedule'].map((value) => ({ label: formatJobSource(value), value }))}
            />
          </Col>
          <Col xs={24} md={8}>
            <Input.Search
              placeholder="任务UUID / 错误关键词"
              allowClear
              onSearch={(value) => setFilters((prev) => ({ ...prev, keyword: value, page: 1 }))}
            />
          </Col>
        </Row>
      </Card>

      <div className={NO_SCROLL_TABLE_CLASS}>
        <Table
          rowKey="id"
          loading={jobsQuery.isLoading}
          tableLayout="fixed"
          columns={columns}
          dataSource={jobsQuery.data?.list ?? []}
          pagination={{
            total: jobsQuery.data?.total ?? 0,
            current: filters.page,
            pageSize: filters.pageSize,
            onChange: (page, pageSize) => setFilters((prev) => ({ ...prev, page, pageSize })),
          }}
        />
      </div>

      <Drawer
        title="新建扫描"
        width={520}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        extra={
          <Button type="primary" onClick={() => form.submit()} loading={createMutation.isPending}>
            启动扫描
          </Button>
        }
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="startPath" label="扫描目录">
            <Input />
          </Form.Item>
          <Form.Item name="startPathBak" label="备份目录">
            <Input />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}><Form.Item name="deleteShow" valuePropName="checked"><Checkbox>显示待删除</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="moveFileShow" valuePropName="checked"><Checkbox>显示待移动</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="modifyDateShow" valuePropName="checked"><Checkbox>显示改时间</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="renameFileShow" valuePropName="checked"><Checkbox>显示重命名</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="md5Show" valuePropName="checked"><Checkbox>计算重复文件</Checkbox></Form.Item></Col>
          </Row>
          <Typography.Text strong>实际执行</Typography.Text>
          <Row gutter={12} style={{ marginTop: 12 }}>
            <Col span={12}><Form.Item name="deleteAction" valuePropName="checked"><Checkbox>删除文件</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="moveFileAction" valuePropName="checked"><Checkbox>移动文件</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="modifyDateAction" valuePropName="checked"><Checkbox>修改时间</Checkbox></Form.Item></Col>
            <Col span={12}><Form.Item name="renameFileAction" valuePropName="checked"><Checkbox>重命名文件</Checkbox></Form.Item></Col>
          </Row>
        </Form>
      </Drawer>
    </Space>
  )
}
