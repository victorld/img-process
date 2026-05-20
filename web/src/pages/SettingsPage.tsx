import { useQuery } from '@tanstack/react-query'
import { Alert, Card, Descriptions, Space, Typography } from 'antd'
import { api } from '../api'
import { configSections } from '../configSections'

function formatConfigValue(value: unknown) {
  if (typeof value === 'boolean') {
    return value ? '启用' : '停用'
  }
  if (value === undefined || value === null || value === '') {
    return '-'
  }
  return String(value)
}

export function SettingsPage() {
  const statusQuery = useQuery({
    queryKey: ['system-status'],
    queryFn: api.getSystemStatus,
  })

  const config = statusQuery.data?.config

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Typography.Title level={3}>系统设置</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="当前页面展示的是服务运行时实际生效配置"
        description={`配置来源：${statusQuery.data?.configFile || '-'}`}
      />
      {configSections.map((section) => (
        <Card key={section.key} title={section.title}>
          <Descriptions column={{ xs: 1, sm: 1, md: 2, xl: 3 }}>
            {section.fields.map((field) => (
              <Descriptions.Item key={field.key} label={`${field.label} (${field.key})`}>
                <Typography.Text className="settings-value">
                  {formatConfigValue(config?.[section.key]?.[field.key])}
                </Typography.Text>
              </Descriptions.Item>
            ))}
          </Descriptions>
        </Card>
      ))}
    </Space>
  )
}
