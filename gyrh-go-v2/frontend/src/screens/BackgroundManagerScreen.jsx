import React from 'react';
import { SimpleFrame } from '../components/Layout';
import { backgroundRows } from '../constants';

export function BackgroundManagerScreen({ onHome, onHistory, onLogout, onToggleModel, model }) {
  return (
    <SimpleFrame 
      title="AI Smart Portrait · 背景管理"
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section table-panel">
        <div className="section-topline">
          <h2>背景图管理</h2>
          <div className="chip-row">
            <button className="tiny-chip" type="button">导入</button>
            <button className="tiny-chip" type="button">同步</button>
            <button className="tiny-chip active" type="button">本地库</button>
          </div>
        </div>
        <div className="table-shell">
          <div className="table-header table-grid">
            <span>编号</span>
            <span>缩略</span>
            <span>Wan 提示词</span>
            <span>Gemini 提示词</span>
            <span>操作</span>
          </div>
          {backgroundRows.map((row) => (
            <div className="table-row table-grid" key={row.id}>
              <span>{row.id}</span>
              <div className={`thumb-swatch tone-${row.tone}`} />
              <span>{row.wan}</span>
              <span>{row.gemini}</span>
              <div className="table-actions">
                <button className="mini-outline" type="button">查看原图</button>
                <button className="mini-outline" type="button">悬浮预览</button>
                <button className="mini-outline" type="button">删除</button>
              </div>
            </div>
          ))}
        </div>
      </section>
    </SimpleFrame>
  );
}
