import { useState } from 'react';
import { ConfigProvider, App as AntApp, Layout } from 'antd';
import { ExperimentOutlined, FileSearchOutlined, SwapOutlined } from '@ant-design/icons';
import zhCN from 'antd/locale/zh_CN';
import { mockRecords } from './mock/data';
import ReportList from './components/ReportList';
import ReportDetail from './components/ReportDetail';
import CompareView from './components/CompareView';
import type { EvalRecord } from './types';
import './App.css';

const { Sider, Content } = Layout;

const darkTheme = {
  token: {
    colorPrimary: '#22d3ee',
    colorBgContainer: '#18181b',
    colorBgElevated: '#27272a',
    colorBorder: '#3f3f46',
    colorText: '#fafafa',
    colorTextSecondary: '#a1a1aa',
    borderRadius: 8,
    fontFamily: "'Outfit', sans-serif",
  },
  components: {
    Card: { colorBgContainer: '#18181b', colorBorderSecondary: '#3f3f46' },
    Table: {
      colorBgContainer: '#18181b',
      headerBg: 'rgba(255,255,255,0.02)',
      headerColor: '#a1a1aa',
      borderColor: '#3f3f46',
      rowHoverBg: 'rgba(34,211,238,0.04)',
    },
    Descriptions: { colorSplit: '#3f3f46' },
    Timeline: { dotBg: '#18181b' },
    Collapse: { headerBg: 'transparent', contentBg: '#09090b' },
    Tag: { defaultBg: 'rgba(255,255,255,0.04)', defaultColor: '#d4d4d8' },
  },
};

function App() {
  const [selected, setSelected] = useState<EvalRecord | undefined>();

  const renderContent = () => {
    if (!selected) {
      return (
        <div className="empty-state">
          <div className="empty-glyph">
            <ExperimentOutlined />
          </div>
          <div className="empty-title">Skill Eval</div>
          <div className="empty-desc">选择左侧评测记录查看详情</div>
        </div>
      );
    }

    if (selected.type === 'single' && selected.report && selected.trace) {
      return <ReportDetail report={selected.report} trace={selected.trace} />;
    }

    if (selected.type === 'compare' && selected.compareReport) {
      return <CompareView compareReport={selected.compareReport} />;
    }

    return null;
  };

  const headerIcon = selected?.type === 'compare' ? (
    <div className="content-header-icon compare"><SwapOutlined /></div>
  ) : selected ? (
    <div className="content-header-icon single"><FileSearchOutlined /></div>
  ) : null;

  const headerTitle = selected
    ? selected.type === 'single' ? '评测报告' : '对比评测'
    : '';

  const headerSubtitle = selected?.type === 'single'
    ? `${selected.report?.model} · ${selected.report?.targetSkill} · ${selected.report?.iterations} rounds`
    : selected?.type === 'compare'
      ? `${selected.compareReport?.reportA.model} vs ${selected.compareReport?.reportB.model}`
      : '';

  return (
    <ConfigProvider locale={zhCN} theme={darkTheme}>
      <AntApp>
        <Layout style={{ height: '100vh' }}>
          <Sider width={300} className="sidebar">
            <ReportList
              records={mockRecords}
              selectedId={selected?.id}
              onSelect={setSelected}
            />
          </Sider>
          <Content className="content-area">
            {selected && (
              <div className="content-inner" style={{ paddingBottom: 0 }}>
                <div className="content-header">
                  {headerIcon}
                  <div>
                    <div className="content-title">{headerTitle}</div>
                    <div className="content-subtitle">{headerSubtitle}</div>
                  </div>
                </div>
              </div>
            )}
            <div className={selected ? 'content-inner' : ''} style={selected ? { paddingTop: 0 } : {}}>
              {renderContent()}
            </div>
          </Content>
        </Layout>
      </AntApp>
    </ConfigProvider>
  );
}

export default App;
