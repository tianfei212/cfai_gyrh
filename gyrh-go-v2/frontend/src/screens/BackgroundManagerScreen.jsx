import React, { useState, useEffect } from 'react';
import { SimpleFrame } from '../components/Layout';
import { RefreshingImage } from '../components/RefreshingImage';
import { buildScreenTitle, DEFAULT_BRANDING } from '../config/branding';
import { fetchApi } from '../services/api';
import {
  BACKGROUND_MANAGER_PAGE_SIZE,
  buildBackgroundPromptListUrl,
  getPageAfterRefresh,
  getTotalPages,
} from '../utils/backgroundPagination';
import { appendImageCacheBucket } from '../utils/imageThumbs';

function buildBackgroundThumbnailUrl(item, width, height) {
  let url = '';
  if (item?.image_asset_id) {
    url = `/api/v1/images/thumbnail?asset_id=${encodeURIComponent(item.image_asset_id)}&w=${width}&h=${height}`;
  } else if (item?.image_url) {
    url = `/api/v1/images/thumbnail?url=${encodeURIComponent(item.image_url)}&w=${width}&h=${height}`;
  }
  return appendImageCacheBucket(url);
}

function BackgroundEditModal({ item, onClose, onSaved }) {
  const [isEditing, setIsEditing] = useState(false);
  const [translating, setTranslating] = useState(false);
  const [saving, setSaving] = useState(false);

  const [formData, setFormData] = useState({
    name: item.name || '',
    wan_prompt_zh: item.wan_prompt_zh || '',
    wan_prompt: item.wan_prompt || '',
    gemini_prompt_zh: item.gemini_prompt_zh || '',
    gemini_prompt: item.gemini_prompt || '',
    gpt_prompt_zh: item.gpt_prompt_zh || '',
    gpt_prompt: item.gpt_prompt || '',
  });

  const handleChange = (field, value) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleTranslate = async () => {
    setTranslating(true);
    try {
      const data = await fetchApi('/api/v1/background-prompts/sync-english', {
        method: 'POST',
        body: JSON.stringify({
          wan_prompt_zh: formData.wan_prompt_zh,
          wan_negative_prompt_zh: item.wan_negative_prompt_zh || '',
          gemini_prompt_zh: formData.gemini_prompt_zh,
          gemini_negative_prompt_zh: item.gemini_negative_prompt_zh || '',
          gpt_prompt_zh: formData.gpt_prompt_zh,
          gpt_negative_prompt_zh: item.gpt_negative_prompt_zh || ''
        })
      });
      setFormData(prev => ({
        ...prev,
        wan_prompt: data.wan_prompt_en || prev.wan_prompt,
        gemini_prompt: data.gemini_prompt_en || prev.gemini_prompt,
        gpt_prompt: data.gpt_prompt_en || prev.gpt_prompt,
      }));
      alert('翻译完成！');
    } catch (err) {
      alert('翻译失败: ' + err.message);
    } finally {
      setTranslating(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await fetchApi(`/api/v1/background-prompts/${item.id}`, {
        method: 'PUT',
        body: JSON.stringify(formData)
      });
      alert('保存成功！');
      setIsEditing(false);
      if (onSaved) onSaved();
    } catch (err) {
      alert('保存失败: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay" style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.8)', zIndex: 9999, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
      <div className="modal-content" style={{ background: '#1e2025', overflowY: 'auto', borderRadius: '12px', padding: '24px', color: '#fff' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <h3 style={{ margin: 0 }}>修改提示词 - {item.name || item.id}</h3>
          <button className="mini-outline" onClick={onClose} style={{ cursor: 'pointer' }}>关闭</button>
        </div>

        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '24px' }}>
          {item.image_url ? (
            <RefreshingImage src={buildBackgroundThumbnailUrl(item, 400, 400)} alt="background" style={{ maxHeight: '300px', maxWidth: '100%', objectFit: 'contain', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.1)' }} />
          ) : (
            <div style={{ height: '200px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'rgba(255,255,255,0.5)' }}>无图片</div>
          )}
        </div>

        <div style={{ marginBottom: '20px' }}>
          <h4 style={{ margin: '0 0 10px 0' }}>图片名称</h4>
          <input 
            type="text"
            value={formData.name} 
            onChange={(e) => handleChange('name', e.target.value)}
            disabled={!isEditing}
            style={{ width: '100%', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px' }}
          />
        </div>

        <div className="responsive-form-grid">
          <div>
            <h4 style={{ margin: '0 0 10px 0' }}>Wan 中文提示词</h4>
            <textarea 
              value={formData.wan_prompt_zh} 
              onChange={(e) => handleChange('wan_prompt_zh', e.target.value)}
              disabled={!isEditing}
              style={{ width: '100%', height: '120px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
            />
          </div>
          <div>
            <h4 style={{ margin: '0 0 10px 0' }}>Wan 英文提示词</h4>
            <textarea 
              value={formData.wan_prompt} 
              onChange={(e) => handleChange('wan_prompt', e.target.value)}
              disabled={!isEditing}
              style={{ width: '100%', height: '120px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
            />
          </div>
          <div>
            <h4 style={{ margin: '0 0 10px 0' }}>Gemini 中文提示词</h4>
            <textarea 
              value={formData.gemini_prompt_zh} 
              onChange={(e) => handleChange('gemini_prompt_zh', e.target.value)}
              disabled={!isEditing}
              style={{ width: '100%', height: '120px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
            />
          </div>
          <div>
            <h4 style={{ margin: '0 0 10px 0' }}>Gemini 英文提示词</h4>
            <textarea 
              value={formData.gemini_prompt} 
              onChange={(e) => handleChange('gemini_prompt', e.target.value)}
              disabled={!isEditing}
              style={{ width: '100%', height: '120px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
            />
          </div>
          <div>
            <h4 style={{ margin: '0 0 10px 0' }}>GPT 中文提示词</h4>
            <textarea
              value={formData.gpt_prompt_zh}
              onChange={(e) => handleChange('gpt_prompt_zh', e.target.value)}
              disabled={!isEditing}
              style={{ width: '100%', height: '120px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
            />
          </div>
          <div>
            <h4 style={{ margin: '0 0 10px 0' }}>GPT 英文提示词</h4>
            <textarea
              value={formData.gpt_prompt}
              onChange={(e) => handleChange('gpt_prompt', e.target.value)}
              disabled={!isEditing}
              style={{ width: '100%', height: '120px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
            />
          </div>
        </div>

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
          {!isEditing ? (
            <button className="primary-btn" onClick={() => setIsEditing(true)}>修改</button>
          ) : (
            <>
              <button className="secondary-btn" onClick={handleTranslate} disabled={translating}>
                {translating ? '翻译中...' : '翻译'}
              </button>
              <button className="primary-btn" onClick={handleSave} disabled={saving}>
                {saving ? '保存中...' : '保存'}
              </button>
              <button className="secondary-btn" onClick={() => setIsEditing(false)} disabled={saving}>取消</button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function OriginalImagePreviewModal({ item, onClose }) {
  if (!item?.image_url) return null;

  return (
    <div className="image-preview-overlay" onClick={onClose}>
      <div className="image-preview-modal" onClick={(event) => event.stopPropagation()}>
        <div className="image-preview-header">
          <div>
            <p>原图预览</p>
            <h3>{item.name || `背景 ${item.id}`}</h3>
          </div>
          <button className="mini-outline" type="button" onClick={onClose}>关闭</button>
        </div>
        <div className="image-preview-stage">
          <RefreshingImage src={item.image_url} alt={item.name || '背景原图'} />
        </div>
      </div>
    </div>
  );
}

export function BackgroundManagerScreen({ onHome, onHistory, onLogout, onToggleModel, backgroundCache, model, branding = DEFAULT_BRANDING }) {
  const [backgrounds, setBackgrounds] = useState([]);
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [previewingItem, setPreviewingItem] = useState(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const fileInputRef = React.useRef(null);
  const limit = BACKGROUND_MANAGER_PAGE_SIZE;

  const fetchBackgrounds = async (targetPage = page) => {
    try {
      setLoading(true);
      const data = await fetchApi(buildBackgroundPromptListUrl(targetPage, limit));
      setBackgrounds(data.items || data.prompts || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to fetch backgrounds:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBackgrounds();
  }, [page]);

  const refreshBackgrounds = async ({ resetToFirstPage = false } = {}) => {
    backgroundCache?.invalidateAll();
    const nextPage = getPageAfterRefresh(page, { resetToFirstPage });
    if (nextPage !== page) {
      setPage(nextPage);
      return;
    }
    await fetchBackgrounds(nextPage);
  };

  const handlePrevPage = () => {
    setPage(prev => Math.max(1, prev - 1));
  };

  const handleNextPage = () => {
    const totalPages = getTotalPages(total, limit);
    setPage(prev => Math.min(totalPages, prev + 1));
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确认删除该背景吗？')) return;
    try {
      await fetchApi(`/api/v1/background-prompts/${id}`, { method: 'DELETE' });
      refreshBackgrounds();
    } catch (err) {
      alert('删除失败: ' + err.message);
    }
  };

  const handleImportClick = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  const handleSyncRemote = async () => {
    setSyncing(true);
    try {
      const result = await fetchApi('/api/v1/background-prompts/sync-remote', {
        method: 'POST',
        body: JSON.stringify({})
      });
      await refreshBackgrounds({ resetToFirstPage: true });
      const failures = result.failures?.length ? `，失败 ${result.failed} 条` : '';
      alert(`同步完成：新增 ${result.imported} 条，跳过 ${result.skipped} 条${failures}`);
    } catch (err) {
      alert('同步失败: ' + err.message);
    } finally {
      setSyncing(false);
    }
  };

  const handleFileChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setImporting(true);
    try {
      const reader = new FileReader();
      reader.onload = async (event) => {
        const base64Str = event.target.result.split(',')[1];
        try {
          const res = await fetchApi('/api/v1/background-prompts/import', {
            method: 'POST',
            body: JSON.stringify({ image: base64Str, name: file.name.split('.')[0] })
          });
          console.log('Import response:', res);
          refreshBackgrounds({ resetToFirstPage: true });
        } catch (apiErr) {
          alert('导入失败: ' + apiErr.message);
        } finally {
          setImporting(false);
          e.target.value = ''; // clear input
        }
      };
      reader.onerror = (error) => {
        console.error('File read error:', error);
        setImporting(false);
      };
      reader.readAsDataURL(file);
    } catch (err) {
      console.error('File read error:', err);
      setImporting(false);
    }
  };

  return (
    <SimpleFrame 
      title={buildScreenTitle(branding, '背景管理')}
      branding={branding}
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
            <input 
              type="file" 
              ref={fileInputRef} 
              style={{ display: 'none' }} 
              accept="image/jpeg, image/png, image/webp" 
              onChange={handleFileChange} 
            />
            <button className="tiny-chip" type="button" onClick={handleImportClick} disabled={importing}>
              {importing ? '导入中...' : '导入'}
            </button>
            <button className="tiny-chip" type="button" onClick={handleSyncRemote} disabled={syncing || importing}>
              {syncing ? '同步中...' : '同步'}
            </button>
            <button className="tiny-chip active" type="button" onClick={() => refreshBackgrounds()} disabled={loading || syncing || importing}>
              {loading ? '刷新中...' : '本地库'}
            </button>
          </div>
        </div>
        <div className="table-shell">
          <div className="table-header table-grid">
            <span>图片名称</span>
            <span>缩略图</span>
            <span>Wan 提示词</span>
            <span>Gemini 提示词</span>
            <span>GPT 提示词</span>
            <span>操作</span>
          </div>
          {loading ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>加载中...</div>
          ) : backgrounds.length === 0 ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>暂无背景图数据</div>
          ) : (
            backgrounds.map((row) => (
              <div className="table-row table-grid" key={row.id}>
                <span data-label="图片名称" title={row.name} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{row.name || '-'}</span>
                <div 
                  className="thumb-swatch" 
                >
                  {row.image_url || row.image_asset_id ? (
                    <RefreshingImage src={buildBackgroundThumbnailUrl(row, 150, 150)} alt={row.name || '背景缩略图'} />
                  ) : null}
                </div>
                <span data-label="Wan 提示词" title={row.wan_prompt_zh} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{row.wan_prompt_zh || '-'}</span>
                <span data-label="Gemini 提示词" title={row.gemini_prompt_zh} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{row.gemini_prompt_zh || '-'}</span>
                <span data-label="GPT 提示词" title={row.gpt_prompt_zh} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{row.gpt_prompt_zh || '-'}</span>
                <div className="table-actions">
                  {row.image_url && <button className="mini-outline" type="button" onClick={() => setPreviewingItem(row)}>查看原图</button>}
                  <button className="mini-outline" type="button" onClick={() => setEditingItem(row)}>编辑</button>
                  <button className="mini-outline" type="button" onClick={() => handleDelete(row.id)}>删除</button>
                </div>
              </div>
            ))
          )}
        </div>
        <div className="table-pager">
          <button className="mini-outline" type="button" onClick={handlePrevPage} disabled={loading || page <= 1}>
            上一页
          </button>
          <span>
            第 {page} / {getTotalPages(total, limit)} 页 · 共 {total} 条
          </span>
          <button className="mini-outline" type="button" onClick={handleNextPage} disabled={loading || page >= getTotalPages(total, limit)}>
            下一页
          </button>
        </div>
      </section>
      {editingItem && (
        <BackgroundEditModal 
          item={editingItem} 
          onClose={() => setEditingItem(null)} 
          onSaved={() => {
            setEditingItem(null);
            refreshBackgrounds();
          }} 
        />
      )}
      {previewingItem && (
        <OriginalImagePreviewModal
          item={previewingItem}
          onClose={() => setPreviewingItem(null)}
        />
      )}
    </SimpleFrame>
  );
}
