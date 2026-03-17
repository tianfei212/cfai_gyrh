import React, { useState, useEffect, useRef } from 'react';
import { Image, Loader2, AlertCircle, RefreshCw, ChevronLeft, ChevronRight } from 'lucide-react';
import { logToServer } from '../utils/logger';
import siteConfig from '../siteConfig.json';

interface MediaItem {
  id: string;
  thumbUrl: string;
  url: string;
  createTime: string;
  dimensions: string;
  prompt: string;
}

interface ApiResponse {
  code: number;
  message: string;
  data: {
    items: MediaItem[];
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

interface PhotoGalleryProps {
  onSelectImage: (imageUrl: string) => void;
}

export const PhotoGallery: React.FC<PhotoGalleryProps> = ({ onSelectImage }) => {
  const [images, setImages] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(5);
  const [totalPages, setTotalPages] = useState(1);
  const [loadingMore, setLoadingMore] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const isPagingRef = useRef(false);
  const prevScrollLeftRef = useRef(0);

  // Build proxy-safe BASE_URL and API URL to avoid TLS errors with IP HTTPS
  const rawApiUrl = siteConfig.gallery.apiUrl;
  let BASE_URL = '';
  try {
    const u = new URL(rawApiUrl);
    const isIpHost = /^\d+\.\d+\.\d+\.\d+$/.test(u.hostname);
    if (isIpHost) {
      BASE_URL = '/gallery';
    } else {
      BASE_URL = u.origin;
    }
  } catch {
    BASE_URL = '/gallery';
  }

  const buildPageUrl = (p: number) => {
    try {
      const u = new URL(rawApiUrl);
      u.searchParams.set('page', String(p));
      u.searchParams.set('pageSize', String(pageSize));
      const isIpHost = /^\d+\.\d+\.\d+\.\d+$/.test(u.hostname);
      if (isIpHost) {
        return `${BASE_URL}${u.pathname}?${u.searchParams.toString()}`;
      } else {
        return u.toString();
      }
    } catch {
      const basePath = rawApiUrl.startsWith('/') ? rawApiUrl : `/gallery${rawApiUrl}`;
      let normalized = basePath;
      if (normalized.includes('?')) {
        normalized = normalized
          .replace(/([?&])page=\d+/i, `$1page=${p}`)
          .replace(/([?&])pageSize=\d+/i, `$1pageSize=${pageSize}`);
        if (!/page=\d+/i.test(normalized)) normalized += `&page=${p}`;
        if (!/pageSize=\d+/i.test(normalized)) normalized += `&pageSize=${pageSize}`;
      } else {
        normalized += `?page=${p}&pageSize=${pageSize}`;
      }
      return normalized;
    }
  };

  useEffect(() => {
    fetchImages(1);
  }, []);

  const fetchImages = async (targetPage: number) => {
    try {
      if (targetPage === 1) setLoading(true);
      setLoadingMore(targetPage !== 1);
      setError(null);
      logToServer("Gallery: Fetching images");
      const response = await fetch(buildPageUrl(targetPage));
      
      if (!response.ok) {
        throw new Error('Failed to fetch images');
      }

      const data: ApiResponse = await response.json();
      
      if (data.code === 0 && data.data.items) {
        logToServer(`Gallery: Fetched ${data.data.items.length} images`);
        setImages(data.data.items.slice(0, pageSize));
        setPage(data.data.page || targetPage);
        setTotalPages(data.data.totalPages || 1);
      } else {
        throw new Error(data.message || 'Invalid data format');
      }
    } catch (err: any) {
      logToServer("Gallery: Fetch Error", { error: err.message }, "ERROR");
      console.error('Error fetching gallery:', err);
      setError('无法加载图片库');
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  const SCROLL_OFFSET = 60;

  const onScroll = async () => {
    const el = scrollRef.current;
    if (!el || loadingMore || isPagingRef.current) return;
    const prev = prevScrollLeftRef.current;
    const delta = el.scrollLeft - prev;
    prevScrollLeftRef.current = el.scrollLeft;
    const nearRight = el.scrollLeft + el.clientWidth >= el.scrollWidth - 40;
    const nearLeft = el.scrollLeft <= 40;
    if (nearRight && delta > 0 && page < totalPages) {
      isPagingRef.current = true;
      await fetchImages(page + 1);
      if (scrollRef.current) {
        scrollRef.current.scrollTo({ left: SCROLL_OFFSET, behavior: 'smooth' });
      }
      setTimeout(() => { isPagingRef.current = false; }, 500);
      return;
    }
    if (nearLeft && delta < 0 && page > 1) {
      isPagingRef.current = true;
      await fetchImages(page - 1);
      if (scrollRef.current) {
        const el2 = scrollRef.current;
        const maxScroll = el2.scrollWidth - el2.clientWidth;
        el2.scrollTo({ left: maxScroll - SCROLL_OFFSET, behavior: 'smooth' });
      }
      setTimeout(() => { isPagingRef.current = false; }, 500);
      return;
    }
  };

  const handleImageClick = async (item: MediaItem) => {
    try {
      logToServer("Gallery: User selected image", { id: item.id, prompt: item.prompt });
      const fullUrl = `${BASE_URL}${item.url}`;
      const response = await fetch(fullUrl);
      const blob = await response.blob();
      const reader = new FileReader();
      reader.onloadend = () => {
        const base64data = reader.result as string;
        onSelectImage(base64data);
      };
      reader.readAsDataURL(blob);
    } catch (err: any) {
      logToServer("Gallery: Image Load Error", { error: err.message }, "ERROR");
      console.error('Error loading image:', err);
    }
  };

  if (loading && images.length === 0) {
    return (
      <div className="w-full flex items-center justify-center py-12 text-zinc-500">
        <Loader2 className="w-6 h-6 animate-spin mr-2" />
        <span>正在加载图库...</span>
      </div>
    );
  }

  if (error && images.length === 0) {
    return (
      <div className="w-full flex items-center justify-center py-12 text-red-400/80">
        <AlertCircle className="w-5 h-5 mr-2" />
        <span>{error}</span>
        <button 
          onClick={() => fetchImages(page)}
          className="ml-4 px-3 py-1 bg-zinc-800 hover:bg-zinc-700 rounded text-xs text-white transition-colors"
        >
          重试
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex items-center justify-between mb-4 lg:mb-6 px-1 shrink-0">
        <h3 className="text-lg lg:text-2xl 2xl:text-3xl font-semibold text-zinc-200 flex items-center gap-3">
          <Image className="w-5 h-5 lg:w-7 lg:h-7 text-indigo-400" />
          {siteConfig.gallery.title}
          <button 
            onClick={() => fetchImages(1)} 
            disabled={loading}
            className={`ml-2 p-2 rounded-full hover:bg-zinc-800 text-zinc-400 hover:text-white transition-all flex items-center justify-center ${loading ? 'animate-spin opacity-50 cursor-not-allowed' : ''}`}
            title="刷新图库"
          >
            <RefreshCw className="w-4 h-4 lg:w-5 lg:h-5" />
          </button>
        </h3>
        <span className="text-xs lg:text-sm text-zinc-500 font-mono">第 {page} / {totalPages} 页</span>
      </div>
      
      <div className="flex-1 min-h-0 relative">
        <div
          ref={scrollRef}
          className="grid grid-cols-5 gap-4 lg:gap-6 2xl:gap-8 h-full items-start overflow-hidden content-start"
        >
          {images.map((item) => (
            <div
              key={item.id}
              className="w-full aspect-square"
            >
              <div
                onClick={() => handleImageClick(item)}
                className="group relative w-full h-full rounded-xl overflow-hidden cursor-pointer bg-zinc-800 border-2 border-zinc-700/50 hover:border-indigo-500 hover:shadow-[0_0_20px_rgba(99,102,241,0.2)] transition-all"
              >
                <img
                  src={`${BASE_URL}${item.thumbUrl}`}
                  alt={item.prompt}
                  className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
                  loading="lazy"
                />
                <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                  <Image className="w-8 h-8 text-white drop-shadow-lg" />
                </div>
              </div>
            </div>
          ))}
          {loading && Array.from({ length: pageSize - images.length }).map((_, i) => (
            <div key={`skeleton-${i}`} className="w-full aspect-square rounded-xl bg-zinc-800/50 animate-pulse" />
          ))}
        </div>
      </div>

      <div className="flex justify-between items-center mt-4 lg:mt-6 shrink-0 gap-4">
        <button
          onClick={() => fetchImages(Math.max(1, page - 1))}
          disabled={loading || loadingMore || page <= 1}
          className="flex-1 py-3 lg:py-4 rounded-xl bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 disabled:hover:bg-zinc-800 text-white transition-all flex items-center justify-center active:scale-95"
        >
          <ChevronLeft className="w-5 h-5 lg:w-6 lg:h-6" />
        </button>
        <button
          onClick={() => fetchImages(Math.min(totalPages, page + 1))}
          disabled={loading || loadingMore || page >= totalPages}
          className="flex-1 py-3 lg:py-4 rounded-xl bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 disabled:hover:bg-zinc-800 text-white transition-all flex items-center justify-center active:scale-95"
        >
          <ChevronRight className="w-5 h-5 lg:w-6 lg:h-6" />
        </button>
      </div>
    </div>
  );
};
