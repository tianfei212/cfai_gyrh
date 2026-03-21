import React, { useMemo, useState } from 'react';
import { Copy, Download, Link2 } from 'lucide-react';

interface ImageDownloadPathBuilderProps {
  initialImagePath?: string;
  initialBaseUrl?: string;
}

export const buildImageDownloadUrl = (baseUrl: string, imagePath: string) => {
  const rawBase = baseUrl.trim();
  const rawPath = imagePath.trim();
  if (!rawPath) return '';
  if (/^https?:\/\//i.test(rawPath)) return rawPath;
  const normalizedBase = rawBase || (typeof window !== 'undefined' ? window.location.origin : '');
  if (!normalizedBase) return '';
  try {
    return new URL(rawPath.startsWith('/') ? rawPath : `/${rawPath}`, normalizedBase).toString();
  } catch {
    return '';
  }
};

export const ImageDownloadPathBuilder: React.FC<ImageDownloadPathBuilderProps> = ({
  initialImagePath = '',
  initialBaseUrl = ''
}) => {
  const [baseUrl, setBaseUrl] = useState(initialBaseUrl);
  const [imagePath, setImagePath] = useState(initialImagePath);
  const [copied, setCopied] = useState(false);

  const fullDownloadUrl = useMemo(() => buildImageDownloadUrl(baseUrl, imagePath), [baseUrl, imagePath]);

  const copyFullUrl = async () => {
    if (!fullDownloadUrl) return;
    await navigator.clipboard.writeText(fullDownloadUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="w-full rounded-2xl border border-white/10 bg-zinc-900/60 backdrop-blur-md p-5 flex flex-col gap-4">
      <div className="flex items-center gap-2 text-white">
        <Link2 className="w-5 h-5 text-indigo-400" />
        <h3 className="text-lg font-semibold">图片下载完整路径拼接</h3>
      </div>

      <input
        value={baseUrl}
        onChange={(e) => setBaseUrl(e.target.value)}
        placeholder="站点地址（可选），例如 https://mqia.chinafilmai.com"
        className="w-full rounded-xl border border-zinc-700 bg-zinc-950/80 px-4 py-3 text-zinc-100 outline-none focus:border-indigo-500"
      />

      <input
        value={imagePath}
        onChange={(e) => setImagePath(e.target.value)}
        placeholder="选中图片路径，例如 /old_pic/img_20260301.png"
        className="w-full rounded-xl border border-zinc-700 bg-zinc-950/80 px-4 py-3 text-zinc-100 outline-none focus:border-indigo-500"
      />

      <div className="rounded-xl border border-zinc-800 bg-zinc-950/70 p-3 text-zinc-300 break-all text-sm">
        {fullDownloadUrl || '请输入图片路径后将自动生成可直接下载的完整 URL'}
      </div>

      <div className="flex flex-wrap gap-3">
        <button
          onClick={copyFullUrl}
          disabled={!fullDownloadUrl}
          className="px-4 py-2 rounded-lg border border-zinc-600 hover:border-zinc-400 text-zinc-100 disabled:opacity-40 flex items-center gap-2"
        >
          <Copy className="w-4 h-4" />
          {copied ? '已复制' : '复制完整路径'}
        </button>

        {fullDownloadUrl && (
          <a
            href={fullDownloadUrl}
            target="_blank"
            rel="noreferrer"
            download
            className="px-4 py-2 rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white font-medium flex items-center gap-2"
          >
            <Download className="w-4 h-4" />
            直接下载图片
          </a>
        )}
      </div>
    </div>
  );
};
