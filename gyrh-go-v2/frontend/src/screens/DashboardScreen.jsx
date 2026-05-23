import React, { useRef, useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { WorkbenchLayout, HeaderIcon, HistorySidebar } from '../components/Layout';
import { RefreshingImage } from '../components/RefreshingImage';
import { HomeIcon, StackIcon, ExitIcon, PlusIcon, ImageIcon, SearchIcon, RefreshIcon, ChevronLeftIcon, ChevronRightIcon, XIcon } from '../components/Icons';
import { DEFAULT_BRANDING } from '../config/branding';
import { fetchApi } from '../services/api';
import {
  buildCaptureBackgroundThumbnailUrl,
  buildFullImagePreviewUrl,
  buildImageThumbnailUrl,
  getImagePreloadUrls,
  preloadImages,
} from '../utils/imageThumbs';
import { getModelLabel, isGPTModel } from '../utils/modelProvider';
import { getTotalPages } from '../utils/backgroundPagination';

const DEFAULT_BACKGROUND_CATEGORY_PARENT = '场景';
const DEFAULT_BACKGROUND_CATEGORY_CHILD = '电影';

function formatCategoryLabel(category) {
  const parentName = category?.parent_name || 'default';
  const childName = category?.child_name || 'default';
  return `${parentName}/${childName}`;
}

function isDefaultWorkbenchCategory(category) {
  return category?.parent_name === DEFAULT_BACKGROUND_CATEGORY_PARENT && category?.child_name === DEFAULT_BACKGROUND_CATEGORY_CHILD;
}

export function DashboardScreen({ onHome, onHistory, onBackgrounds, onLogout, onToggleModel, onCapture, onPreview, backgroundCache, model, branding = DEFAULT_BRANDING }) {
  const fileInputRef = useRef(null);
  const backgroundRequestSeq = useRef(0);
  const [backgrounds, setBackgrounds] = useState([]);
  const [categories, setCategories] = useState([]);
  const [selectedCategoryId, setSelectedCategoryId] = useState(null);
  const [categoryPickerOpen, setCategoryPickerOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [previewingBackground, setPreviewingBackground] = useState(null);
  const limit = 9;
  const selectedCategory = categories.find(category => category.id === selectedCategoryId);
  const totalPages = getTotalPages(total, limit);
  const previewImageUrl = buildFullImagePreviewUrl({
    assetId: previewingBackground?.image_asset_id,
    imageUrl: previewingBackground?.image_url,
  });

  const fetchBackgrounds = async ({ force = false } = {}) => {
    if (!backgroundCache || selectedCategoryId === null) return;
    if (selectedCategoryId === 0) {
      setBackgrounds([]);
      setTotal(0);
      setLoading(false);
      return;
    }
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
      const nextCategories = data || [];
      const defaultCategory = nextCategories.find(isDefaultWorkbenchCategory);
      setCategories(nextCategories);
      setSelectedCategoryId(defaultCategory?.id || 0);
    } catch (err) {
      console.error('Failed to fetch background categories:', err);
      setSelectedCategoryId(0);
      setLoading(false);
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

  const handlePreviewBackground = (image) => (e) => {
    e.stopPropagation();
    setPreviewingBackground(image);
  };

  const processFile = (file) => {
    if (file && file.type.startsWith('image/')) {
      const imageUrl = URL.createObjectURL(file);
      if (onCapture) {
        onCapture(imageUrl);
      }
      console.log('File processed:', file.name);
    }
  };

  const handleFileChange = (event) => {
    const file = event.target.files?.[0];
    processFile(file);
  };

  const previewModal = previewingBackground && (
    <div className="image-preview-overlay" onClick={() => setPreviewingBackground(null)}>
      <div className="image-preview-modal" onClick={(event) => event.stopPropagation()}>
        <div className="image-preview-header">
          <div>
            <p>背景图预览</p>
            <h3>{previewingBackground.name || `背景 ${previewingBackground.id}`}</h3>
          </div>
          <button
            className="image-preview-close"
            type="button"
            onClick={() => setPreviewingBackground(null)}
            aria-label="关闭预览"
            title="关闭"
          >
            <XIcon />
          </button>
        </div>
        <div className="image-preview-stage">
          {previewImageUrl ? (
            <RefreshingImage src={previewImageUrl} alt={previewingBackground.name || '背景图预览'} />
          ) : (
            <div style={{ padding: '2%', color: 'rgba(255,255,255,0.65)' }}>暂无可预览图片</div>
          )}
        </div>
      </div>
    </div>
  );

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
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        style={{ display: 'none' }}
        accept="image/*"
      />

      <section className="glass-section gallery-section">
        <div className="section-topline">
          <h2>{selectedCategory ? `背景图库 · ${formatCategoryLabel(selectedCategory)}` : '背景图库'}</h2>
          <div className="topbar-actions">
            <button
              className="ghost-pill icon-pill liquid-glass-button"
              type="button"
              onClick={handleUploadClick}
              aria-label="上传背景图"
              title="上传背景图"
            >
              <PlusIcon />
            </button>
            <button className={`ghost-pill liquid-glass-button ${selectedCategoryId ? 'active' : ''}`} type="button" onClick={() => setCategoryPickerOpen(open => !open)}>
              类型
            </button>
            <button className="ghost-pill icon-pill liquid-glass-button" type="button" onClick={() => fetchBackgrounds({ force: true })}>
              <RefreshIcon />
            </button>
            <button className="ghost-pill icon-pill liquid-glass-button" type="button" onClick={onBackgrounds}>
              <ImageIcon />
            </button>
          </div>
        </div>
        {categoryPickerOpen && (
          <div className="chip-row compact" style={{ marginBottom: '1rem' }}>
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
                {(card.image_url || card.image_asset_id) && (
                  <button
                    className="gallery-preview-button"
                    type="button"
                    onClick={handlePreviewBackground(card)}
                    aria-label={`放大预览 ${card.name || `背景 ${card.id}`}`}
                    title="放大"
                  >
                    <SearchIcon />
                  </button>
                )}
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
      {previewModal && typeof document !== 'undefined' ? createPortal(previewModal, document.body) : previewModal}
    </WorkbenchLayout>
  );
}
