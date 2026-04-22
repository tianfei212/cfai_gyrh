import React, { useRef, useState, useEffect } from 'react';
import { WorkbenchLayout, HeaderIcon, HistorySidebar } from '../components/Layout';
import { HomeIcon, StackIcon, ExitIcon, PlusIcon, ImageIcon, RefreshIcon, ChevronLeftIcon, ChevronRightIcon, XIcon, CameraIcon } from '../components/Icons';
import { fetchApi } from '../services/api';

export function DashboardScreen({ onHome, onHistory, onBackgrounds, onLogout, onToggleModel, onCapture, model }) {
  const fileInputRef = useRef(null);
  const [uploadedImage, setUploadedImage] = useState(null);
  const [isDragging, setIsDragging] = useState(false);
  const [backgrounds, setBackgrounds] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchBackgrounds = async () => {
    try {
      setLoading(true);
      const data = await fetchApi('/api/v1/background-prompts');
      setBackgrounds(data.items || data.prompts || []);
    } catch (err) {
      console.error('Failed to fetch backgrounds:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBackgrounds();
  }, []);

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleUseImage = (image) => (e) => {
    e.stopPropagation();
    if (onCapture) {
      onCapture(image);
    }
  };

  const handleRemoveImage = (e) => {
    e.stopPropagation();
    setUploadedImage(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const processFile = (file) => {
    if (file && file.type.startsWith('image/')) {
      const imageUrl = URL.createObjectURL(file);
      setUploadedImage(imageUrl);
      console.log('File processed:', file.name);
    }
  };

  const handleFileChange = (event) => {
    const file = event.target.files?.[0];
    processFile(file);
  };

  const handleDragOver = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const file = e.dataTransfer.files?.[0];
    processFile(file);
  };

  return (
    <WorkbenchLayout
      title="AI Smart Portrait · Apple Glass"
      headerActions={
        <>
          <HeaderIcon label={model === 'W' ? 'W' : 'G'} onClick={onToggleModel} />
          <HeaderIcon icon={<HomeIcon />} onClick={onHome} />
          <HeaderIcon icon={<StackIcon />} onClick={onHistory} />
          <HeaderIcon icon={<ExitIcon />} onClick={onLogout} />
        </>
      }
      rightSidebar={
        <HistorySidebar />
      }
    >
      <section className="glass-section hero-workspace">
        <div className="section-topline">
          <h2>快速选择场景</h2>
          <span>1 / 19</span>
        </div>
        <div 
          className={`upload-stage ${isDragging ? 'dragging' : ''} ${uploadedImage ? 'has-image' : ''}`}
          onClick={handleUploadClick}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          style={{ 
            cursor: 'pointer'
          }}
        >
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            style={{ display: 'none' }}
            accept="image/*"
          />
          
          {uploadedImage ? (
            <>
              <img 
                src={uploadedImage} 
                alt="Uploaded background" 
                style={{ 
                  width: '100%', 
                  height: 'auto', 
                  maxHeight: '70vh',
                  objectFit: 'contain',
                  display: 'block'
                }} 
              />
              <button 
                className="close-stage-button"
                onClick={handleRemoveImage}
                type="button"
                aria-label="Remove image"
              >
                <XIcon />
              </button>
              <div className="hud-action-overlay">
                <button 
                  className="hud-use-button"
                  onClick={handleUseImage(uploadedImage)}
                  type="button"
                >
                  <CameraIcon />
                  <span>使用</span>
                </button>
              </div>
            </>
          ) : (
            <>
              <div className="upload-badge">
                <PlusIcon />
              </div>
              <h3>点击或拖拽上传背景图</h3>
              <p>支持 JPG / PNG / WebP，建议 4K 高清图</p>
            </>
          )}
        </div>
      </section>

      <section className="glass-section gallery-section">
        <div className="section-topline">
          <h2>背景图库</h2>
          <div className="topbar-actions">
            <button className="ghost-pill icon-pill" type="button" onClick={fetchBackgrounds}>
              <RefreshIcon />
            </button>
            <button className="ghost-pill icon-pill" type="button" onClick={onBackgrounds}>
              <ImageIcon />
            </button>
          </div>
        </div>
        <div className="gallery-grid">
          {loading ? (
             <div style={{ gridColumn: '1 / -1', padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>加载中...</div>
          ) : backgrounds.length === 0 ? (
             <div style={{ gridColumn: '1 / -1', padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>暂无背景图数据</div>
          ) : (
            backgrounds.map((card) => (
              <article
                key={card.id}
                className="gallery-card"
                onClick={handleUseImage(card)}
                style={{ cursor: 'pointer', backgroundImage: card.image_url ? `url(${card.image_url})` : 'none', backgroundSize: 'cover', backgroundPosition: 'center' }}
                title={model === 'W' ? card.wan_prompt : card.gemini_prompt}
              >
                <span style={{ background: 'rgba(0,0,0,0.5)', padding: '2px 8px', borderRadius: '4px' }}>{card.name || `背景 ${card.id}`}</span>
              </article>
            ))
          )}
        </div>
        <div className="footer-slider">
          <button className="slider-button" type="button">
            <ChevronLeftIcon />
          </button>
          <button className="slider-button" type="button">
            <ChevronRightIcon />
          </button>
        </div>
      </section>
    </WorkbenchLayout>
  );
}
