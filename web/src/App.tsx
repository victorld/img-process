import { useQuery } from '@tanstack/react-query'
import { App as AntApp, Button, Form, Input, Layout, Menu, message, Spin } from 'antd'
import { useMemo } from 'react'
import { Link, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { api } from './api'
import { DashboardPage } from './pages/DashboardPage'
import { JobDetailPage } from './pages/JobDetailPage'
import { JobsPage } from './pages/JobsPage'
import { SchedulesPage } from './pages/SchedulesPage'
import { SettingsPage } from './pages/SettingsPage'

const { Header, Sider, Content } = Layout

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
      <div className="login-card">
        <h1>img-process 管理台</h1>
        <p>扫描历史、实时过程、计划任务统一查看与控制。</p>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Button type="primary" htmlType="submit" block>
            登录
          </Button>
        </Form>
      </div>
    </div>
  )
}

function ProtectedLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['me'],
    queryFn: api.me,
    retry: false,
  })

  const selectedKey = useMemo(() => {
    if (location.pathname.startsWith('/jobs')) return 'jobs'
    if (location.pathname.startsWith('/schedules')) return 'schedules'
    if (location.pathname.startsWith('/settings')) return 'settings'
    return 'dashboard'
  }, [location.pathname])

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
    <AntApp>
      <Layout className="app-shell">
        <Sider width={220} theme="light" className="app-sider">
          <div className="brand">img-process</div>
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            items={[
              { key: 'dashboard', label: <Link to="/">仪表盘</Link> },
              { key: 'jobs', label: <Link to="/jobs">扫描历史</Link> },
              { key: 'schedules', label: <Link to="/schedules">计划任务</Link> },
              { key: 'settings', label: <Link to="/settings">系统设置</Link> },
            ]}
          />
        </Sider>
        <Layout>
          <Header className="app-header">
            <div>当前用户：{data.username}</div>
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
              <Route path="/schedules" element={<SchedulesPage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </AntApp>
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
