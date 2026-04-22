import React from 'react';
import { HomeIcon, StackIcon, ExitIcon, SearchIcon } from './Icons';
import { screens } from '../constants';

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

export function HistorySidebar() {
  return (
    <aside className="history-sidebar">
      <div className="section-topline">
        <h2>历史记录 (74)</h2>
      </div>
      <div className="sidebar-list">
        {['blue', 'red', 'red', 'blue', 'red'].map((tone, index) => (
          <div key={`${tone}-${index}`} className={`sidebar-card tone-${tone}`} />
        ))}
      </div>
      <div className="sidebar-search">
        <input placeholder="搜索历史记录" />
        <button type="button">
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
