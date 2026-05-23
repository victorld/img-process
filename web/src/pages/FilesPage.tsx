import { useQuery } from '@tanstack/react-query'
import {
  Button,
  Card,
  Col,
  Drawer,
  Image,
  Input,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tooltip,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useMemo, useState } from 'react'
import { api } from '../api'
import type { FileAnalysisItem, FileAnalysisStatItem } from '../types'
import { formatDateTime } from '../utils/dateTime'

type FileFilters = {
  fileKey: string
  shootDateStatus: string
  geoStatus: string
  locAddrKeyword: string
  shootDateStart: string
  shootDateEnd: string
  page: number
  pageSize: number
}

const initialFilters: FileFilters = {
  fileKey: '',
  shootDateStatus: '',
  geoStatus: '',
  locAddrKeyword: '',
  shootDateStart: '',
  shootDateEnd: '',
  page: 1,
  pageSize: 10,
}

function formatPercent(value?: number) {
  return `${Number(value ?? 0).toFixed(1)}%`
}

function formatCount(value?: number) {
  return Number(value ?? 0).toLocaleString()
}

function StatBars({ items }: { items: FileAnalysisStatItem[] }) {
  const max = Math.max(...items.map((item) => item.count), 1)
  return (
    <Space direction="vertical" size={10} style={{ width: '100%' }}>
      {items.map((item) => (
        <div className="file-stat-row" key={item.key}>
          <Typography.Text>{item.key || '-'}</Typography.Text>
          <div className="file-stat-track">
            <span style={{ width: `${Math.max(4, (item.count / max) * 100)}%` }} />
          </div>
          <Typography.Text strong>
            {formatCount(item.count)}
            {item.percent ? ` / ${formatPercent(item.percent)}` : ''}
          </Typography.Text>
        </div>
      ))}
    </Space>
  )
}

function CoverageCard({ title, value, coverage }: { title: string; value?: number; coverage?: number }) {
  return (
    <Card>
      <Statistic title={title} value={value ?? 0} />
      <Typography.Text className="file-coverage-text" type="secondary">
        覆盖率 {formatPercent(coverage)}
      </Typography.Text>
    </Card>
  )
}

function EllipsisText({ value, className }: { value?: string; className?: string }) {
  const text = value || '-'
  return (
    <Tooltip title={text}>
      <Typography.Text className={className} ellipsis>
        {text}
      </Typography.Text>
    </Tooltip>
  )
}

