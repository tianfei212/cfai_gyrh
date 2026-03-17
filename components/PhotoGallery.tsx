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
  const [pageSize] = useState(20);
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
    <div className="w-full mt-12 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="flex items-center justify-between mb-4 px-1 lg:mb-6 lg:px-2">
        <h3 className="text-lg lg:text-xl 2xl:text-2xl font-semibold text-zinc-200 flex items-center gap-2 lg:gap-3">
          <Image className="w-5 h-5 lg:w-6 lg:h-6 2xl:w-8 2xl:h-8 text-indigo-400" />
          {siteConfig.gallery.title}
          <button 
            onClick={() => fetchImages(1)} 
            disabled={loading}
            className={`ml-1 p-1.5 lg:p-2 rounded-full hover:bg-zinc-800 text-zinc-400 hover:text-white transition-all flex items-center justify-center ${loading ? 'animate-spin opacity-50 cursor-not-allowed' : ''}`}
            title="刷新图库"
          >
            <RefreshCw className="w-4 h-4 lg:w-5 lg:h-5 2xl:w-6 2xl:h-6" />
          </button>
          <div className="ml-2 flex items-center gap-2">
            <button
              onClick={() => fetchImages(Math.max(1, page - 1))}
              disabled={loading || loadingMore || page <= 1}
              className={`flex items-center gap-1 px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs ${page <= 1 || loading || loadingMore ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <ChevronLeft className="w-4 h-4" />
              上一页
            </button>
            <button
              onClick={() => fetchImages(Math.min(totalPages, page + 1))}
              disabled={loading || loadingMore || page >= totalPages}
              className={`flex items-center gap-1 px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs ${page >= totalPages || loading || loadingMore ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              下一页
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </h3>
        <span className="text-xs lg:text-sm 2xl:text-base text-zinc-500">第 {page} / {totalPages} 页 · 每页 20 张</span>
      </div>
      <div
        ref={scrollRef}
        className="flex overflow-x-auto gap-4 lg:gap-6 2xl:gap-8 pb-2 snap-x snap-proximity"
      >
        {images.map((item) => (
          <div
            key={item.id}
            className="flex-none w-28 md:w-32 lg:w-36 2xl:w-44 snap-start"
          >
            <div
              onClick={() => handleImageClick(item)}
              className="group relative aspect-square rounded-xl overflow-hidden cursor-pointer bg-zinc-800 border border-zinc-700/50 hover:border-indigo-500/50 transition-all hover:shadow-lg hover:shadow-indigo-500/10"
            >
              <img
                src={`${BASE_URL}${item.thumbUrl}`}
                alt={item.prompt}
                className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                loading="lazy"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex items-end p-3 lg:p-4">
                <p className="text-[10px] lg:text-xs 2xl:text-sm text-zinc-300 line-clamp-2 leading-tight">
                  {item.prompt}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
      <div className="w-full mt-4 flex items-center justify-center gap-4">
        <button
          onClick={() => fetchImages(Math.max(1, page - 1))}
          disabled={loading || loadingMore || page <= 1}
          className={`flex items-center gap-1 px-3 py-1.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs ${page <= 1 || loading || loadingMore ? 'opacity-50 cursor-not-allowed' : ''}`}
        >
          <ChevronLeft className="w-4 h-4" />
          上一页
        </button>
        <span className="text-xs lg:text-sm text-zinc-400">
          第 {page} / {totalPages} 页
        </span>
        <button
          onClick={() => fetchImages(Math.min(totalPages, page + 1))}
          disabled={loading || loadingMore || page >= totalPages}
          className={`flex items-center gap-1 px-3 py-1.5 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 text-xs ${page >= totalPages || loading || loadingMore ? 'opacity-50 cursor-not-allowed' : ''}`}
        >
          下一页
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
      {loadingMore && (
        <div className="w-full flex items-center justify-center py-3 text-zinc-400 text-xs">
          正在加载下一页…
        </div>
      )}
    </div>
  );
};
