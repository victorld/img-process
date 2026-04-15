import { useQuery } from '@tanstack/react-query'
import { Card, Descriptions, Space, Typography } from 'antd'
import { api } from '../api'

export function SettingsPage() {
  const statusQuery = useQuery({
    queryKey: ['system-status'],
    queryFn: api.getSystemStatus,
  })

  const server = statusQuery.data?.server

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Typography.Title level={3}>系统设置</Typography.Title>
      <Card>
        <Descriptions column={2}>
          <Descriptions.Item label="HTTP 端口">{server?.httpPort}</Descriptions.Item>
          <Descriptions.Item label="扫描目录">{server?.startPath}</Descriptions.Item>
          <Descriptions.Item label="备份目录">{server?.startPathBak}</Descriptions.Item>
          <Descriptions.Item label="备份统计">{server?.bakStatEnable ? '启用' : '停用'}</Descriptions.Item>
          <Descriptions.Item label="并发数">{server?.poolSize}</Descriptions.Item>
          <Descriptions.Item label="图片缓存">{server?.imgCache ? '启用' : '停用'}</Descriptions.Item>
          <Descriptions.Item label="SQL 调试">{server?.sqlDebug ? '启用' : '停用'}</Descriptions.Item>
        </Descriptions>
      </Card>
    </Space>
  )
}
