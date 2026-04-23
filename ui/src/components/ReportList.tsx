import { useState } from 'react';
import { Select, Badge } from 'antd';
import {
  CheckCircleFilled,
  CloseCircleFilled,
  SwapOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import type { EvalRecord } from '../types';

interface ReportListProps {
  records: EvalRecord[];
  selectedId?: string;
  onSelect: (record: EvalRecord) => void;
}

export default function ReportList({ records, selectedId, onSelect }: ReportListProps) {
  const [modelFilter, setModelFilter] = useState<string | undefined>();
  const [skillFilter, setSkillFilter] = useState<string | undefined>();
  const [passFilter, setPassFilter] = useState<string | undefined>();
  const [typeFilter, setTypeFilter] = useState<string | undefined>();

  const models = [...new Set(records.map((r) => {
    if (r.type === 'single') return r.report?.model;
    return r.compareReport?.reportA.model;
  }).filter(Boolean))] as string[];

  const skills = [...new Set(records.map((r) => {
    if (r.type === 'single') return r.report?.targetSkill;
    return r.compareReport?.reportA.targetSkill;
  }).filter(Boolean))] as string[];

  const filtered = records.filter((r) => {
    if (typeFilter && r.type !== typeFilter) return false;
    if (r.type === 'single') {
      if (modelFilter && r.report?.model !== modelFilter) return false;
      if (skillFilter && r.report?.targetSkill !== skillFilter) return false;
      if (passFilter === 'pass' && !r.report?.pass) return false;
      if (passFilter === 'fail' && r.report?.pass) return false;
    } else {
      if (modelFilter && r.compareReport?.reportA.model !== modelFilter && r.compareReport?.reportB.model !== modelFilter) return false;
      if (skillFilter && r.compareReport?.reportA.targetSkill !== skillFilter) return false;
      if (passFilter === 'pass' && r.compareReport?.scores.some((s) => !s.pass)) return false;
      if (passFilter === 'fail' && r.compareReport?.scores.every((s) => s.pass)) return false;
    }
    return true;
  });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className="sidebar-header">
        <div className="sidebar-logo">
          <div className="sidebar-logo-icon">
            <ExperimentOutlined />
          </div>
          <div>
            <div className="sidebar-logo-text">Skill Eval</div>
            <div className="sidebar-logo-sub">
              <Badge status="processing" /> {records.length} records
            </div>
          </div>
        </div>
        <div className="filter-group">
          <div className="filter-row">
            <Select placeholder="模型" allowClear size="small" value={modelFilter} onChange={setModelFilter}
              options={models.map((m) => ({ label: m, value: m }))} />
            <Select placeholder="Skill" allowClear size="small" value={skillFilter} onChange={setSkillFilter}
              options={skills.map((s) => ({ label: s, value: s }))} />
          </div>
          <div className="filter-row">
            <Select placeholder="状态" allowClear size="small" value={passFilter} onChange={setPassFilter}
              options={[{ label: 'PASS', value: 'pass' }, { label: 'FAIL', value: 'fail' }]} />
            <Select placeholder="类型" allowClear size="small" value={typeFilter} onChange={setTypeFilter}
              options={[{ label: '单评测', value: 'single' }, { label: '对比', value: 'compare' }]} />
          </div>
        </div>
      </div>

      <div className="sidebar-list">
        {filtered.map((item, i) => {
          const isSingle = item.type === 'single';
          const pass = isSingle ? item.report?.pass : item.compareReport?.scores.every((s) => s.pass);
          const model = isSingle
            ? item.report?.model
            : `${item.compareReport?.reportA.model} vs ${item.compareReport?.reportB.model}`;
          const skill = isSingle
            ? item.report?.targetSkill
            : item.compareReport?.reportA.targetSkill;
          const isSelected = selectedId === item.id;

          return (
            <div
              key={item.id}
              className={`record-card ${isSelected ? 'selected' : ''}`}
              style={{ animationDelay: `${i * 40}ms` }}
              onClick={() => onSelect(item)}
            >
              <div className="record-card-inner">
                <div className="record-card-top">
                  <div className="record-card-tags">
                    {isSingle
                      ? <span className="mtag mtag-single"><FileTextOutlined /> 单评测</span>
                      : <span className="mtag mtag-compare"><SwapOutlined /> 对比</span>
                    }
                    {pass
                      ? <span className="mtag mtag-pass"><CheckCircleFilled /> PASS</span>
                      : <span className="mtag mtag-fail"><CloseCircleFilled /> FAIL</span>
                    }
                  </div>
                </div>
                <div className="record-card-model">{model}</div>
                <div className="record-card-meta">
                  <span>skill: {skill}</span>
                  <span><ClockCircleOutlined style={{ fontSize: 9 }} />{' '}
                    {new Date(item.createdAt).toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
