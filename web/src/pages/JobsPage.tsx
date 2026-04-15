import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Button,
  Card,
  Checkbox,
  Col,
  Drawer,
  Form,
  Input,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import type { Job } from '../types'
import { formatDateTime } from '../utils/dateTime'
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

  const columns: ColumnsType<Job> = useMemo(
    () => [
      { title: '任务ID', dataIndex: 'id', width: 76 },
      { title: '任务UUID', dataIndex: 'jobUuid', width: 180 },
      { title: '状态', dataIndex: 'status', width: 88, render: (value) => <Tag>{formatJobStatus(String(value ?? ''))}</Tag> },
      { title: '来源', dataIndex: 'source', width: 72, render: (value) => formatJobSource(String(value ?? '')) },
      { title: '总文件夹数', dataIndex: 'totalFolderCount', width: 96, render: (value) => Number(value ?? 0) },
      { title: '总文件数', dataIndex: 'totalCount', width: 88, render: (value) => Number(value ?? 0) },
      { title: '待执行动作', dataIndex: 'pendingActionCount', width: 96, render: (value) => Number(value ?? 0) },
      { title: '已执行动作', dataIndex: 'executedActionCount', width: 96, render: (value) => Number(value ?? 0) },
      { title: '开始时间', dataIndex: 'startAt', width: 132, render: (value) => formatDateTime(value) },
      {
        title: '操作',
        dataIndex: 'id',
        width: 84,
        render: (_, record) => <Link to={`/jobs/${record.id}`}>查看详情</Link>,
      },
    ],
    [],
  )

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <Row justify="space-between" align="middle">
        <Col>
          <Typography.Title level={3}>扫描历史</Typography.Title>
        </Col>
        <Col>
          <Button
            type="primary"
            onClick={async () => {
              const result = await systemStatusQuery.refetch()
              const defaults = result.data?.server.scanDefaults as ScanFormValues | undefined
              form.resetFields()
              form.setFieldsValue(defaults ?? {})
              setDrawerOpen(true)
            }}
          >
            新建扫描
          </Button>
        </Col>
      </Row>

      <Card>
        <Row gutter={12}>
          <Col span={6}>
            <Select
              placeholder="状态"
              allowClear
              style={{ width: '100%' }}
              onChange={(value) => setFilters((prev) => ({ ...prev, status: value ?? '', page: 1 }))}
              options={['pending', 'running', 'succeeded', 'failed', 'interrupted', 'skipped'].map((value) => ({ label: formatJobStatus(value), value }))}
            />
          </Col>
          <Col span={6}>
            <Select
              placeholder="来源"
              allowClear
              style={{ width: '100%' }}
              onChange={(value) => setFilters((prev) => ({ ...prev, source: value ?? '', page: 1 }))}
              options={['manual', 'schedule'].map((value) => ({ label: formatJobSource(value), value }))}
            />
          </Col>
          <Col span={8}>
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
