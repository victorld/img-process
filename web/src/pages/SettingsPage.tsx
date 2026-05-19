import { useQuery } from '@tanstack/react-query'
import { Alert, Card, Descriptions, Space, Typography } from 'antd'
import { api } from '../api'

type ConfigSection = {
  key: string
  title: string
  values?: Record<string, unknown>
  fields: Array<{ key: string; label: string }>
}

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
  const sections: ConfigSection[] = [
    {
      key: 'database',
      title: 'database 数据库参数',
      values: config?.database,
      fields: [
        { key: 'DbUsername', label: '数据库用户名' },
        { key: 'DbPassword', label: '数据库密码' },
        { key: 'DbHost', label: '数据库主机' },
        { key: 'DbPort', label: '数据库端口' },
        { key: 'DbName', label: '数据库名称' },
        { key: 'DbConfig', label: '数据库连接参数' },
      ],
    },
    {
      key: 'server',
      title: 'server Web 服务参数',
      values: config?.server,
      fields: [
        { key: 'HttpPort', label: 'HTTP 端口' },
        { key: 'HttpUsername', label: 'HTTP 用户名' },
        { key: 'HttpPassword', label: 'HTTP 密码' },
      ],
    },
    {
      key: 'scanArgs',
      title: 'scanArgs 扫描参数',
      values: config?.scanArgs,
      fields: [
        { key: 'StartPath', label: '扫描目录' },
        { key: 'DeleteShow', label: '显示重复删除项' },
        { key: 'MoveFileShow', label: '显示移动项' },
        { key: 'ModifyDateShow', label: '显示修改时间项' },
        { key: 'RenameFileShow', label: '显示重命名项' },
        { key: 'Md5Show', label: '计算 MD5' },
        { key: 'DeleteAction', label: '执行重复删除' },
        { key: 'MoveFileAction', label: '执行移动' },
        { key: 'ModifyDateAction', label: '执行修改时间' },
        { key: 'RenameFileAction', label: '执行重命名' },
      ],
    },
    {
      key: 'basic',
      title: 'basic 基础配置',
      values: config?.basic,
      fields: [
        { key: 'ColorOutput', label: '彩色日志' },
        { key: 'SqlDebug', label: 'SQL 调试' },
      ],
    },
    {
      key: 'cache',
      title: 'cache 缓存配置',
      values: config?.cache,
      fields: [
        { key: 'ImgCache', label: '图片缓存' },
        { key: 'SyncTable', label: '同步表数据' },
        { key: 'TruncateTable', label: '清空表数据' },
      ],
    },
    {
      key: 'dump',
      title: 'dump 扫描执行配置',
      values: config?.dump,
      fields: [
        { key: 'PoolSize', label: '并发数' },
        { key: 'Md5Retry', label: 'MD5 重试次数' },
        { key: 'Md5CountLength', label: 'MD5 采样长度' },
      ],
    },
    {
      key: 'bak',
      title: 'bak 备份配置',
      values: config?.bak,
      fields: [{ key: 'StartPathBak', label: '备份目录' }],
    },
    {
      key: 'gis',
      title: 'gis 地理编码配置',
      values: config?.gis,
      fields: [{ key: 'key', label: '高德 Key' }],
    },
    {
      key: 'batch',
      title: 'batch 批处理配置',
      values: config?.batch,
      fields: [
        { key: 'IDInsertBatchSize', label: '图片记录插入批次' },
        { key: 'IDDeleteBatchSize', label: '图片记录删除批次' },
        { key: 'GDUpdateBatchSize', label: '地理数据更新批次' },
      ],
    },
  ]

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Typography.Title level={3}>系统设置</Typography.Title>
      <Alert
        type="info"
        showIcon
        message="当前页面展示的是服务运行时实际生效配置"
        description={`配置来源：${statusQuery.data?.configFile || '-'}`}
      />
      {sections.map((section) => (
        <Card key={section.key} title={section.title}>
          <Descriptions column={{ xs: 1, sm: 1, md: 2, xl: 3 }}>
            {section.fields.map((field) => (
              <Descriptions.Item key={field.key} label={`${field.label} (${field.key})`}>
                <Typography.Text className="settings-value">
                  {formatConfigValue(section.values?.[field.key])}
                </Typography.Text>
              </Descriptions.Item>
            ))}
          </Descriptions>
        </Card>
      ))}
    </Space>
  )
}
