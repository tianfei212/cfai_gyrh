import React, { useState, useEffect } from 'react';
import { SimpleFrame } from '../components/Layout';
import { buildScreenTitle, DEFAULT_BRANDING } from '../config/branding';
import { fetchApi } from '../services/api';

const SKILL_PROVIDER_OPTIONS = [
  { value: 'wan', label: 'Wan' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'qwen', label: 'Qwen' },
  { value: '302-gpt-image', label: '302 GPT Image' },
];

function SkillEditModal({ item, onClose, onSaved }) {
  const isNew = !item.id;
  const [saving, setSaving] = useState(false);
  const [loadingContent, setLoadingContent] = useState(!isNew);
  
  const [formData, setFormData] = useState({
    name: item.name || '',
    provider: item.provider || 'wan',
    content: item.content || '',
    is_active: item.is_active || false,
  });

  useEffect(() => {
    if (!isNew && !formData.content) {
      const fetchDetail = async () => {
        try {
          const res = await fetchApi(`/api/v1/skills/${item.id}`);
          setFormData(prev => ({ ...prev, content: res.content || '' }));
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
  }, [item.id, isNew, formData.content]);

  const handleChange = (field, value) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleSave = async () => {
    if (!formData.name || !formData.provider || !formData.content) {
      alert('名称、提供商和内容均不能为空');
      return;
    }
    
    setSaving(true);
    try {
      if (isNew) {
        await fetchApi('/api/v1/skills', {
          method: 'POST',
          body: JSON.stringify(formData)
        });
      } else {
        await fetchApi(`/api/v1/skills/${item.id}`, {
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
          <h3 style={{ margin: 0 }}>{isNew ? '新建 SKILL 技能' : `修改 SKILL 技能 - ${item.name || item.id}`}</h3>
          <button className="mini-outline" onClick={onClose} style={{ cursor: 'pointer' }}>关闭</button>
        </div>

        {loadingContent ? (
          <div style={{ padding: '40px', textAlign: 'center' }}>加载内容中...</div>
        ) : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
              <div>
                <h4 style={{ margin: '0 0 10px 0' }}>名称</h4>
                <input 
                  value={formData.name} 
                  onChange={(e) => handleChange('name', e.target.value)}
                  style={{ width: '100%', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px' }}
                  placeholder="如: Wan 默认技能"
                />
              </div>
              <div>
                <h4 style={{ margin: '0 0 10px 0' }}>提供商 (Provider)</h4>
                <select 
                  value={formData.provider} 
                  onChange={(e) => handleChange('provider', e.target.value)}
                  style={{ width: '100%', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px' }}
                >
                  {SKILL_PROVIDER_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
              </div>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <h4 style={{ margin: '0 0 10px 0' }}>技能内容 (Content)</h4>
              <textarea 
                value={formData.content} 
                onChange={(e) => handleChange('content', e.target.value)}
                style={{ width: '100%', height: '300px', padding: '10px', background: 'rgba(0,0,0,0.3)', border: '1px solid rgba(255,255,255,0.1)', color: '#fff', borderRadius: '6px', resize: 'vertical', fontFamily: 'monospace' }}
                placeholder="在此输入系统提示词/技能配置..."
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
              <label htmlFor="is_active_checkbox" style={{ cursor: 'pointer' }}>设为激活状态 (同 Provider 只能有一个激活)</label>
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

export function SkillManagerScreen({ onHome, onHistory, onLogout, onToggleModel, model, branding = DEFAULT_BRANDING }) {
  const [skills, setSkills] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editingItem, setEditingItem] = useState(null);
  const [filterProvider, setFilterProvider] = useState(''); // empty string means all

  const fetchSkills = async () => {
    try {
      setLoading(true);
      const query = filterProvider ? `?provider=${filterProvider}` : '';
      const data = await fetchApi(`/api/v1/skills${query}`);
      setSkills(data.items || []);
    } catch (err) {
      console.error('Failed to fetch skills:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSkills();
  }, [filterProvider]);

  const handleDelete = async (id) => {
    if (!window.confirm('确认删除该 SKILL 技能吗？')) return;
    try {
      await fetchApi(`/api/v1/skills/${id}`, { method: 'DELETE' });
      fetchSkills();
    } catch (err) {
      alert('删除失败: ' + err.message);
    }
  };

  const handleSetActive = async (row) => {
    try {
      const fullItem = await fetchApi(`/api/v1/skills/${row.id}`);
      await fetchApi(`/api/v1/skills/${row.id}`, { 
        method: 'PUT',
        body: JSON.stringify({ 
          name: fullItem.name,
          content: fullItem.content,
          provider: fullItem.provider,
          is_active: true 
        })
      });
      fetchSkills();
    } catch (err) {
      alert('激活失败: ' + err.message);
    }
  };

  return (
    <SimpleFrame 
      title={buildScreenTitle(branding, 'SKILL 管理')}
      branding={branding}
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section table-panel">
        <div className="section-topline">
          <h2>SKILL 技能列表</h2>
          <div className="chip-row">
            <select 
              value={filterProvider}
              onChange={(e) => setFilterProvider(e.target.value)}
              style={{ padding: '6px 12px', background: 'rgba(255,255,255,0.1)', border: 'none', color: '#fff', borderRadius: '4px', outline: 'none' }}
            >
              <option value="">全部 Provider</option>
              {SKILL_PROVIDER_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
            <button className="tiny-chip active" type="button" onClick={() => setEditingItem({ provider: filterProvider || 'wan' })}>
              + 新建
            </button>
            <button className="tiny-chip" type="button" onClick={fetchSkills}>刷新</button>
          </div>
        </div>
        <div className="table-shell">
          <div className="table-header table-grid" style={{ gridTemplateColumns: '80px 200px 100px 80px 180px 1fr' }}>
            <span>编号</span>
            <span>名称</span>
            <span>提供商</span>
            <span>状态</span>
            <span>更新时间</span>
            <span>操作</span>
          </div>
          {loading ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>加载中...</div>
          ) : skills.length === 0 ? (
            <div style={{ padding: '20px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>暂无 SKILL 数据</div>
          ) : (
            skills.map((row) => (
              <div className="table-row table-grid" style={{ gridTemplateColumns: '80px 200px 100px 80px 180px 1fr' }} key={row.id}>
                <span>{row.id}</span>
                <span title={row.name} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{row.name}</span>
                <span style={{ textTransform: 'capitalize' }}>{row.provider}</span>
                <span style={{ color: row.is_active ? '#4ade80' : 'rgba(255,255,255,0.4)' }}>
                  {row.is_active ? '激活' : '未激活'}
                </span>
                <span>{new Date(row.updated_at).toLocaleString()}</span>
                <div className="table-actions">
                  {!row.is_active && (
                    <button className="mini-outline" type="button" onClick={() => handleSetActive(row)}>设为激活</button>
                  )}
                  <button className="mini-outline" type="button" onClick={() => setEditingItem(row)}>编辑</button>
                  <button className="mini-outline" type="button" onClick={() => handleDelete(row.id)}>删除</button>
                </div>
              </div>
            ))
          )}
        </div>
      </section>
      {editingItem && (
        <SkillEditModal 
          item={editingItem} 
          onClose={() => setEditingItem(null)} 
          onSaved={() => {
            setEditingItem(null);
            fetchSkills();
          }} 
        />
      )}
    </SimpleFrame>
  );
}
