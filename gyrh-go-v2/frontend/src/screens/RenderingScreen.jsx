import React from 'react';
import { SimpleFrame } from '../components/Layout';
import { buildScreenTitle, DEFAULT_BRANDING } from '../config/branding';

export function RenderingScreen({ onHome, onHistory, onLogout, onToggleModel, model, branding = DEFAULT_BRANDING }) {
  return (
    <SimpleFrame 
      title={buildScreenTitle(branding, '生成等待中')}
      branding={branding}
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section rendering-shell">
        <div className="particle-field" />
        <div className="countdown-badge">
          <span className="countdown-number">08</span>
          <span className="countdown-unit">秒</span>
        </div>
        <div className="rendering-copy">
          <h2>片场渲染中</h2>
          <p>Light · Grain · Color 正在融合</p>
        </div>
        <div className="pulse-rings">
          <span />
          <span />
          <span />
        </div>
        <div className="render-pill">GENERATING PRESET</div>
      </section>
    </SimpleFrame>
  );
}
