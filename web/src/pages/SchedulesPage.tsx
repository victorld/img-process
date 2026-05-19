import { QuestionCircleOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button, Checkbox, Drawer, Form, Input, InputNumber, Row, Col, Select, Space, Switch, Table, Typography, message, Popover, Tabs } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useState } from 'react'
import { api } from '../api'
import type { Schedule } from '../types'
import { formatDateTime } from '../utils/dateTime'
import { formatJobStatus, formatScheduleMode } from '../utils/displayText'
import { NO_SCROLL_TABLE_CLASS, TABLE_ACTIONS_CLASS } from '../utils/table'

type ScheduleFormValues = {
  name: string
  enabled: boolean
  mode: string
  cronExpr?: string
  hour?: number
  minute?: number
  dayOfMonth?: number
  weekdays?: number[]
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

type ScheduleConfig = {
  hour?: number
  minute?: number
  dayOfMonth?: number
  weekdays?: number[]
}

const defaultScheduleValues: Partial<ScheduleFormValues> = {
  enabled: true,
  mode: 'hourly',
  minute: 0,
  md5Show: true,
}

const cronExamples = [
  { label: '每小时', value: '0 * * * *' },
  { label: '每天 02:30', value: '30 2 * * *' },
  { label: '每周一 03:15', value: '15 3 * * 1' },
  { label: '每月 1 日 04:00', value: '0 4 1 * *' },
  { label: '每 10 分钟', value: '*/10 * * * *' },
]

function parseScheduleConfig(raw?: string): ScheduleConfig {
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw) as ScheduleConfig
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function parseJSONRecord(raw?: string | Record<string, unknown>): Record<string, unknown> {
  if (!raw) return {}
  if (typeof raw === 'object') return raw
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function normalizeScanDefaults(defaults?: Record<string, unknown>): Partial<ScheduleFormValues> {
  const values: Partial<ScheduleFormValues> = {}
  if (!defaults) return values
  if (typeof defaults.startPath === 'string') values.startPath = defaults.startPath
  if (typeof defaults.startPathBak === 'string') values.startPathBak = defaults.startPathBak
  if ('deleteShow' in defaults) values.deleteShow = Boolean(defaults.deleteShow)
  if ('moveFileShow' in defaults) values.moveFileShow = Boolean(defaults.moveFileShow)
  if ('modifyDateShow' in defaults) values.modifyDateShow = Boolean(defaults.modifyDateShow)
  if ('renameFileShow' in defaults) values.renameFileShow = Boolean(defaults.renameFileShow)
  if ('md5Show' in defaults) values.md5Show = Boolean(defaults.md5Show)
  if ('deleteAction' in defaults) values.deleteAction = Boolean(defaults.deleteAction)
  if ('moveFileAction' in defaults) values.moveFileAction = Boolean(defaults.moveFileAction)
  if ('modifyDateAction' in defaults) values.modifyDateAction = Boolean(defaults.modifyDateAction)
  if ('renameFileAction' in defaults) values.renameFileAction = Boolean(defaults.renameFileAction)
  return values
}

function mergeScanValues(defaults?: Record<string, unknown>, raw?: string | Record<string, unknown>): Partial<ScheduleFormValues> {
  return {
    ...normalizeScanDefaults(defaults),
    ...normalizeScanDefaults({ ...defaults, ...parseJSONRecord(raw) }),
  }
}

function clearFieldsForMode(mode?: string): Partial<ScheduleFormValues> {
  switch (mode) {
    case 'hourly':
      return { hour: undefined, dayOfMonth: undefined, weekdays: undefined, cronExpr: undefined }
    case 'daily':
      return { dayOfMonth: undefined, weekdays: undefined, cronExpr: undefined }
    case 'weekly':
      return { dayOfMonth: undefined, cronExpr: undefined }
    case 'monthly':
      return { weekdays: undefined, cronExpr: undefined }
    case 'custom':
      return { hour: undefined, minute: undefined, dayOfMonth: undefined, weekdays: undefined }
    default:
      return {}
  }
}

function buildScheduleConfig(values: ScheduleFormValues): ScheduleConfig {
  switch (values.mode) {
    case 'hourly':
      return { minute: values.minute ?? 0 }
    case 'daily':
      return { hour: values.hour ?? 0, minute: values.minute ?? 0 }
    case 'weekly':
      return { hour: values.hour ?? 0, minute: values.minute ?? 0, weekdays: values.weekdays ?? [] }
    case 'monthly':
      return { hour: values.hour ?? 0, minute: values.minute ?? 0, dayOfMonth: values.dayOfMonth ?? 1 }
    default:
      return {}
  }
}

function buildCronPreview(values: Partial<ScheduleFormValues>): string {
  const minute = values.minute ?? 0
  const hour = values.hour ?? 0
  switch (values.mode) {
    case 'hourly':
      return `${minute} * * * *`
    case 'daily':
      return `${minute} ${hour} * * *`
    case 'weekly':
      return values.weekdays?.length ? `${minute} ${hour} * * ${values.weekdays.join(',')}` : '请选择每周几'
    case 'monthly':
      return `${minute} ${hour} ${values.dayOfMonth ?? 1} * *`
    case 'custom':
      return values.cronExpr?.trim() || '请输入高级 Cron'
    default:
      return '-'
  }
}

export function SchedulesPage() {
  const [messageApi, contextHolder] = message.useMessage()
  const queryClient = useQueryClient()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<Schedule | null>(null)
  const [form] = Form.useForm<ScheduleFormValues>()
  const mode = Form.useWatch('mode', form)
  const formValues = Form.useWatch([], form) ?? {}
  const cronPreview = buildCronPreview(formValues)

  const schedulesQuery = useQuery({
    queryKey: ['schedules'],
    queryFn: () => api.getSchedules(new URLSearchParams({ page: '1', pageSize: '50' })),
  })

  const systemStatusQuery = useQuery({
    queryKey: ['system-status'],
    queryFn: api.getSystemStatus,
  })

  const upsertMutation = useMutation({
    mutationFn: async (values: ScheduleFormValues) => {
      const payload = {
        name: values.name,
        enabled: values.enabled,
        mode: values.mode,
        cronExpr: values.mode === 'custom' ? values.cronExpr : undefined,
        scheduleConfig: JSON.stringify(buildScheduleConfig(values)),
        scanArgs: {
          startPath: values.startPath,
          startPathBak: values.startPathBak,
          deleteShow: values.deleteShow,
          moveFileShow: values.moveFileShow,
          modifyDateShow: values.modifyDateShow,
          renameFileShow: values.renameFileShow,
          md5Show: values.md5Show,
          deleteAction: values.deleteAction,
          moveFileAction: values.moveFileAction,
          modifyDateAction: values.modifyDateAction,
          renameFileAction: values.renameFileAction,
        },
      }
      if (editing) {
        return api.updateSchedule(editing.id, payload)
      }
      return api.createSchedule(payload)
    },
    onSuccess: () => {
      messageApi.success('计划已保存')
      setDrawerOpen(false)
      setEditing(null)
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : '保存失败')
    },
  })

  const columns: ColumnsType<Schedule> = [
    { title: '名称', dataIndex: 'name', width: 120 },
    { title: '模式', dataIndex: 'mode', width: 96, render: (value) => formatScheduleMode(String(value ?? '')) },
    { title: 'Cron', dataIndex: 'cronExpr', width: 160 },
    { title: '时区', dataIndex: 'timezone', width: 104 },
    { title: '启用', dataIndex: 'enabled', width: 64, render: (value) => (value ? '是' : '否') },
    { title: '下次执行', dataIndex: 'nextRunAt', width: 132, render: (value) => formatDateTime(value) },
    { title: '最近状态', dataIndex: 'lastJobStatus', width: 84, render: (value) => formatJobStatus(String(value ?? '')) },
    {
      title: '操作',
      width: 144,
      render: (_, record) => (
        <div className={TABLE_ACTIONS_CLASS}>
          <Button size="small" onClick={() => openEdit(record)}>编辑</Button>
          <Button size="small" onClick={() => api.runSchedule(record.id).then(() => queryClient.invalidateQueries({ queryKey: ['schedules'] }))}>立即执行</Button>
          <Button
            size="small"
            onClick={() =>
              (record.enabled ? api.disableSchedule(record.id) : api.enableSchedule(record.id)).then(() =>
                queryClient.invalidateQueries({ queryKey: ['schedules'] }),
              )
            }
          >
            {record.enabled ? '停用' : '启用'}
          </Button>
          <Button danger size="small" onClick={() => api.deleteSchedule(record.id).then(() => queryClient.invalidateQueries({ queryKey: ['schedules'] }))}>删除</Button>
        </div>
      ),
    },
  ]

  const openEdit = async (record: Schedule) => {
    const scheduleConfig = parseScheduleConfig(record.scheduleConfig)
    const status = systemStatusQuery.data ?? (await systemStatusQuery.refetch()).data
    const scanValues = mergeScanValues(status?.server.scanDefaults, record.scanArgs)
    setEditing(record)
    form.resetFields()
    setDrawerOpen(true)
    form.setFieldsValue({
      name: record.name,
      enabled: record.enabled,
      mode: record.mode,
      cronExpr: record.mode === 'custom' ? record.cronExpr : undefined,
      ...scanValues,
      ...(record.mode !== 'custom' ? { minute: scheduleConfig.minute ?? 0 } : {}),
      ...(['daily', 'weekly', 'monthly'].includes(record.mode) ? { hour: scheduleConfig.hour } : {}),
      ...(record.mode === 'monthly' ? { dayOfMonth: scheduleConfig.dayOfMonth } : {}),
      ...(record.mode === 'weekly' ? { weekdays: scheduleConfig.weekdays } : {}),
    })
  }

  const handleModeChange = (nextMode: string) => {
    form.setFieldsValue({
      ...clearFieldsForMode(nextMode),
      ...(nextMode === 'hourly' ? { minute: form.getFieldValue('minute') ?? 0 } : {}),
      ...(nextMode === 'daily' ? { hour: form.getFieldValue('hour') ?? 2, minute: form.getFieldValue('minute') ?? 0 } : {}),
      ...(nextMode === 'weekly' ? { hour: form.getFieldValue('hour') ?? 2, minute: form.getFieldValue('minute') ?? 0, weekdays: [] } : {}),
      ...(nextMode === 'monthly' ? { hour: form.getFieldValue('hour') ?? 2, minute: form.getFieldValue('minute') ?? 0, dayOfMonth: 1 } : {}),
    })
  }

  const cronLabel = (
    <Space size={4}>
      高级 Cron
      <Popover
        trigger="click"
        title="常用样例"
        content={
          <Space direction="vertical" size={4}>
            {cronExamples.map((item) => (
              <Typography.Text key={item.value}>
                {item.label}：<Typography.Text code>{item.value}</Typography.Text>
              </Typography.Text>
            ))}
          </Space>
        }
      >
        <QuestionCircleOutlined style={{ color: '#1677ff', cursor: 'pointer' }} />
      </Popover>
    </Space>
  )

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <Row justify="space-between">
        <Typography.Title level={3}>计划任务</Typography.Title>
        <Button type="primary" onClick={async () => {
          const status = systemStatusQuery.data ?? (await systemStatusQuery.refetch()).data
          setEditing(null)
          form.resetFields()
          form.setFieldsValue({
            ...defaultScheduleValues,
            ...normalizeScanDefaults(status?.server.scanDefaults),
          })
          setDrawerOpen(true)
        }}>
          新建计划
        </Button>
      </Row>
      <div className={NO_SCROLL_TABLE_CLASS}>
        <Table rowKey="id" tableLayout="fixed" columns={columns} dataSource={schedulesQuery.data?.list ?? []} />
      </div>
      <Drawer
        title={editing ? '编辑计划' : '新建计划'}
        width={520}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        extra={<Button type="primary" onClick={() => form.submit()} loading={upsertMutation.isPending}>保存</Button>}
      >
        <Form layout="vertical" form={form} onFinish={(values) => upsertMutation.mutate(values)}>
          <Tabs
            items={[
              {
                key: 'basic',
                label: '基本信息',
                children: (
                  <>
                    <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
                    <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
                  </>
                ),
              },
              {
                key: 'schedule',
                label: '计划周期',
                children: (
                  <>
                    <Form.Item name="mode" label="模式"><Select onChange={handleModeChange} options={[
                      { label: '每小时', value: 'hourly' },
                      { label: '每天', value: 'daily' },
                      { label: '每周', value: 'weekly' },
                      { label: '每月', value: 'monthly' },
                      { label: '高级 Cron', value: 'custom' },
                    ]} /></Form.Item>
                    {mode !== 'custom' && (
                      <Row gutter={12}>
                        {mode !== 'hourly' && (
                          <Col span={12}><Form.Item name="hour" label="小时"><InputNumber min={0} max={23} style={{ width: '100%' }} /></Form.Item></Col>
                        )}
                        <Col span={mode === 'hourly' ? 24 : 12}><Form.Item name="minute" label="分钟"><InputNumber min={0} max={59} style={{ width: '100%' }} /></Form.Item></Col>
                      </Row>
                    )}
                    {mode === 'monthly' && <Form.Item name="dayOfMonth" label="每月日期"><InputNumber min={1} max={31} style={{ width: '100%' }} /></Form.Item>}
                    {mode === 'weekly' && <Form.Item name="weekdays" label="每周几"><Select mode="multiple" options={[
                      { label: '周日', value: 0 },
                      { label: '周一', value: 1 },
                      { label: '周二', value: 2 },
                      { label: '周三', value: 3 },
                      { label: '周四', value: 4 },
                      { label: '周五', value: 5 },
                      { label: '周六', value: 6 },
                    ]} /></Form.Item>}
                    {mode === 'custom' && <Form.Item name="cronExpr" label={cronLabel}><Input placeholder="例如：30 2 * * *" /></Form.Item>}
                    <Form.Item label="Cron 预览">
                      <Typography.Text code>{cronPreview}</Typography.Text>
                    </Form.Item>
                  </>
                ),
              },
              {
                key: 'scan',
                label: '扫描参数',
                children: (
                  <>
                    <Form.Item name="startPath" label="扫描目录"><Input /></Form.Item>
                    <Form.Item name="startPathBak" label="备份目录"><Input /></Form.Item>
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
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Drawer>
    </Space>
  )
}
