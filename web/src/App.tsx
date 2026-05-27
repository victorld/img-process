import { MenuOutlined } from '@ant-design/icons'
import { useQuery } from '@tanstack/react-query'
import { App as AntApp, Button, Checkbox, ConfigProvider, Form, Input, Layout, Menu, message, Spin } from 'antd'
import { useMemo, useState } from 'react'
import { Link, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { api } from './api'
import { DashboardPage } from './pages/DashboardPage'
import { FilesPage } from './pages/FilesPage'
import { JobDetailPage } from './pages/JobDetailPage'
import { JobsPage } from './pages/JobsPage'
import { SchedulesPage } from './pages/SchedulesPage'
import { SettingsPage } from './pages/SettingsPage'

const { Header, Sider, Content } = Layout

const pageMeta = {
  dashboard: { title: '工作台', subtitle: '当前用户', index: '01', path: '/' },
  jobs: { title: '扫描历史', subtitle: '当前用户', index: '02', path: '/jobs' },
  files: { title: '文件分析', subtitle: '当前用户', index: '03', path: '/files' },
  schedules: { title: '计划任务', subtitle: '当前用户', index: '04', path: '/schedules' },
  settings: { title: '系统设置', subtitle: '当前用户', index: '05', path: '/settings' },
} as const

function LoginPage() {
  const [form] = Form.useForm()
  const navigate = useNavigate()
  const [messageApi, contextHolder] = message.useMessage()

  const handleSubmit = async (values: { username: string; password: string }) => {
    try {
      await api.login(values.username, values.password)
      messageApi.success('登录成功')
      navigate('/')
    } catch (error) {
      messageApi.error(error instanceof Error ? error.message : '登录失败')
    }
  }

  return (
    <div className="login-shell">
      {contextHolder}
      <section className="form-panel" aria-label="登录">
        <div className="login-stack">
          <div className="login-card">
            <div className="card-head">
              <div className="brand-lockup" aria-label="img-process">
                <div className="login-logo">IP</div>
                <div className="brand-text">
                  <strong>img-process</strong>
                  <span>图片治理与扫描任务管理</span>
                </div>
              </div>
            </div>
            <Form form={form} layout="vertical" className="login-form" onFinish={handleSubmit}>
              <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
                <Input placeholder="请输入用户名" autoComplete="username" />
              </Form.Item>
              <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
                <Input.Password placeholder="请输入密码" autoComplete="current-password" />
              </Form.Item>
              <div className="row login-options">
                <Checkbox defaultChecked>记住登录状态</Checkbox>
              </div>
              <Button type="primary" htmlType="submit" block>
                登录工作台
              </Button>
            </Form>
          </div>
        </div>
      </section>
    </div>
  )
}

function ProtectedLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['me'],
    queryFn: api.me,
    retry: false,
  })

  const selectedKey = useMemo(() => {
    if (location.pathname.startsWith('/jobs')) return 'jobs'
    if (location.pathname.startsWith('/files')) return 'files'
    if (location.pathname.startsWith('/schedules')) return 'schedules'
    if (location.pathname.startsWith('/settings')) return 'settings'
    return 'dashboard'
  }, [location.pathname])
  const activeMeta = pageMeta[selectedKey as keyof typeof pageMeta]

  if (isLoading) {
    return (
      <div className="center-shell">
        <Spin size="large" />
      </div>
    )
  }

  if (!data?.authenticated) {
    return <Navigate to="/login" replace />
  }

  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#2f8a66',
          borderRadius: 8,
          colorText: '#20303d',
          colorTextSecondary: '#6d7b86',
          colorBorder: '#dfe6e1',
          fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif',
        },
      }}
    >
      <AntApp>
      <Layout className="app-shell">
        <Sider width={248} theme="dark" className={`app-sider${menuOpen ? ' open' : ''}`}>
          <div className="brand">
            <div className="logo-mark">IP</div>
            <div>
              <b>img-process</b>
              <span>照片治理业务系统</span>
            </div>
          </div>
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            items={[
              ...Object.entries(pageMeta).map(([key, item]) => ({
                key,
                label: (
                  <Link to={item.path} onClick={() => setMenuOpen(false)}>
                    <span className="nav-icon">{item.index}</span>
                    <span>{item.title}</span>
                  </Link>
                ),
              })),
            ]}
          />
        </Sider>
        <div className={`mobile-sider-mask${menuOpen ? ' open' : ''}`} onClick={() => setMenuOpen(false)} />
        <Layout className="app-main">
          <Header className="app-header">
            <div className="header-left">
              <Button className="mobile-menu" icon={<MenuOutlined />} onClick={() => setMenuOpen((value) => !value)}>
                菜单
              </Button>
              <div className="header-title">
                <b>{activeMeta.title}</b>
                <span>{activeMeta.subtitle}：{data.username}</span>
              </div>
            </div>
            <div className="header-actions">
              <Button onClick={() => refetch()}>刷新会话</Button>
              <Button
                danger
                onClick={async () => {
                  await api.logout()
                  navigate('/login')
                }}
              >
                退出
              </Button>
            </div>
          </Header>
          <Content className="app-content">
            <Routes>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/jobs" element={<JobsPage />} />
              <Route path="/jobs/:id" element={<JobDetailPage />} />
              <Route path="/files" element={<FilesPage />} />
              <Route path="/schedules" element={<SchedulesPage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
      </AntApp>
    </ConfigProvider>
  )
}

export function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/*" element={<ProtectedLayout />} />
    </Routes>
  )
}
