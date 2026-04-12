import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button, Drawer, Form, Input, InputNumber, Row, Col, Select, Space, Switch, Table, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useState } from 'react'
import { api } from '../api'
import type { Schedule } from '../types'

type ScheduleFormValues = {
  name: string
  enabled: boolean
  timezone: string
  mode: string
  cronExpr?: string
  hour?: number
  minute?: number
  dayOfMonth?: number
  weekdays?: number[]
  startPath?: string
  startPathBak?: string
  md5Show?: boolean
}

export function SchedulesPage() {
  const [messageApi, contextHolder] = message.useMessage()
  const queryClient = useQueryClient()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [editing, setEditing] = useState<Schedule | null>(null)
  const [form] = Form.useForm<ScheduleFormValues>()

  const schedulesQuery = useQuery({
    queryKey: ['schedules'],
    queryFn: () => api.getSchedules(new URLSearchParams({ page: '1', pageSize: '50' })),
  })

  const upsertMutation = useMutation({
    mutationFn: async (values: ScheduleFormValues) => {
      const payload = {
        name: values.name,
        enabled: values.enabled,
        timezone: values.timezone,
        mode: values.mode,
        cronExpr: values.cronExpr,
        scheduleConfig: JSON.stringify({
          hour: values.hour ?? 0,
          minute: values.minute ?? 0,
          dayOfMonth: values.dayOfMonth ?? 1,
          weekdays: values.weekdays ?? [],
        }),
        scanArgs: {
          startPath: values.startPath,
          startPathBak: values.startPathBak,
          md5Show: values.md5Show,
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
    { title: '名称', dataIndex: 'name' },
    { title: '模式', dataIndex: 'mode', width: 120 },
    { title: 'Cron', dataIndex: 'cronExpr' },
    { title: '时区', dataIndex: 'timezone', width: 140 },
    { title: '启用', dataIndex: 'enabled', width: 90, render: (value) => (value ? '是' : '否') },
    { title: '下次执行', dataIndex: 'nextRunAt', width: 180 },
    { title: '最近状态', dataIndex: 'lastJobStatus', width: 120 },
    {
      title: '操作',
      width: 280,
      render: (_, record) => (
        <Space>
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
        </Space>
      ),
    },
  ]

  const openEdit = (record: Schedule) => {
    setEditing(record)
    setDrawerOpen(true)
    form.setFieldsValue({
      name: record.name,
      enabled: record.enabled,
      timezone: record.timezone,
      mode: record.mode,
      cronExpr: record.cronExpr,
    })
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <Row justify="space-between">
        <Typography.Title level={3}>计划任务</Typography.Title>
        <Button type="primary" onClick={() => {
          setEditing(null)
          form.resetFields()
          form.setFieldsValue({ enabled: true, timezone: 'Asia/Shanghai', mode: 'daily', hour: 2, minute: 0, md5Show: true })
          setDrawerOpen(true)
        }}>
          新建计划
        </Button>
      </Row>
      <Table rowKey="id" columns={columns} dataSource={schedulesQuery.data?.list ?? []} />
      <Drawer
        title={editing ? '编辑计划' : '新建计划'}
        width={520}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        extra={<Button type="primary" onClick={() => form.submit()} loading={upsertMutation.isPending}>保存</Button>}
      >
        <Form layout="vertical" form={form} onFinish={(values) => upsertMutation.mutate(values)}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="timezone" label="时区"><Input /></Form.Item>
          <Form.Item name="mode" label="模式"><Select options={[
            { label: '每天', value: 'daily' },
            { label: '每周', value: 'weekly' },
            { label: '每月', value: 'monthly' },
            { label: '自定义 Cron', value: 'custom' },
          ]} /></Form.Item>
          <Row gutter={12}>
            <Col span={12}><Form.Item name="hour" label="小时"><InputNumber min={0} max={23} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={12}><Form.Item name="minute" label="分钟"><InputNumber min={0} max={59} style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
          <Form.Item name="dayOfMonth" label="每月日期"><InputNumber min={1} max={31} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="weekdays" label="每周几"><Select mode="multiple" options={[
            { label: '周日', value: 0 },
            { label: '周一', value: 1 },
            { label: '周二', value: 2 },
            { label: '周三', value: 3 },
            { label: '周四', value: 4 },
            { label: '周五', value: 5 },
            { label: '周六', value: 6 },
          ]} /></Form.Item>
          <Form.Item name="cronExpr" label="高级 Cron"><Input placeholder="custom 模式时生效" /></Form.Item>
          <Form.Item name="startPath" label="扫描目录"><Input /></Form.Item>
          <Form.Item name="startPathBak" label="备份目录"><Input /></Form.Item>
          <Form.Item name="md5Show" label="计算重复文件" valuePropName="checked"><Switch /></Form.Item>
        </Form>
      </Drawer>
    </Space>
  )
}
