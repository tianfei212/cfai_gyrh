import React, { useEffect, useState } from 'react';
import { HomeIcon, StackIcon, ExitIcon, SearchIcon } from './Icons';
import { RefreshingImage } from './RefreshingImage';
import { screens } from '../constants';
import { fetchApi } from '../services/api';
import { DEFAULT_BRANDING } from '../config/branding';
import {
  buildHistoryPreviewPayload,
  buildHistoryTitle,
  mapGeneratedImagesToHistoryRecords,
} from '../utils/historyRecords';
import { getModelLabel } from '../utils/modelProvider';

export function HeaderIcon({ icon, label, onClick }) {
  return (
    <button className="header-icon" type="button" onClick={onClick}>
      {label ? <span>{label}</span> : icon}
    </button>
  );
}

export function BrandLogo({ branding = DEFAULT_BRANDING }) {
  return (
    <div className="brand-logo">
      {branding.logo ? <img src={branding.logo} alt={`${branding.appName} logo`} /> : null}
    </div>
  );
}

export function TopBar({ title, children, branding = DEFAULT_BRANDING }) {
  return (
    <header className="topbar">
      <div className="brand-cluster">
        <BrandLogo branding={branding} />
        <span>{title}</span>
      </div>
      <div className="topbar-actions">{children}</div>
    </header>
  );
}

export function WorkbenchLayout({ title, headerActions, children, rightSidebar, branding }) {
  return (
    <div className="workbench-screen">
      <TopBar title={title} branding={branding}>{headerActions}</TopBar>
      <div className="workbench-grid">
        <div className="workbench-main">{children}</div>
        <div className="workbench-side">{rightSidebar}</div>
      </div>
    </div>
  );
}

export function SimpleFrame({ title, children, onHome, onHistory, onLogout, onToggleModel, model, branding }) {
  return (
    <div className="simple-screen">
      <TopBar title={title} branding={branding}>
        <HeaderIcon label={getModelLabel(model)} onClick={onToggleModel} />
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

export function ControlRail({ screen, onSelect, items = screens }) {
  return (
    <nav className="control-rail" aria-label="Screen selector">
      {items
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

function SidebarPagerIcon({ direction }) {
  const path = direction === 'left' ? 'm14.5 7.5-4.5 4.5 4.5 4.5' : 'm9.5 7.5 4.5 4.5-4.5 4.5';

  return (
    <svg className="sidebar-pager-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d={path} stroke="#f4f7ff" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.4" />
    </svg>
  );
}

export function HistorySidebar({ onPreview }) {
  const [history, setHistory] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const limit = 5;
  const totalPages = Math.ceil(total / limit) || 1;

  const fetchHistory = async (pageNumber = page) => {
    try {
      setLoading(true);
      const offset = (pageNumber - 1) * limit;
      const data = await fetchApi(`/api/v1/images?limit=${limit}&offset=${offset}`);
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
    fetchHistory(page);
  }, [page]);

  const handlePrevPage = () => {
    if (page > 1 && !loading) {
      setPage(page - 1);
    }
  };

  const handleNextPage = () => {
    if (page < totalPages && !loading) {
      setPage(page + 1);
    }
  };

  const handleRefresh = () => {
    fetchHistory(page);
  };

  return (
    <aside className="history-sidebar">
      <div className="section-topline">
        <div className="sidebar-title-row">
          <h2>{buildHistoryTitle(total)}</h2>
          <div className="sidebar-pager" aria-label="历史记录翻页">
            <button
              className="tiny-chip icon-chip"
              type="button"
              aria-label="上一页历史记录"
              onClick={handlePrevPage}
              disabled={page === 1 || loading}
            >
              <SidebarPagerIcon direction="left" />
            </button>
            <button
              className="tiny-chip icon-chip"
              type="button"
              aria-label="下一页历史记录"
              onClick={handleNextPage}
              disabled={page >= totalPages || loading}
            >
              <SidebarPagerIcon direction="right" />
            </button>
          </div>
        </div>
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
                <RefreshingImage src={record.url} alt={`历史记录 ${record.id}`} />
              ) : (
                <span>图片丢失</span>
              )}
            </button>
          ))
        )}
      </div>
      <div className="sidebar-search">
        <input placeholder="搜索历史记录" />
        <button type="button" onClick={handleRefresh}>
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
