import React, { useState } from 'react';
import { SimpleFrame } from '../components/Layout';
import { styleTags } from '../constants';
import { DownloadIcon, XIcon } from '../components/Icons';

export function PreviewScreen({ onHome, onHistory, onLogout, onToggleModel, model, capturedImage }) {
  const [showQR, setShowQR] = useState(false);

  return (
    <SimpleFrame 
      title="AI Smart Portrait · 全屏预览与风格转换"
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section preview-shell full-screen-preview">
        <div className="section-topline">
          <h2>全屏效果预览</h2>
          <button className="tiny-chip" type="button" onClick={onHome}>
            返回首页
          </button>
        </div>
        <div className="preview-stage-container">
          <div className="preview-stage">
            {capturedImage ? (
              <>
                <img 
                  src={capturedImage && capturedImage.startsWith('/api/v1/images/view') ? `/api/v1/images/thumbnail?url=${encodeURIComponent(capturedImage)}&w=1080&h=1920` : capturedImage} 
                  alt="Captured Portrait" 
                  className="full-preview-img"
                />
                
                {/* HUB Style Logo Overlay (Bottom Left) */}
                <div className="preview-hud-logo">
                  <div className="hud-logo-mark"></div>
                  <div className="hud-logo-text">
                    <span className="brand">GYRH</span>
                    <span className="product">AI PORTRAIT</span>
                  </div>
                </div>

                {/* Download Button (Top Right) */}
                <button 
                  className="preview-download-btn" 
                  onClick={() => setShowQR(true)}
                  title="下载图片"
                >
                  <DownloadIcon />
                </button>
              </>
            ) : (
              <div className="preview-label">
                <strong>暂无捕捉图像</strong>
                <span>请先在拍摄页面进行拍摄</span>
              </div>
            )}
          </div>
        </div>
        <div className="style-switcher">
          <h3>风格转换</h3>
          <div className="style-grid">
            {styleTags.map((tag) => (
              <button key={tag} className="style-chip" type="button">
                {tag}
              </button>
            ))}
          </div>
        </div>
      </section>

      {/* QR Code Modal */}
      {showQR && (
        <div className="qr-modal-overlay" onClick={() => setShowQR(false)}>
          <div className="qr-modal-content" onClick={e => e.stopPropagation()}>
            <button className="qr-close-btn" onClick={() => setShowQR(false)}>
              <XIcon />
            </button>
            <h3>扫码下载照片</h3>
            <div className="qr-container">
              {/* Using a placeholder QR code image */}
              <img 
                src="https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=https://github.com/anthropic" 
                alt="Download QR Code" 
              />
            </div>
            <p>使用手机扫描二维码，即可保存您的 AI 证件照</p>
          </div>
        </div>
      )}
    </SimpleFrame>
  );
}
