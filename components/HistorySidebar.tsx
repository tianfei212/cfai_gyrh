import React, { useState } from 'react';
import { FolderOpen, ChevronLeft, ChevronRight, Clock } from 'lucide-react';

interface FileItem {
  name: string;
  data: string;
  timestamp: number;
  isRemote?: boolean;
}

interface HistorySidebarProps {
  files: FileItem[];
  onSelect: (file: FileItem) => void;
  resultImage: string | null;
}

const ITEMS_PER_PAGE = 4;

export const HistorySidebar: React.FC<HistorySidebarProps> = ({ files, onSelect, resultImage }) => {
  const [page, setPage] = useState(1);
  const totalPages = Math.ceil(files.length / ITEMS_PER_PAGE);

  const displayedFiles = files.slice((page - 1) * ITEMS_PER_PAGE, page * ITEMS_PER_PAGE);

  const handlePrev = () => {
    if (page > 1) setPage(page - 1);
  };

  const handleNext = () => {
    if (page < totalPages) setPage(page + 1);
  };

  return (
    <div className="w-full flex flex-col bg-zinc-900/80 backdrop-blur-md border border-white/10 h-full rounded-2xl overflow-hidden shadow-2xl">
      <div className="p-4 lg:p-6 border-b border-white/10 bg-zinc-800/50 flex items-center justify-between shrink-0">
        <h2 className="font-semibold flex items-center gap-3 text-zinc-200 text-base lg:text-lg">
          <FolderOpen className="w-5 h-5 text-indigo-500" />
          历史记录 ({files.length})
        </h2>
        <span className="text-xs lg:text-sm text-zinc-500 font-mono">
          {page} / {totalPages || 1}
        </span>
      </div>

      <div className="flex-1 overflow-hidden p-4 lg:p-6 grid grid-rows-2 grid-cols-2 gap-4 lg:gap-6 content-start">
        {files.length === 0 ? (
          <div className="col-span-2 row-span-2 flex flex-col items-center justify-center text-zinc-600 gap-4 h-full border-2 border-dashed border-zinc-800 rounded-xl">
            <FolderOpen className="w-12 h-12 opacity-20" />
            <p className="text-sm">暂无历史记录</p>
          </div>
        ) : (
          displayedFiles.map((file, index) => (
            <div 
              key={index}
              onClick={() => onSelect(file)}
              className={`group relative rounded-xl overflow-hidden cursor-pointer border-2 transition-all w-full h-full aspect-[3/4] ${
                resultImage === file.data 
                  ? 'border-indigo-500 shadow-[0_0_20px_rgba(99,102,241,0.3)]' 
                  : 'border-transparent hover:border-zinc-500 hover:shadow-lg bg-zinc-950'
              }`}
            >
              <div className="w-full h-full relative">
                <img 
                  src={file.data} 
                  alt={file.name} 
                  className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" 
                  loading="lazy"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex flex-col justify-end p-4">
                  <p className="text-white font-medium text-sm truncate w-full mb-1">{file.name}</p>
                  <div className="flex items-center gap-1.5 text-zinc-400 text-xs">
                    <Clock className="w-3.5 h-3.5" />
                    <span>{new Date(file.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="p-4 lg:p-6 border-t border-white/10 bg-zinc-900/50 shrink-0 flex justify-between items-center gap-4">
        <button
          onClick={handlePrev}
          disabled={page <= 1}
          className="flex-1 py-3 lg:py-4 rounded-xl bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 disabled:hover:bg-zinc-800 text-white transition-all flex items-center justify-center active:scale-95"
        >
          <ChevronLeft className="w-5 h-5 lg:w-6 lg:h-6" />
        </button>
        <button
          onClick={handleNext}
          disabled={page >= totalPages}
          className="flex-1 py-3 lg:py-4 rounded-xl bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 disabled:hover:bg-zinc-800 text-white transition-all flex items-center justify-center active:scale-95"
        >
          <ChevronRight className="w-5 h-5 lg:w-6 lg:h-6" />
        </button>
      </div>
    </div>
  );
};
