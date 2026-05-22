import React, { useRef, useState, useEffect } from 'react';
import { WorkbenchLayout, HeaderIcon, HistorySidebar } from '../components/Layout';
import { RefreshingImage } from '../components/RefreshingImage';
import { HomeIcon, StackIcon, ExitIcon, PlusIcon, ImageIcon, RefreshIcon, ChevronLeftIcon, ChevronRightIcon, XIcon, CameraIcon } from '../components/Icons';
import { DEFAULT_BRANDING } from '../config/branding';
import {
  buildCaptureBackgroundThumbnailUrl,
  buildImageThumbnailUrl,
  getImagePreloadUrls,
  preloadImages,
} from '../utils/imageThumbs';
import { getModelLabel, isGPTModel } from '../utils/modelProvider';
import { getTotalPages } from '../utils/backgroundPagination';

function formatCategoryLabel(category) {
  const parentName = category?.parent_name || 'default';
  const childName = category?.child_name || 'default';
  return `${parentName}/${childName}`;
}

export function DashboardScreen({ onHome, onHistory, onBackgrounds, onLogout, onToggleModel, onCapture, onPreview, backgroundCache, model, branding = DEFAULT_BRANDING }) {
  const fileInputRef = useRef(null);
  const backgroundRequestSeq = useRef(0);
  const [uploadedImage, setUploadedImage] = useState(null);
  const [isDragging, setIsDragging] = useState(false);
  const [backgrounds, setBackgrounds] = useState([]);
  const [categories, setCategories] = useState([]);
  const [selectedCategoryId, setSelectedCategoryId] = useState(0);
  const [categoryPickerOpen, setCategoryPickerOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const limit = 6;
  const selectedCategory = categories.find(category => category.id === selectedCategoryId);
  const totalPages = getTotalPages(total, limit);

  const fetchBackgrounds = async ({ force = false } = {}) => {
    if (!backgroundCache) return;
    const requestId = backgroundRequestSeq.current + 1;
    backgroundRequestSeq.current = requestId;
    try {
      setLoading(true);
      const data = await backgroundCache.loadPage(page, { limit, force, categoryId: selectedCategoryId });
      if (requestId !== backgroundRequestSeq.current) {
        return;
      }
      const nextItems = data.items || data.prompts || [];
      setBackgrounds(nextItems);
      setTotal(data.total || 0);
      preloadImages(getImagePreloadUrls(nextItems));
    } catch (err) {
      if (requestId === backgroundRequestSeq.current) {
        console.error('Failed to fetch backgrounds:', err);
      }
    } finally {
      if (requestId === backgroundRequestSeq.current) {
        setLoading(false);
      }
    }
  };

  const fetchCategories = async () => {
    try {
      const data = await fetchApi('/api/v1/background-categories');
      setCategories(data || []);
    } catch (err) {
      console.error('Failed to fetch background categories:', err);
    }
  };

  useEffect(() => {
    fetchBackgrounds();
  }, [page, selectedCategoryId]);

  useEffect(() => {
    fetchCategories();
  }, []);

  const handlePrevPage = () => {
    if (page > 1) {
      setPage(page - 1);
    }
  };

  const handleNextPage = () => {
    if (page < totalPages) {
      setPage(page + 1);
    }
  };

  const handleSelectCategory = (categoryId) => {
    setSelectedCategoryId(categoryId);
    setPage(1);
    setCategoryPickerOpen(false);
  };

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleUseImage = (image) => (e) => {
    e.stopPropagation();
    if (typeof image === 'object') {
      preloadImages([buildCaptureBackgroundThumbnailUrl({
        assetId: image.image_asset_id,
        imageUrl: image.image_url,
      })]);
    }
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
      title={branding.appName}
      branding={branding}
      headerActions={
        <>
          <HeaderIcon label={getModelLabel(model)} onClick={onToggleModel} />
          <HeaderIcon icon={<HomeIcon />} onClick={onHome} />
          <HeaderIcon icon={<StackIcon />} onClick={onHistory} />
          <HeaderIcon icon={<ExitIcon />} onClick={onLogout} />
        </>
      }
      rightSidebar={
        <HistorySidebar onPreview={onPreview} />
      }
    >
      <section className="glass-section hero-workspace">
        <div className="section-topline">
          <h2>快速选择场景</h2>
          <span>{page} / {totalPages}</span>
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
          <h2>{selectedCategory ? `背景图库 · ${formatCategoryLabel(selectedCategory)}` : '背景图库'}</h2>
          <div className="topbar-actions">
            <button className={`ghost-pill ${selectedCategoryId ? 'active' : ''}`} type="button" onClick={() => setCategoryPickerOpen(open => !open)}>
              类型
            </button>
            <button className="ghost-pill icon-pill" type="button" onClick={() => fetchBackgrounds({ force: true })}>
              <RefreshIcon />
            </button>
            <button className="ghost-pill icon-pill" type="button" onClick={onBackgrounds}>
              <ImageIcon />
            </button>
          </div>
        </div>
        {categoryPickerOpen && (
          <div className="chip-row compact" style={{ marginBottom: '1rem' }}>
            <button
              className={`tiny-chip ${selectedCategoryId === 0 ? 'active' : ''}`}
              type="button"
              onClick={() => handleSelectCategory(0)}
            >
              全部背景
            </button>
            {categories.map((category) => (
              <button
                key={category.id}
                className={`tiny-chip ${selectedCategoryId === category.id ? 'active' : ''}`}
                type="button"
                onClick={() => handleSelectCategory(category.id)}
              >
                {formatCategoryLabel(category)}
              </button>
            ))}
          </div>
        )}
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
                style={{ cursor: 'pointer' }}
                title={isGPTModel(model) ? '302 GPT Image 通用融合 Skill' : model === 'W' ? card.wan_prompt : card.gemini_prompt}
              >
                {card.image_url || card.image_asset_id ? (
                  <RefreshingImage src={buildImageThumbnailUrl({ assetId: card.image_asset_id, imageUrl: card.image_url })} alt={card.name || `背景 ${card.id}`} />
                ) : null}
                <span style={{ background: 'rgba(0,0,0,0.5)', padding: '2px 8px', borderRadius: '4px' }}>{card.name || `背景 ${card.id}`}</span>
              </article>
            ))
          )}
        </div>
        <div className="footer-slider">
          <button 
            className="slider-button" 
            type="button" 
            onClick={handlePrevPage} 
            disabled={page === 1 || loading}
            style={{ opacity: (page === 1 || loading) ? 0.5 : 1, cursor: (page === 1 || loading) ? 'not-allowed' : 'pointer' }}
          >
            <ChevronLeftIcon />
          </button>
          <button 
            className="slider-button" 
            type="button" 
            onClick={handleNextPage} 
            disabled={page >= totalPages || loading}
            style={{ opacity: (page >= totalPages || loading) ? 0.5 : 1, cursor: (page >= totalPages || loading) ? 'not-allowed' : 'pointer' }}
          >
            <ChevronRightIcon />
          </button>
        </div>
      </section>
    </WorkbenchLayout>
  );
}
