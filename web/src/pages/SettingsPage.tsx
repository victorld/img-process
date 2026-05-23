import { SaveOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Form, Input, InputNumber, Space, Switch, Typography, message } from 'antd'
import { useEffect } from 'react'
import { api } from '../api'
import { configSections, type ConfigField, type ConfigSection } from '../configSections'

type SettingValue = string | number | boolean | undefined
type SettingsFormValues = Record<string, Record<string, SettingValue>>
type SettingsPayload = Record<string, Record<string, string | number | boolean>>

function normalizeConfig(config?: SettingsFormValues): SettingsFormValues {
  const values: SettingsFormValues = {}
  configSections.forEach((section) => {
    values[section.key] = {}
    section.fields.forEach((field) => {
      values[section.key][field.key] = config?.[section.key]?.[field.key]
    })
  })
  return values
}

function normalizeFieldValue(field: ConfigField, value: SettingValue): string | number | boolean {
  if (field.valueType === 'boolean') {
    return Boolean(value)
  }
  if (field.valueType === 'number') {
    if (typeof value === 'number') return value
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return typeof value === 'string' ? value : ''
}

function buildPayload(values: SettingsFormValues, readonlySections: Set<string>): SettingsPayload {
  const payload: SettingsPayload = {}
  configSections.forEach((section) => {
    if (readonlySections.has(section.key)) {
      return
    }
    payload[section.key] = {}
    section.fields.forEach((field) => {
      payload[section.key][field.key] = normalizeFieldValue(field, values?.[section.key]?.[field.key])
    })
  })
  return payload
}

function renderField(field: ConfigField, readonly = false) {
  if (field.valueType === 'boolean') {
    return <Switch checkedChildren="启用" unCheckedChildren="停用" disabled={readonly} />
  }
  if (field.valueType === 'number') {
    return <InputNumber style={{ width: '100%' }} precision={0} disabled={readonly} />
  }
  if (field.secret) {
    return <Input.Password autoComplete="new-password" disabled={readonly} />
  }
  return <Input disabled={readonly} />
}

const fallbackReadonlySections = ['database', 'server']

function renderSection(section: ConfigSection, readonlySections: Set<string>) {
  const readonly = readonlySections.has(section.key)
  return (
    <Card key={section.key} title={section.title} className="settings-section">
      <div className="settings-form-grid">
        {section.fields.map((field) => (
          <Form.Item
            key={field.key}
            name={[section.key, field.key]}
            label={`${field.label} (${field.key})`}
            valuePropName={field.valueType === 'boolean' ? 'checked' : 'value'}
          >
            {renderField(field, readonly)}
          </Form.Item>
        ))}
      </div>
    </Card>
  )
}

export function SettingsPage() {
  const [messageApi, contextHolder] = message.useMessage()
  const queryClient = useQueryClient()
  const [form] = Form.useForm<SettingsFormValues>()

  const statusQuery = useQuery({
    queryKey: ['system-status'],
    queryFn: api.getSystemStatus,
  })
  const readonlySections = new Set(statusQuery.data?.readonlySections ?? fallbackReadonlySections)

  useEffect(() => {
    if (statusQuery.data?.config) {
      form.setFieldsValue(normalizeConfig(statusQuery.data.config as SettingsFormValues))
    }
  }, [form, statusQuery.data?.config])

  const saveMutation = useMutation({
    mutationFn: async (values: SettingsFormValues) => api.updateSystemSettings(buildPayload(values, readonlySections)),
    onSuccess: (result) => {
      messageApi.success('设置已保存')
      queryClient.setQueryData(['system-status'], {
        ...statusQuery.data,
        ...result,
      })
      queryClient.invalidateQueries({ queryKey: ['system-status'] })
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : '保存失败')
    },
  })

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <div className="settings-header">
        <Typography.Title level={3}>系统设置</Typography.Title>
        <Button
          type="primary"
          icon={<SaveOutlined />}
          loading={saveMutation.isPending}
          onClick={() => form.submit()}
        >
          保存设置
        </Button>
      </div>
      <Alert
        type="info"
        showIcon
        message="文件配置只读，运行配置保存到数据库"
        description={`只读配置：database、server；可编辑配置来源：${statusQuery.data?.configSource === 'database' ? '数据库' : '-'}；启动配置文件：${statusQuery.data?.configFile || '-'}`}
      />
      <Form
        form={form}
        layout="vertical"
        onFinish={(values) => saveMutation.mutate(values)}
        disabled={statusQuery.isLoading || saveMutation.isPending}
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {configSections.map((section) => renderSection(section, readonlySections))}
        </Space>
      </Form>
    </Space>
  )
}
