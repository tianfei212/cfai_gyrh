import React, { useMemo, useState } from 'react';
import QRCode from 'qrcode';
import { Copy, Download, QrCode, RefreshCw } from 'lucide-react';

interface UrlQrCodeGeneratorProps {
  initialUrl?: string;
}

export const generateUrlQrCodeBase64 = async (url: string) => {
  return QRCode.toDataURL(url.trim(), {
    width: 360,
    margin: 2,
    errorCorrectionLevel: 'M'
  });
};

export const UrlQrCodeGenerator: React.FC<UrlQrCodeGeneratorProps> = ({ initialUrl = '' }) => {
  const [url, setUrl] = useState(initialUrl);
  const [qrDataUrl, setQrDataUrl] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);

  const normalizedUrl = useMemo(() => url.trim(), [url]);

  const generateQrCode = async () => {
    setError('');
    setCopied(false);
    if (!normalizedUrl) {
      setQrDataUrl('');
      setError('请输入完整 URL');
      return;
    }
    try {
      setIsGenerating(true);
      const value = await generateUrlQrCodeBase64(normalizedUrl);
      setQrDataUrl(value);
    } catch {
      setQrDataUrl('');
      setError('二维码生成失败，请检查 URL 是否正确');
    } finally {
      setIsGenerating(false);
    }
  };

  const copyUrl = async () => {
    if (!normalizedUrl) return;
    await navigator.clipboard.writeText(normalizedUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="w-full rounded-2xl border border-white/10 bg-zinc-900/60 backdrop-blur-md p-5 flex flex-col gap-4">
      <div className="flex items-center gap-2 text-white">
        <QrCode className="w-5 h-5 text-indigo-400" />
        <h3 className="text-lg font-semibold">URL 二维码生成</h3>
      </div>

      <input
        value={url}
        onChange={(e) => setUrl(e.target.value)}
        placeholder="请输入完整 URL，例如 https://mqia.chinafilmai.com/old_pic/img_20260301.png"
        className="w-full rounded-xl border border-zinc-700 bg-zinc-950/80 px-4 py-3 text-zinc-100 outline-none focus:border-indigo-500"
      />

      <div className="flex flex-wrap gap-3">
        <button
          onClick={generateQrCode}
          disabled={isGenerating}
          className="px-4 py-2 rounded-lg bg-indigo-500 hover:bg-indigo-400 text-white font-medium disabled:opacity-50 flex items-center gap-2"
        >
          <RefreshCw className={`w-4 h-4 ${isGenerating ? 'animate-spin' : ''}`} />
          生成二维码
        </button>

        <button
          onClick={copyUrl}
          disabled={!normalizedUrl}
          className="px-4 py-2 rounded-lg border border-zinc-600 hover:border-zinc-400 text-zinc-100 disabled:opacity-40 flex items-center gap-2"
        >
          <Copy className="w-4 h-4" />
          {copied ? '已复制' : '复制链接'}
        </button>

        {qrDataUrl && (
          <a
            href={qrDataUrl}
            download="url-qrcode.png"
            className="px-4 py-2 rounded-lg border border-zinc-600 hover:border-zinc-400 text-zinc-100 flex items-center gap-2"
          >
            <Download className="w-4 h-4" />
            下载二维码
          </a>
        )}
      </div>

      {error && <p className="text-red-400 text-sm">{error}</p>}

      {qrDataUrl && (
        <div className="rounded-xl border border-white/10 bg-white p-4 w-fit">
          <img src={qrDataUrl} alt="URL QR Code" className="w-48 h-48 object-contain" />
        </div>
      )}
    </div>
  );
};
