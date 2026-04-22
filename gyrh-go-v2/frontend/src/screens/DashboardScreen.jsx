import React, { useRef, useState } from 'react';
import { WorkbenchLayout, HeaderIcon, HistorySidebar } from '../components/Layout';
import { HomeIcon, StackIcon, ExitIcon, PlusIcon, ImageIcon, RefreshIcon, ChevronLeftIcon, ChevronRightIcon, XIcon, CameraIcon } from '../components/Icons';
import { galleryCards } from '../constants';

export function DashboardScreen({ onHome, onHistory, onBackgrounds, onLogout, onToggleModel, onCapture, model }) {
  const fileInputRef = useRef(null);
  const [uploadedImage, setUploadedImage] = useState(null);
  const [isDragging, setIsDragging] = useState(false);

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
            <button className="ghost-pill icon-pill" type="button">
              <RefreshIcon />
            </button>
            <button className="ghost-pill icon-pill" type="button" onClick={onBackgrounds}>
              <ImageIcon />
            </button>
          </div>
        </div>
        <div className="gallery-grid">
          {galleryCards.map((card) => (
            <article
              key={card.name}
              className={`gallery-card tone-${card.tone}`}
              onClick={handleUseImage(card)}
              style={{ cursor: 'pointer' }}
            >
              <span>{card.name}</span>
            </article>
          ))}
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