function downloadCSV(rows: FileAnalysisItem[]) {
  const headers = ['img_key', '目录日期', '文件名', 'shoot_date', 'loc_num', 'loc_street', 'loc_addr', 'remark', '更新时间']
  const escape = (value: unknown) => `"${String(value ?? '').replace(/"/g, '""')}"`
  const lines = [
    headers.map(escape).join(','),
    ...rows.map((item) =>
      [
        item.imgKey,
        item.dirDate,
        item.fileName,
        item.shootDate,
        item.locNum,
        item.locStreet,
        item.locAddr,
        item.remark,
        item.updatedAt,
      ].map(escape).join(','),
    ),
  ]
  const blob = new Blob([`\ufeff${lines.join('\n')}`], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `file-analysis-${Date.now()}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

export function FilesPage() {
  const [messageApi, contextHolder] = message.useMessage()
  const [filters, setFilters] = useState<FileFilters>(initialFilters)
  const [draft, setDraft] = useState<FileFilters>(initialFilters)
  const [statDrawer, setStatDrawer] = useState<{ title: string; items: FileAnalysisStatItem[] } | null>(null)
  const [detail, setDetail] = useState<FileAnalysisItem | null>(null)

  const query = useQuery({
    queryKey: ['file-analysis', filters],
    queryFn: () =>
      api.getFileAnalysis(
        new URLSearchParams({
          page: String(filters.page),
          pageSize: String(filters.pageSize),
          fileKey: filters.fileKey,
          shootDateStatus: filters.shootDateStatus,
          geoStatus: filters.geoStatus,
          locAddrKeyword: filters.locAddrKeyword,
          shootDateStart: filters.shootDateStart,
          shootDateEnd: filters.shootDateEnd,
        }),
      ),
  })

  const rows = query.data?.list ?? []
  const summary = query.data?.summary
  const yearStats = query.data?.yearStats ?? []
  const suffixStats = query.data?.suffixStats ?? []

  const columns: ColumnsType<FileAnalysisItem> = useMemo(
    () => [
      {
        title: '缩略图',
        dataIndex: 'previewUrl',
        width: 76,
        render: (value: string, record) => (
          <span onClick={(event) => event.stopPropagation()}>
            <Image
              className="action-preview"
              width={56}
              height={56}
              src={value}
              preview={{ src: value ? value.replace('quality=thumb', 'quality=full') : value }}
              alt={record.fileName || '文件'}
            />
          </span>
        ),
      },
      { title: 'img_key', dataIndex: 'imgKey', width: 170, render: (value) => <EllipsisText value={value} /> },
      { title: '目录日期', dataIndex: 'dirDate', width: 104 },
      { title: '文件名', dataIndex: 'fileName', width: 150, render: (value) => <EllipsisText value={value} /> },
      { title: 'shoot_date', dataIndex: 'shootDate', width: 132, render: (value) => <EllipsisText value={value} /> },
      { title: 'loc_num', dataIndex: 'locNum', width: 132, render: (value) => <EllipsisText value={value} /> },
      { title: 'loc_street', dataIndex: 'locStreet', width: 120, render: (value) => <EllipsisText value={value} /> },
      { title: 'loc_addr', dataIndex: 'locAddr', width: 160, render: (value) => <EllipsisText value={value} /> },
      { title: 'remark', dataIndex: 'remark', width: 180, render: (value) => <EllipsisText className="file-remark-text" value={value} /> },
      { title: '更新时间', dataIndex: 'updatedAt', width: 124, render: (value) => <EllipsisText value={formatDateTime(value)} /> },
      {
        title: '操作',
        width: 72,
        render: (_, record) => <Button type="link" onClick={(event) => {
          event.stopPropagation()
          setDetail(record)
        }}>明细</Button>,
      },
    ],
    [],
  )

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {contextHolder}
      <Row justify="space-between" align="middle">
        <Col>
          <Typography.Title level={3}>文件分析</Typography.Title>
          <Typography.Text type="secondary">基于 img_database 表查看文件拍摄时间、地理信息和缓存信息。</Typography.Text>
        </Col>
        <Col>
          <Button onClick={() => query.refetch()}>刷新分析</Button>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card><Statistic title="总缓存文件数" value={summary?.totalCount ?? 0} /></Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <CoverageCard title="有拍摄时间文件数" value={summary?.withShootDateCount} coverage={summary?.shootDateCoverage} />
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <CoverageCard title="有经纬度文件数" value={summary?.withLocNumCount} coverage={summary?.locNumCoverage} />
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <CoverageCard title="有地址信息文件数" value={summary?.withLocAddrCount} coverage={summary?.locAddrCoverage} />
        </Col>
      </Row>

      <Card title="筛选条件" extra={<Button type="primary" onClick={() => setFilters({ ...draft, page: 1 })}>筛选</Button>}>
        <Row gutter={[12, 12]}>
          <Col xs={24} md={8}>
            <Input placeholder="文件 key，支持模糊搜索 img_key" value={draft.fileKey} onChange={(event) => setDraft((prev) => ({ ...prev, fileKey: event.target.value }))} />
          </Col>
          <Col xs={24} md={8}>
            <Select
              placeholder="拍摄时间状态"
              allowClear
              style={{ width: '100%' }}
              value={draft.shootDateStatus || undefined}
              onChange={(value) => setDraft((prev) => ({ ...prev, shootDateStatus: value ?? '' }))}
              options={[
                { label: '有拍摄时间', value: 'present' },
                { label: '缺失拍摄时间', value: 'missing' },
              ]}
            />
          </Col>
          <Col xs={24} md={8}>
            <Select
              placeholder="地理信息状态"
              allowClear
              style={{ width: '100%' }}
              value={draft.geoStatus || undefined}
              onChange={(value) => setDraft((prev) => ({ ...prev, geoStatus: value ?? '' }))}
              options={[
                { label: '有经纬度和地址', value: 'full' },
                { label: '仅要求有经纬度', value: 'loc_num' },
                { label: '缺失地理信息', value: 'missing' },
              ]}
            />
          </Col>
          <Col xs={24} md={8}>
            <Input placeholder="地理信息检索，模糊搜索 loc_addr" value={draft.locAddrKeyword} onChange={(event) => setDraft((prev) => ({ ...prev, locAddrKeyword: event.target.value }))} />
          </Col>
          <Col xs={24} md={8}>
            <Input placeholder="文件拍摄日期最早时间" value={draft.shootDateStart} onChange={(event) => setDraft((prev) => ({ ...prev, shootDateStart: event.target.value }))} />
          </Col>
          <Col xs={24} md={8}>
            <Input placeholder="文件拍摄日期最晚时间" value={draft.shootDateEnd} onChange={(event) => setDraft((prev) => ({ ...prev, shootDateEnd: event.target.value }))} />
          </Col>
        </Row>
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <Card title="按年份统计" extra={<Button type="link" onClick={() => setStatDrawer({ title: '全部年份统计', items: yearStats })}>更多</Button>}>
            <StatBars items={yearStats.slice(0, 10)} />
          </Card>
        </Col>
        <Col xs={24} xl={12}>
          <Card title="文件后缀统计" extra={<Button type="link" onClick={() => setStatDrawer({ title: '全部文件后缀统计', items: suffixStats })}>更多</Button>}>
            <StatBars items={suffixStats.slice(0, 10)} />
          </Card>
        </Col>
      </Row>

      <Card title="文件明细" extra={<Button type="link" onClick={() => {
        downloadCSV(rows)
        messageApi.success('已导出当前页数据')
      }}>导出</Button>}>
        <div className="file-analysis-table">
          <Table
            rowKey="id"
            loading={query.isLoading}
            tableLayout="fixed"
            scroll={{ x: 1420 }}
            columns={columns}
            dataSource={rows}
            pagination={{
              total: query.data?.total ?? 0,
              current: filters.page,
              pageSize: filters.pageSize,
              onChange: (page, pageSize) => {
                setFilters((prev) => ({ ...prev, page, pageSize }))
                setDraft((prev) => ({ ...prev, page, pageSize }))
              },
            }}
          />
        </div>
      </Card>

      <Drawer title={statDrawer?.title} open={Boolean(statDrawer)} width={520} onClose={() => setStatDrawer(null)}>
        <StatBars items={statDrawer?.items ?? []} />
      </Drawer>

      <Drawer title="文件明细" open={Boolean(detail)} width={620} onClose={() => setDetail(null)}>
        {detail ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Image width={120} height={120} className="action-preview" src={detail.previewUrl} alt={detail.fileName} />
            <Row gutter={[12, 12]}>
              {[
                ['img_key', detail.imgKey],
                ['目录日期', detail.dirDate],
                ['文件名', detail.fileName],
                ['shoot_date', detail.shootDate || '缺失'],
                ['loc_num', detail.locNum || '缺失'],
                ['更新时间', formatDateTime(detail.updatedAt)],
              ].map(([label, value]) => (
                <Col span={12} key={label}>
                  <Typography.Text type="secondary">{label}</Typography.Text>
                  <div>{value}</div>
                </Col>
              ))}
            </Row>
            <Card title="地理信息" size="small">
              <Space direction="vertical">
                <Typography.Text>loc_street：{detail.locStreet || '缺失'}</Typography.Text>
                <Typography.Text>loc_addr：{detail.locAddr || '缺失'}</Typography.Text>
              </Space>
            </Card>
            <Card title="备注 / EXIF 输出" size="small">
              <pre className="json-block">{detail.remark || '-'}</pre>
            </Card>
          </Space>
        ) : null}
      </Drawer>
    </Space>
  )
}
