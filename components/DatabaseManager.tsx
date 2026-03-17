import React, { useState, useEffect } from 'react';
import { Trash2, RefreshCw, X, CheckSquare, Square, Eye, FolderOpen } from 'lucide-react';
import { ResultViewer } from './ResultViewer';

interface FileItem {
  name: string;
  url: string;
  timestamp: number;
}

interface DatabaseManagerProps {
  onClose: () => void;
}

export const DatabaseManager: React.FC<DatabaseManagerProps> = ({ onClose }) => {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [previewImage, setPreviewImage] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchFiles();
  }, []);

  const fetchFiles = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/list-images');
      if (res.ok) {
        const data = await res.json();
        setFiles(data);
      }
    } catch (err) {
      console.error("Failed to list images", err);
    } finally {
      setLoading(false);
    }
  };

  const toggleSelect = (name: string) => {
    const newSet = new Set(selectedFiles);
    if (newSet.has(name)) {
      newSet.delete(name);
    } else {
      newSet.add(name);
    }
    setSelectedFiles(newSet);
  };

  const selectAll = () => {
    if (selectedFiles.size === files.length) {
      setSelectedFiles(new Set());
    } else {
      setSelectedFiles(new Set(files.map(f => f.name)));
    }
  };

  const handleDelete = async () => {
    if (selectedFiles.size === 0) return;
    if (!confirm(`确定要删除选中的 ${selectedFiles.size} 张图片吗？`)) return;

    setLoading(true);
    try {
      const res = await fetch('/api/delete-images', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ names: Array.from(selectedFiles) })
      });
      
      if (res.ok) {
        setSelectedFiles(new Set());
        fetchFiles();
      } else {
        alert("删除失败");
      }
    } catch (err) {
      console.error("Delete failed", err);
      alert("删除出错");
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {/* Invisible Trigger Area */}
      <div className="fixed top-0 left-0 right-0 h-4 z-[51] peer bg-transparent" />

      <div className="fixed top-0 left-0 right-0 z-50 bg-zinc-950/95 backdrop-blur-xl flex flex-col transition-transform duration-300 ease-in-out shadow-2xl h-[85vh] -translate-y-full hover:translate-y-0 peer-hover:translate-y-0 border-b border-white/10">
        {/* Header */}
        <div className="flex items-center justify-between px-8 py-6 border-b border-white/10 bg-zinc-900/50">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-indigo-500/20 rounded-xl">
              <FolderOpen className="w-6 h-6 text-indigo-400" />
            </div>
            <div>
              <h2 className="text-2xl font-bold text-white">图库管理</h2>
              <p className="text-zinc-400 text-sm">共 {files.length} 张图片</p>
            </div>
          </div>
          <button 
            onClick={onClose}
            className="p-2 hover:bg-white/10 rounded-full transition-colors"
          >
            <X className="w-8 h-8 text-zinc-400 hover:text-white" />
          </button>
        </div>

        {/* Toolbar */}
        <div className="px-8 py-4 border-b border-white/5 bg-zinc-900/30 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button 
              onClick={selectAll}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 transition-colors text-zinc-300"
            >
              {selectedFiles.size === files.length && files.length > 0 ? (
                <CheckSquare className="w-4 h-4 text-indigo-400" />
              ) : (
                <Square className="w-4 h-4" />
              )}
              全选
            </button>
            
            <button 
              onClick={fetchFiles}
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-zinc-800 hover:bg-zinc-700 transition-colors text-zinc-300 disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </button>
          </div>

          <div className="flex items-center gap-4">
            <span className="text-zinc-500 text-sm">
              已选择 {selectedFiles.size} 项
            </span>
            <button 
              onClick={handleDelete}
              disabled={selectedFiles.size === 0 || loading}
              className="flex items-center gap-2 px-6 py-2 rounded-lg bg-red-500/10 border border-red-500/20 hover:bg-red-500/20 text-red-400 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Trash2 className="w-4 h-4" />
              删除选中
            </button>
          </div>
        </div>

        {/* Grid Content */}
        <div className="flex-1 overflow-y-auto p-8">
          {files.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center text-zinc-500 gap-4">
              <FolderOpen className="w-16 h-16 opacity-20" />
              <p>暂无图片</p>
            </div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8 gap-6">
              {files.map((file) => (
                <div 
                  key={file.name}
                  className={`group relative aspect-[3/4] rounded-xl overflow-hidden border-2 transition-all cursor-pointer ${
                    selectedFiles.has(file.name) 
                      ? 'border-indigo-500 ring-2 ring-indigo-500/20' 
                      : 'border-white/5 hover:border-white/20'
                  }`}
                  onClick={() => toggleSelect(file.name)}
                >
                  <img 
                    src={file.url} 
                    alt={file.name}
                    className="w-full h-full object-cover"
                    loading="lazy"
                  />
                  
                  {/* Overlay Gradient */}
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/40 opacity-0 group-hover:opacity-100 transition-opacity" />

                  {/* Selection Indicator */}
                  <div className="absolute top-3 left-3">
                    <div className={`w-6 h-6 rounded-md border flex items-center justify-center transition-colors ${
                      selectedFiles.has(file.name)
                        ? 'bg-indigo-500 border-indigo-500'
                        : 'bg-black/40 border-white/30 group-hover:bg-black/60'
                    }`}>
                      {selectedFiles.has(file.name) && <CheckSquare className="w-4 h-4 text-white" />}
                    </div>
                  </div>

                  {/* Preview Button */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setPreviewImage(file.url);
                    }}
                    className="absolute top-3 right-3 p-2 rounded-full bg-black/40 hover:bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity backdrop-blur-sm"
                    title="预览"
                  >
                    <Eye className="w-4 h-4" />
                  </button>

                  {/* Info */}
                  <div className="absolute bottom-0 left-0 right-0 p-3 opacity-0 group-hover:opacity-100 transition-opacity">
                    <p className="text-xs text-zinc-300 truncate font-mono">{file.name}</p>
                    <p className="text-[10px] text-zinc-500">
                      {new Date(file.timestamp).toLocaleString()}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Preview Modal */}
        {previewImage && (
          <ResultViewer 
            image={previewImage}
            isProcessing={false}
            onEdit={() => {}} // Read-only in manager
            onUpscale={() => {}}
            onDownload={() => {
               const link = document.createElement('a');
               link.href = previewImage;
               link.download = previewImage.split('/').pop() || 'download.png';
               link.click();
            }}
            onClose={() => setPreviewImage(null)}
          />
        )}
      </div>
    </>
  );
};
