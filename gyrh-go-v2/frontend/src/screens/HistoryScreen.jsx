import React from 'react';
import { SimpleFrame } from '../components/Layout';
import { SearchIcon, ChevronLeftIcon, ChevronRightIcon } from '../components/Icons';
import { historyCards } from '../constants';

export function HistoryScreen({ onHome, onHistory, onLogout, onToggleModel, model }) {
  return (
    <SimpleFrame 
      title="AI Smart Portrait · 历史记录管理"
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section history-panel">
        <div className="section-stack">
          <h2>生成记录（标签筛选）</h2>
          <div className="chip-row compact">
            {['全部', '人像', '背景', '构图', '写实'].map((tag, index) => (
              <button
                key={tag}
                className={`tiny-chip ${index === 0 ? 'active' : ''}`}
                type="button"
              >
                {tag}
              </button>
            ))}
          </div>
        </div>
        <button className="ghost-pill icon-pill small-float" type="button">
          <SearchIcon />
        </button>
        <div className="history-grid">
          {historyCards.map((card) => (
            <div
              key={card.id}
              className={`history-card tone-${card.tone}`}
            />
          ))}
        </div>
        <div className="history-footer">
          <button className="slider-button fill" type="button">
            <ChevronLeftIcon />
          </button>
          <div className="pager-badge">1 / 12</div>
          <button className="slider-button fill" type="button">
            <ChevronRightIcon />
          </button>
        </div>
      </section>
    </SimpleFrame>
  );
}
