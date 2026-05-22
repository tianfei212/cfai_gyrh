import React, { useState, useEffect } from 'react';
import { SimpleFrame } from '../components/Layout';
import { RefreshingImage } from '../components/RefreshingImage';
import { DownloadIcon, XIcon } from '../components/Icons';
import { buildScreenTitle, DEFAULT_BRANDING } from '../config/branding';
import { fetchApi } from '../services/api';
import { resolveRewriteResponse } from '../services/rewriteTask';
import { appendImageCacheBucket } from '../utils/imageThumbs';
import { getProviderForModel } from '../utils/modelProvider';

export function PreviewScreen({ onHome, onHistory, onLogout, onToggleModel, model, capturedImage, capturedAssetId = '', previewMode = 'compare', onPreview, branding = DEFAULT_BRANDING }) {
  const [showQR, setShowQR] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isTransferring, setIsTransferring] = useState(false);
  const [stylePrompts, setStylePrompts] = useState([]);
  const [originalImage, setOriginalImage] = useState(capturedImage);
  const [currentImage, setCurrentImage] = useState(capturedImage);
  const isSinglePreview = previewMode === 'single';

  useEffect(() => {
    const handleFullscreenChange = () => {
      if (!document.fullscreenElement) {
        setIsFullscreen(false);
      }
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  // Fetch active style prompts from the backend
  useEffect(() => {
    const fetchStyles = async () => {
      try {
        const data = await fetchApi('/api/v1/style-prompts?active=true');
        setStylePrompts(data || []);
      } catch (err) {
        console.error('Failed to fetch style prompts:', err);
      }
    };
    fetchStyles();
  }, []);

  // Update state when capturedImage prop changes
  useEffect(() => {
    if (!originalImage && capturedImage) {
      setOriginalImage(capturedImage);
    }
    setCurrentImage(capturedImage);
  }, [capturedImage]);

  const handleStyleTransfer = async (styleId, styleName) => {
    if (!originalImage || isTransferring) return;
    setIsTransferring(true);
    try {
      const payload = {
        style_prompt_id: styleId,
        provider: getProviderForModel(model)
      };

      // 优先判断是否已经是后端的图片（提取 asset_id）
      let isLocalBase64 = false;
      let foregroundAssetId = '';
      
      if (capturedAssetId) {
        foregroundAssetId = capturedAssetId;
      } else if (originalImage.startsWith('/api/v1/images/view/')) {
        foregroundAssetId = originalImage.replace('/api/v1/images/view/', '').split('?')[0];
      } else if (originalImage.startsWith('http')) {
        try {
          const urlObj = new URL(originalImage);
          if (urlObj.pathname.startsWith('/api/v1/images/view/')) {
            foregroundAssetId = urlObj.pathname.replace('/api/v1/images/view/', '');
          }
        } catch (e) {}
      }

      if (foregroundAssetId) {
        payload.foreground_asset_id = foregroundAssetId;
      } else {
        // 如果是本地选的图或相机直出的 base64
        let base64;
        if (originalImage.startsWith('data:image')) {
          base64 = originalImage.split(',')[1];
        } else {
          // 最后降级处理：尝试重新下载并转 base64
          const url = originalImage.startsWith('/api/v1/images/view') 
            ? appendImageCacheBucket(`/api/v1/images/thumbnail?url=${encodeURIComponent(originalImage)}&w=1080&h=1920`) 
            : originalImage;
          const res = await fetch(url);
          const blob = await res.blob();
          base64 = await new Promise((resolve) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result.split(',')[1]);
            reader.readAsDataURL(blob);
          });
        }
        payload.foreground = base64;
        isLocalBase64 = true;
      }

      console.log('Sending rewrite request:', { 
        ...payload, 
        foreground: isLocalBase64 ? 'base64_data_omitted' : undefined 
      });

      const rewriteData = await fetchApi('/api/v1/images/rewrite', {
        method: 'POST',
        body: JSON.stringify(payload)
      });
      const data = await resolveRewriteResponse(rewriteData);
      console.log('Rewrite response:', data);

      if (data && data.image_url) {
        setCurrentImage(data.image_url);
        if (onPreview) {
          onPreview(data.image_url);
        }
      }
    } catch (err) {
      console.error('Style transfer failed:', err);
      alert('风格转换失败: ' + err.message);
    } finally {
      setIsTransferring(false);
    }
  };

  const getAbsoluteUrl = (url) => {
    if (!url) return '';
    try {
      return new URL(url, window.location.href).href;
    } catch (e) {
      return url;
    }
  };

  const getDisplayImageSrc = (imgSrc) => {
    return imgSrc && imgSrc.startsWith('/api/v1/images/view') 
      ? appendImageCacheBucket(`/api/v1/images/thumbnail?url=${encodeURIComponent(imgSrc)}&w=1080&h=1920`) 
      : imgSrc;
  };

  const openFullscreenPreview = async () => {
    if (!currentImage) return;
    setIsFullscreen(true);

    try {
      if (!document.fullscreenElement && document.documentElement.requestFullscreen) {
        await document.documentElement.requestFullscreen();
      }
    } catch (err) {
      console.warn('Native fullscreen failed, falling back to overlay:', err);
    }
  };

  const closeFullscreenPreview = async () => {
    setIsFullscreen(false);

    try {
      if (document.fullscreenElement && document.exitFullscreen) {
        await document.exitFullscreen();
      }
    } catch (err) {
      console.warn('Exit fullscreen failed:', err);
    }
  };

  return (
    <SimpleFrame 
      title={buildScreenTitle(branding, '全屏预览与风格转换')}
      branding={branding}
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section preview-shell full-screen-preview">
        <div className="section-topline">
          <h2>全屏效果预览 (对比)</h2>
          <button className="tiny-chip" type="button" onClick={onHome}>
            返回首页
          </button>
        </div>
        <div
          className={`preview-stage-container ${isSinglePreview ? 'single-preview' : ''}`}
        >
          {/* Left: Original Image */}
          {!isSinglePreview && (
          <div className="preview-stage" style={{ position: 'relative' }}>
            {originalImage ? (
              <>
                <RefreshingImage 
                  src={getDisplayImageSrc(originalImage)} 
                  alt="Original Portrait" 
                  className="full-preview-img"
                />
                <div style={{ position: 'absolute', top: 10, left: 10, background: 'rgba(0,0,0,0.6)', padding: '4px 8px', borderRadius: '4px', color: '#fff', fontSize: '12px' }}>
                  原图
                </div>
              </>
            ) : (
              <div className="preview-label">
                <strong>暂无原始图像</strong>
              </div>
            )}
          </div>
          )}

          {/* Right: Current/Style Transferred Image */}
          <div
            className="preview-stage"
            style={{
              position: 'relative',
            }}
          >
            {currentImage ? (
              <>
                <RefreshingImage 
                  src={getDisplayImageSrc(currentImage)} 
                  alt="Style Transferred Portrait" 
                  className="full-preview-img"
                  style={{ cursor: 'zoom-in' }}
                  onClick={openFullscreenPreview}
                />
                <div style={{ position: 'absolute', top: 10, left: 10, background: 'rgba(0,0,0,0.6)', padding: '4px 8px', borderRadius: '4px', color: '#fff', fontSize: '12px' }}>
                  效果图
                </div>
                
                {/* HUB Style Logo Overlay (Bottom Left) */}
                <div className="preview-hud-logo">
                  <div className="hud-logo-mark">
                    {branding.logo ? <img src={branding.logo} alt={`${branding.appName} logo`} /> : null}
                  </div>
                  <div className="hud-logo-text">
                    <span className="brand">{branding.previewWatermark.brand}</span>
                    <span className="product">{branding.previewWatermark.product}</span>
                  </div>
                </div>

                {/* Download Button (Top Right) */}
                <button 
                  className="preview-download-btn" 
                  onClick={() => setShowQR(true)}
                  title="下载图片"
                  style={{
                    position: 'absolute',
                    top: '10px',
                    right: '10px',
                    width: '44px',
                    height: '44px',
                    borderRadius: '50%',
                    background: 'rgba(0, 85, 255, 0.9)',
                    border: 'none',
                    color: '#fff',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer',
                    boxShadow: '0 4px 12px rgba(0, 85, 255, 0.4)',
                    zIndex: 10
                  }}
                >
                  <DownloadIcon />
                </button>
              </>
            ) : (
              <div className="preview-label">
                <strong>暂无生成图像</strong>
              </div>
            )}
          </div>
        </div>

        <div className="style-switcher">
          <h3>风格转换</h3>
          <div className="style-grid">
            {stylePrompts.length === 0 ? (
              <div style={{ color: 'rgba(255,255,255,0.5)', padding: '10px' }}>暂无风格配置，请在后台管理添加</div>
            ) : (
              stylePrompts.map((item) => (
                <button 
                  key={item.id} 
                  className="style-chip" 
                  type="button"
                  onClick={() => handleStyleTransfer(item.id, item.name)}
                  disabled={isTransferring}
                >
                  {item.name}
                </button>
              ))
            )}
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
              <img 
                src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(getAbsoluteUrl(currentImage))}`} 
                alt="Download QR Code" 
              />
            </div>
            <p>使用手机扫描二维码，即可保存您的 AI 证件照</p>
          </div>
        </div>
      )}

      {/* Fullscreen Overlay */}
      {isFullscreen && (
        <div 
          className="fullscreen-overlay" 
          onClick={closeFullscreenPreview} 
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 9999,
            background: '#000',
            display: 'flex',
            alignItems: 'stretch',
            justifyContent: 'stretch',
            cursor: 'zoom-out',
            overflow: 'hidden'
          }}
        >
          <button
            type="button"
            aria-label="关闭全屏预览"
            onClick={(event) => {
              event.stopPropagation();
              closeFullscreenPreview();
            }}
            style={{
              position: 'fixed',
              top: '18px',
              right: '18px',
              zIndex: 10000,
              width: '44px',
              height: '44px',
              borderRadius: '50%',
              border: '1px solid rgba(255,255,255,0.22)',
              background: 'rgba(255,255,255,0.12)',
              color: '#fff',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              cursor: 'pointer',
              backdropFilter: 'blur(12px)'
            }}
          >
            <XIcon />
          </button>
          <RefreshingImage 
            src={getDisplayImageSrc(currentImage)} 
            alt="Fullscreen" 
            onClick={(event) => event.stopPropagation()}
            style={{
              width: '100vw',
              height: '100dvh',
              objectFit: 'contain',
              display: 'block'
            }} 
          />
          <div style={{
            position: 'fixed',
            left: '24px',
            bottom: '20px',
            color: 'rgba(255,255,255,0.74)',
            background: 'rgba(0,0,0,0.35)',
            border: '1px solid rgba(255,255,255,0.12)',
            borderRadius: '999px',
            padding: '8px 12px',
            fontSize: '13px',
            backdropFilter: 'blur(12px)'
          }}>
            Esc 或点击空白区域退出全屏
          </div>
        </div>
      )}

      {/* Full Screen "Generating" Overlay */}
      {isTransferring && (
        <div className="generating-overlay" style={{
          position: 'fixed',
          inset: 0,
          zIndex: 9999,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'rgba(10, 15, 28, 0.9)',
          backdropFilter: 'blur(20px)',
          color: 'white'
        }}>
          <div className="generating-content" style={{ textAlign: 'center' }}>
            <div className="loading-spinner-large" style={{ 
              width: '80px', 
              height: '80px', 
              border: '4px solid rgba(255,255,255,0.1)',
              borderTop: '4px solid #0055ff',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite',
              margin: '0 auto 2rem'
            }} />
            <h2 style={{ fontSize: '2rem', marginBottom: '1rem', fontWeight: '600' }}>风格转换中</h2>
            <p style={{ color: 'rgba(255,255,255,0.6)', fontSize: '1.1rem' }}>
              正在利用 AI 重绘图像风格，请稍候...
            </p>
          </div>
        </div>
      )}
    </SimpleFrame>
  );
}
