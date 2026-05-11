import React, { useState, useEffect } from 'react';
import { SimpleFrame } from '../components/Layout';
import { ChevronLeftIcon, ChevronRightIcon } from '../components/Icons';
import { fetchApi } from '../services/api';
import {
  buildHistoryPreviewPayload,
  buildHistoryTitle,
  mapGeneratedImagesToHistoryRecords,
} from '../utils/historyRecords';

export function HistoryScreen({ onHome, onHistory, onLogout, onToggleModel, model, onPreview }) {
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
      setHistory(mapGeneratedImagesToHistoryRecords(data.images || []));
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
          <h2>{buildHistoryTitle(total)}</h2>
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
                title={`生成时间: ${new Date(card.created_at).toLocaleString()}`}
                onClick={() => onPreview(buildHistoryPreviewPayload(card))}
                style={{ 
                  cursor: 'pointer',
                  aspectRatio: '16 / 9'
                }}
              >
                {card.url ? (
                  <img 
                    src={card.url} 
                    alt={`生成的图片 ${card.id}`}
                    style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block', borderRadius: 'inherit' }} 
                  />
                ) : (
                  <div style={{ width: '100%', minHeight: '8.45rem', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'rgba(255,255,255,0.3)' }}>
                    图片丢失
                  </div>
                )}
              </div>
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
