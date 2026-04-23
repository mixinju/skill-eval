import {ConfigProvider, App as AntApp, Typography, Space} from 'antd'
import { SmileOutlined } from '@ant-design/icons'
import zhCN from 'antd/locale/zh_CN'

function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <AntApp>
        <Space orientation="vertical" align="center" style={{ width: '100%', paddingTop: 120 }}>
          <SmileOutlined style={{ fontSize: 48, color: '#1677ff' }} />
          <Typography.Title level={2}>Skill Eval</Typography.Title>
          <Typography.Text type="secondary">评测系统前端就绪</Typography.Text>
        </Space>
      </AntApp>
    </ConfigProvider>
  )
}

export default App
