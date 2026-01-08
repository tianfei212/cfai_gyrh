import React, { useState, useEffect } from 'react';
import { Image, Loader2, AlertCircle, RefreshCw } from 'lucide-react';
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

  // Build proxy-safe BASE_URL and API URL to avoid TLS errors with IP HTTPS
  const rawApiUrl = siteConfig.gallery.apiUrl;
  let BASE_URL = '';
  let apiUrl = rawApiUrl;
  try {
    const u = new URL(rawApiUrl);
    const isIpHost = /^\d+\.\d+\.\d+\.\d+$/.test(u.hostname);
    if (isIpHost) {
      BASE_URL = '/gallery';
      apiUrl = `${BASE_URL}${u.pathname}${u.search}`;
    } else {
      BASE_URL = u.origin;
      apiUrl = rawApiUrl;
    }
  } catch {
    BASE_URL = '/gallery';
    apiUrl = rawApiUrl.startsWith('/') ? rawApiUrl : `/gallery${rawApiUrl}`;
  }

  useEffect(() => {
    fetchImages();
  }, []);

  const fetchImages = async () => {
    try {
      setLoading(true);
      setError(null);
      logToServer("Gallery: Fetching images");
      const response = await fetch(apiUrl);
      
      if (!response.ok) {
        throw new Error('Failed to fetch images');
      }

      const data: ApiResponse = await response.json();
      
      if (data.code === 0 && data.data.items) {
        logToServer(`Gallery: Fetched ${data.data.items.length} images`);
        setImages(data.data.items);
      } else {
        throw new Error(data.message || 'Invalid data format');
      }
    } catch (err: any) {
      logToServer("Gallery: Fetch Error", { error: err.message }, "ERROR");
      console.error('Error fetching gallery:', err);
      setError('无法加载图片库');
    } finally {
      setLoading(false);
    }
  };

  const handleImageClick = async (item: MediaItem) => {
    try {
      logToServer("Gallery: User selected image", { id: item.id, prompt: item.prompt });
      // Convert relative URL to absolute
      const fullUrl = `${BASE_URL}${item.url}`;
      
      // Fetch the image and convert to Base64 to match existing app logic
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
      // Fallback: pass the URL directly if base64 conversion fails
      // Note: This might need CORS handling on the server side
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
          onClick={fetchImages}
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
            onClick={fetchImages} 
            disabled={loading}
            className={`ml-1 p-1.5 lg:p-2 rounded-full hover:bg-zinc-800 text-zinc-400 hover:text-white transition-all flex items-center justify-center ${loading ? 'animate-spin opacity-50 cursor-not-allowed' : ''}`}
            title="刷新图库"
          >
            <RefreshCw className="w-4 h-4 lg:w-5 lg:h-5 2xl:w-6 2xl:h-6" />
          </button>
        </h3>
        <span className="text-xs lg:text-sm 2xl:text-base text-zinc-500">点击图片即可作为背景使用</span>
      </div>
      
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8 gap-4 lg:gap-6 2xl:gap-8">
        {images.map((item) => (
          <div 
            key={item.id}
            onClick={() => handleImageClick(item)}
            className="group relative aspect-square rounded-xl overflow-hidden cursor-pointer bg-zinc-800 border border-zinc-700/50 hover:border-indigo-500/50 transition-all hover:shadow-lg hover:shadow-indigo-500/10"
          >
            <img 
              src={`${BASE_URL}${item.thumbUrl}`} 
              alt={item.prompt}
              className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
              loading="lazy"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex items-end p-3 lg:p-4">
              <p className="text-[10px] lg:text-xs 2xl:text-sm text-zinc-300 line-clamp-2 leading-tight">
                {item.prompt}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
