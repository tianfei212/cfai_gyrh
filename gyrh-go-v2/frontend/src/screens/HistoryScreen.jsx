import React, { useState, useEffect } from 'react';
import { SimpleFrame } from '../components/Layout';
import { SearchIcon, ChevronLeftIcon, ChevronRightIcon } from '../components/Icons';
import { fetchApi } from '../services/api';

export function HistoryScreen({ onHome, onHistory, onLogout, onToggleModel, model }) {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const limit = 12;

  const fetchHistory = async (pageNumber) => {
    try {
      setLoading(true);
      const offset = (pageNumber - 1) * limit;
      const data = await fetchApi(`/api/v1/images?limit=${limit}&offset=${offset}`);
      // 修复 bug：映射后端数据为前端需要的字段格式
      const mappedHistory = (data.images || []).map(img => ({
        id: img.id,
        // 这里使用新的 thumbnail API，动态传入需要的分辨率
        url: `/api/v1/images/thumbnail?url=${encodeURIComponent(`/api/v1/images/view?id=${img.id}`)}&w=400&h=400`,
        rawUrl: `/api/v1/images/view?id=${img.id}`,
        provider: img.style_transform,
        created_at: img.created_at
      }));
      setHistory(mappedHistory);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to fetch history:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHistory(page);
  }, [page]);

  const totalPages = Math.ceil(total / limit) || 1;

  const handlePrev = () => {
    if (page > 1) setPage(page - 1);
  };

  const handleNext = () => {
    if (page < totalPages) setPage(page + 1);
  };

  return (
    <SimpleFrame 
      title="AI Smart Portrait · 历史记录管理"
      onHome={onHome}
      onHistory={onHistory}
      onLogout={onLogout}
      onToggleModel={onToggleModel}
      model={model}
    >
      <section className="glass-section history-panel">
        <div className="section-stack">
          <h2>生成记录（全部）</h2>
          <div className="chip-row compact">
            <button className="tiny-chip active" type="button" onClick={() => fetchHistory(1)}>刷新</button>
          </div>
        </div>
        <div className="history-grid">
          {loading ? (
            <div style={{ gridColumn: '1 / -1', padding: '40px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>加载中...</div>
          ) : history.length === 0 ? (
             <div style={{ gridColumn: '1 / -1', padding: '40px', textAlign: 'center', color: 'rgba(255,255,255,0.6)' }}>暂无生成记录</div>
          ) : (
            history.map((card) => (
              <div
                key={card.id}
                className="history-card"
                style={{ backgroundImage: card.url ? `url(${card.url})` : 'none', backgroundSize: 'cover', backgroundPosition: 'center' }}
                title={`模型: ${card.provider || '未知'} | 时间: ${new Date(card.created_at).toLocaleString()}`}
                onClick={() => window.open(card.rawUrl || card.url, '_blank')}
              />
            ))
          )}
        </div>
        <div className="history-footer">
          <button className="slider-button fill" type="button" onClick={handlePrev} disabled={page === 1}>
            <ChevronLeftIcon />
          </button>
          <div className="pager-badge">{page} / {totalPages}</div>
          <button className="slider-button fill" type="button" onClick={handleNext} disabled={page === totalPages}>
            <ChevronRightIcon />
          </button>
        </div>
      </section>
    </SimpleFrame>
  );
}
