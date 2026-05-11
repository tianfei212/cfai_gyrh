import React, { useEffect, useState } from 'react';
import { HomeIcon, StackIcon, ExitIcon, SearchIcon } from './Icons';
import { screens } from '../constants';
import { fetchApi } from '../services/api';
import {
  buildHistoryPreviewPayload,
  buildHistoryTitle,
  mapGeneratedImagesToHistoryRecords,
} from '../utils/historyRecords';

export function HeaderIcon({ icon, label, onClick }) {
  return (
    <button className="header-icon" type="button" onClick={onClick}>
      {label ? <span>{label}</span> : icon}
    </button>
  );
}

export function TopBar({ title, children }) {
  return (
    <header className="topbar">
      <div className="brand-cluster">
        <div className="brand-logo" />
        <span>{title}</span>
      </div>
      <div className="topbar-actions">{children}</div>
    </header>
  );
}

export function WorkbenchLayout({ title, headerActions, children, rightSidebar }) {
  return (
    <div className="workbench-screen">
      <TopBar title={title}>{headerActions}</TopBar>
      <div className="workbench-grid">
        <div className="workbench-main">{children}</div>
        <div className="workbench-side">{rightSidebar}</div>
      </div>
    </div>
  );
}

export function SimpleFrame({ title, children, onHome, onHistory, onLogout, onToggleModel, model }) {
  return (
    <div className="simple-screen">
      <TopBar title={title}>
        <HeaderIcon label={model === 'W' ? 'W' : 'G'} onClick={onToggleModel} />
        <HeaderIcon icon={<HomeIcon />} onClick={onHome} />
        <HeaderIcon icon={<StackIcon />} onClick={onHistory} />
        <HeaderIcon icon={<ExitIcon />} onClick={onLogout} />
      </TopBar>
      <div className="simple-content">{children}</div>
    </div>
  );
}

export function CenteredStage({ children, muted = false }) {
  return (
    <div className={`centered-screen ${muted ? 'muted' : ''}`}>
      <div className="centered-panel">{children}</div>
    </div>
  );
}

export function ControlRail({ screen, onSelect }) {
  return (
    <nav className="control-rail" aria-label="Screen selector">
      {screens
        .filter((item) => !item.hideInNav)
        .map((item) => (
          <button
            key={item.key}
            type="button"
            className={item.key === screen ? 'active' : ''}
            onClick={() => onSelect(item.key)}
          >
            {item.label}
          </button>
        ))}
    </nav>
  );
}

export function HistorySidebar({ onPreview }) {
  const [history, setHistory] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const limit = 5;

  const fetchHistory = async () => {
    try {
      setLoading(true);
      const data = await fetchApi(`/api/v1/images?limit=${limit}&offset=0`);
      setHistory(mapGeneratedImagesToHistoryRecords(data.images || []));
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to fetch sidebar history:', err);
      setHistory([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHistory();
  }, []);

  return (
    <aside className="history-sidebar">
      <div className="section-topline">
        <h2>{buildHistoryTitle(total)}</h2>
      </div>
      <div className="sidebar-list">
        {loading ? (
          ['blue', 'red', 'red', 'blue', 'red'].map((tone, index) => (
            <div key={`${tone}-${index}`} className={`sidebar-card tone-${tone} sidebar-card-loading`} />
          ))
        ) : history.length === 0 ? (
          <div className="sidebar-empty">暂无历史记录</div>
        ) : (
          history.map((record) => (
            <button
              key={record.id}
              className="sidebar-card sidebar-card-image"
              type="button"
              title={`生成时间: ${new Date(record.created_at).toLocaleString()}`}
              onClick={() => onPreview?.(buildHistoryPreviewPayload(record))}
            >
              {record.url ? (
                <img src={record.url} alt={`历史记录 ${record.id}`} />
              ) : (
                <span>图片丢失</span>
              )}
            </button>
          ))
        )}
      </div>
      <div className="sidebar-search">
        <input placeholder="搜索历史记录" />
        <button type="button" onClick={fetchHistory}>
          <SearchIcon />
        </button>
      </div>
    </aside>
  );
}

export function GlowBackdrop() {
  return (
    <div className="glow-backdrop" aria-hidden="true">
      <span className="glow orb-a" />
      <span className="glow orb-b" />
      <span className="glow orb-c" />
    </div>
  );
}
