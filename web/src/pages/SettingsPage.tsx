import { FolderOpenOutlined, SaveOutlined } from '@ant-design/icons'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Form, Input, InputNumber, Space, Switch, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { api } from '../api'
import { configSections, type ConfigField, type ConfigSection, type ConfigSectionKey } from '../configSections'

type SettingValue = string | number | boolean | undefined
type SettingsFormValues = Record<string, Record<string, SettingValue>>
type SettingsPayload = Record<string, Record<string, string | number | boolean>>
type PathPickerTarget = {
  sectionKey: ConfigSectionKey
  fieldKey: string
  path?: string
}
type PathInputProps = {
  value?: string
  disabled?: boolean
  onChange?: (value: string) => void
  onPickPath?: () => void
  loading?: boolean
}

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

function PathInput({ value, disabled, onChange, onPickPath, loading }: PathInputProps) {
  return (
    <Space.Compact style={{ width: '100%' }}>
      <Input value={value} disabled={disabled} onChange={(event) => onChange?.(event.target.value)} />
      <Button icon={<FolderOpenOutlined />} disabled={disabled} loading={loading} onClick={onPickPath}>
        选择目录
      </Button>
    </Space.Compact>
  )
}

function renderField(field: ConfigField, readonly = false, onPickPath?: () => void, pickingPath = false) {
  if (field.valueType === 'boolean') {
    return <Switch checkedChildren="启用" unCheckedChildren="停用" disabled={readonly} />
  }
  if (field.valueType === 'number') {
    return <InputNumber style={{ width: '100%' }} precision={0} disabled={readonly} />
  }
  if (field.valueType === 'path') {
    return <PathInput disabled={readonly} loading={pickingPath} onPickPath={onPickPath} />
  }
  if (field.secret) {
    return <Input.Password autoComplete="new-password" disabled={readonly} />
  }
  return <Input disabled={readonly} />
}

const fallbackReadonlySections = ['database', 'server']

function fieldItem(
  section: ConfigSection,
  field: ConfigField,
  readonly: boolean,
  openPathPicker: (target: PathPickerTarget) => void,
  pickingPathKey?: string,
) {
  const openPicker = () => openPathPicker({ sectionKey: section.key, fieldKey: field.key })
  const fieldPathKey = `${section.key}.${field.key}`
  return (
    <Form.Item
      key={field.key}
      name={[section.key, field.key]}
      label={`${field.label} (${field.key})`}
      valuePropName={field.valueType === 'boolean' ? 'checked' : 'value'}
    >
      {renderField(field, readonly, openPicker, pickingPathKey === fieldPathKey)}
    </Form.Item>
  )
}

function renderScanArgsSection(
  section: ConfigSection,
  readonly: boolean,
  openPathPicker: (target: PathPickerTarget) => void,
  pickingPathKey?: string,
) {
  const groups = [
    section.fields.filter((field) => field.group === 'path'),
    section.fields.filter((field) => field.group === 'show'),
    section.fields.filter((field) => field.group === 'action'),
  ]
  return (
    <div className="settings-scan-groups">
      {groups.map((fields, index) => (
        <div key={index} className={index === 0 ? 'settings-scan-path-row' : 'settings-scan-grid-row'}>
          {fields.map((field) => fieldItem(section, field, readonly, openPathPicker, pickingPathKey))}
        </div>
      ))}
    </div>
  )
}

function renderSection(
  section: ConfigSection,
  readonlySections: Set<string>,
  openPathPicker: (target: PathPickerTarget) => void,
  pickingPathKey?: string,
) {
  const readonly = readonlySections.has(section.key)
  return (
    <Card key={section.key} title={section.title} className="settings-section">
      {section.description ? <Typography.Paragraph type="secondary">{section.description}</Typography.Paragraph> : null}
      {section.key === 'scanArgs' ? (
        renderScanArgsSection(section, readonly, openPathPicker, pickingPathKey)
      ) : (
        <div className="settings-form-grid">
          {section.fields.map((field) => fieldItem(section, field, readonly, openPathPicker, pickingPathKey))}
        </div>
      )}
    </Card>
  )
}

export function SettingsPage() {
  const [messageApi, contextHolder] = message.useMessage()
  const queryClient = useQueryClient()
  const [form] = Form.useForm<SettingsFormValues>()
  const [pickingPathKey, setPickingPathKey] = useState<string>()

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
  const openPathPicker = async (target: PathPickerTarget) => {
    const currentPath = form.getFieldValue([target.sectionKey, target.fieldKey])
    const nextPath = typeof currentPath === 'string' && currentPath.trim() ? currentPath.trim() : undefined
    const section = configSections.find((item) => item.key === target.sectionKey)
    const field = section?.fields.find((item) => item.key === target.fieldKey)
    const pathKey = `${target.sectionKey}.${target.fieldKey}`
    setPickingPathKey(pathKey)
    try {
      const result = await api.selectSystemDirectory(nextPath, field ? `选择${field.label}` : '选择目录')
      form.setFieldValue([target.sectionKey, target.fieldKey], result.path)
    } catch (error) {
      const text = error instanceof Error ? error.message : '选择目录失败'
      if (text.includes('取消')) {
        messageApi.info('已取消选择目录')
      } else {
        messageApi.error(text)
      }
    } finally {
      setPickingPathKey(undefined)
    }
  }

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
          {configSections.map((section) => renderSection(section, readonlySections, openPathPicker, pickingPathKey))}
        </Space>
      </Form>
    </Space>
  )
}
