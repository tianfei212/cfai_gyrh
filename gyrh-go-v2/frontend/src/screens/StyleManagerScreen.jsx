import React, { useState, useEffect } from 'react';
import { SimpleFrame } from '../components/Layout';
import { fetchApi } from '../services/api';

function StyleEditModal({ item, onClose, onSaved }) {
  const isNew = !item.id;
  const [saving, setSaving] = useState(false);
  const [loadingContent, setLoadingContent] = useState(!isNew);
  
  const [formData, setFormData] = useState({
    name: item.name || '',
    prompt: item.prompt || '',
    negative_prompt: item.negative_prompt || '',
    is_active: item.is_active !== undefined ? item.is_active : true,
  });

  useEffect(() => {
    if (!isNew && !formData.prompt) {
      const fetchDetail = async () => {
        try {
          const res = await fetchApi(`/api/v1/style-prompts/${item.id}`);
          setFormData(prev => ({ 
            ...prev, 
            prompt: res.prompt || '',
            negative_prompt: res.negative_prompt || ''
          }));
        } catch (err) {
          alert('获取详情失败: ' + err.message);
        } finally {
          setLoadingContent(false);
        }
      };
      fetchDetail();
    } else {
      setLoadingContent(false);
    }
  }, [item.id, isNew, formData.prompt]);

  const handleChange = (field, value) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleSave = async () => {
    if (!formData.name || !formData.prompt) {
      alert('名称和提示词均不能为空');
      return;
    }
    
    setSaving(true);
    try {
      if (isNew) {
        await fetchApi('/api/v1/style-prompts', {
          method: 'POST',
          body: JSON.stringify(formData)
        });
      } else {
        await fetchApi(`/api/v1/style-prompts/${item.id}`, {
          method: 'PUT',
          body: JSON.stringify(formData)
        });
      }
      alert('保存成功！');
      if (onSaved) onSaved();
    } catch (err) {
      alert('保存失败: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay" style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.8)', zIndex: 9999, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
      <div className="modal-content" style={{ background: '#1e2025', width: '80%', maxWidth: '800px', maxHeight: '90vh', overflowY: 'auto', borderRadius: '12px', padding: '24px', color: '#fff' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
          <h3 style={{ margin: 0 }}>{isNew ? '新建风格转换提示词' : `修改风格转换提示词 - ${item.name || item.id}`}</h3>
          <button className="mini-outline" onClick={onClose} style={{ cursor: 'pointer' }}>关闭</button>
        </div>

        {loadingContent ? (
          <div style={{ padding: '40px', textAlign: 'center' }}>加载内容中...</div>
        ) : (
          <>
            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ margin: '0 0 10px 0' }}>名称 (展示在前端界面的按钮上)</h4>
              <input 
                value={formData.name} 
                onChange={(e) => handleChange('name', e.target.value)}
                style={{ width: '100%', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px' }}
                placeholder="如: 电影感"
              />
            </div>

            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ margin: '0 0 10px 0' }}>提示词 (Prompt)</h4>
              <textarea 
                value={formData.prompt} 
                onChange={(e) => handleChange('prompt', e.target.value)}
                style={{ width: '100%', height: '150px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
                placeholder="在此输入正向提示词..."
              />
            </div>

            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ margin: '0 0 10px 0' }}>反向提示词 (Negative Prompt - 可选)</h4>
              <textarea 
                value={formData.negative_prompt} 
                onChange={(e) => handleChange('negative_prompt', e.target.value)}
                style={{ width: '100%', height: '100px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical' }}
                placeholder="在此输入反向提示词..."
              />
            </div>

            <div style={{ marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '10px' }}>
              <input 
                type="checkbox" 
                id="is_active_checkbox"
                checked={formData.is_active} 
                onChange={(e) => handleChange('is_active', e.target.checked)}
                style={{ width: '18px', height: '18px' }}
              />
              <label htmlFor="is_active_checkbox" style={{ cursor: 'pointer' }}>在前端展示此风格按钮</label>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
              <button className="primary-btn" onClick={handleSave} disabled={saving}>
                {saving ? '保存中...' : '保存'}
              </button>
              <button className="secondary-btn" onClick={onClose} disabled={saving}>取消</button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export function StyleManagerScreen({ onHome, onHistory, onLogout, onToggleModel, model }) {
  const [prompts, setPrompts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editingItem, setEditingItem] = useState(null);

  const fetchPrompts = async () => {
    try {
      setLoading(true);
      const data = await fetchApi('/api/v1/style-prompts');
      setPrompts(data || []);
    } catch (err) {
      console.error('Failed to fetch style prompts:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPrompts();
  }, []);

  const handleDelete = async (id) => {
    if (!window.confirm('确认删除该风格提示词吗？')) return;
    try {
      await fetchApi(`/api/v1/style-prompts/${id}`, { method: 'DELETE' });
      fetchPrompts();
    } catch (err) {
      alert('删除失败: ' + err.message);
    }
  };

  const handleToggleActive = async (row) => {
    try {
      await fetchApi(`/api/v1/style-prompts/${row.id}`, { 
        method: 'PUT',
        body: JSON.stringify({ 
          name: row.name,
          prompt: row.prompt,
          negative_prompt: row.negative_prompt,
          is_active: !row.is_active 
        })
      });
      fetchPrompts();
    } catch (err) {
      alert('切换状态失败: ' + err.message);
    }
  };

  return (
    <SimpleFrame 
      title="AI Smart Portrait · 风格转换管理"
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section table-panel">
        <div className="section-topline">
          <h2>风格转换提示词列表</h2>
          <div className="chip-row">
            <button className="tiny-chip active" type="button" onClick={() => setEditingItem({})}>
              + 新建
            </button>
            <button className="tiny-chip" type="button" onClick={fetchPrompts}>刷新</button>
          </div>
        </div>
        <div className="table-shell">
          <div className="table-header table-grid" style={{ gridTemplateColumns: '80px 200px 100px 180px 1fr' }}>
            <span>编号</span>
            <span>名称</span>
            <span>状态</span>
            <span>更新时间</span>
            <span>操作</span>
          </div>
          {loading ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>加载中...</div>
          ) : prompts.length === 0 ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>暂无风格数据</div>
          ) : (
            prompts.map((row) => (
              <div className="table-row table-grid" style={{ gridTemplateColumns: '80px 200px 100px 180px 1fr' }} key={row.id}>
                <span>{row.id}</span>
                <span title={row.name} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{row.name}</span>
                <span style={{ color: row.is_active ? '#4ade80' : 'rgba(255,255,255,0.4)' }}>
                  {row.is_active ? '展示中' : '已隐藏'}
                </span>
                <span>{new Date(row.updated_at).toLocaleString()}</span>
                <div className="table-actions">
                  <button className="mini-outline" type="button" onClick={() => handleToggleActive(row)}>
                    {row.is_active ? '隐藏' : '展示'}
                  </button>
                  <button className="mini-outline" type="button" onClick={() => setEditingItem(row)}>编辑</button>
                  <button className="mini-outline" type="button" onClick={() => handleDelete(row.id)}>删除</button>
                </div>
              </div>
            ))
          )}
        </div>
      </section>
      {editingItem && (
        <StyleEditModal 
          item={editingItem} 
          onClose={() => setEditingItem(null)} 
          onSaved={() => {
            setEditingItem(null);
            fetchPrompts();
          }} 
        />
      )}
    </SimpleFrame>
  );
}
