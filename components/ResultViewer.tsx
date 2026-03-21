import React, { useState, useRef, useEffect } from 'react';
import { Copy, Download, QrCode, Zap, X, ChevronUp, ChevronDown } from 'lucide-react';
import { CountdownTimer } from './CountdownTimer';
import { blobToBase64 } from '../utils/fileUtils';
import { transcribeAudio } from '../services/geminiService';
import { buildImageDownloadUrl } from './ImageDownloadPathBuilder';
import { generateUrlQrCodeBase64 } from './UrlQrCodeGenerator';
import siteConfig from '../siteConfig.json';

interface ResultViewerProps {
  image: string;
  isProcessing: boolean;
  onEdit: (prompt: string) => void;
  onUpscale: () => void;
  onDownload: () => void;
  onClose?: () => void; // Add close prop
}

export const ResultViewer: React.FC<ResultViewerProps> = ({ 
  image, 
  isProcessing,
  onEdit, 
  onUpscale,
  onDownload,
  onClose
}) => {
  const [editPrompt, setEditPrompt] = useState('');
  const [isRecording, setIsRecording] = useState(false);
  const [isTranscribing, setIsTranscribing] = useState(false);
  const [isToolsOpen, setIsToolsOpen] = useState(true);
  const [isQrModalOpen, setIsQrModalOpen] = useState(false);
  const [qrImageBase64, setQrImageBase64] = useState('');
  const [downloadPageUrl, setDownloadPageUrl] = useState('');
  const [isPreparingQr, setIsPreparingQr] = useState(false);
  const [qrError, setQrError] = useState('');
  const [isCopied, setIsCopied] = useState(false);
  
  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const audioChunksRef = useRef<Blob[]>([]);
  
  // Audio Visualization Refs
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const animationFrameRef = useRef<number | null>(null);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (animationFrameRef.current) cancelAnimationFrame(animationFrameRef.current);
      if (audioContextRef.current && audioContextRef.current.state !== 'closed') {
        audioContextRef.current.close();
      }
    };
  }, []);

  const drawWaveform = () => {
    if (!canvasRef.current || !analyserRef.current) return;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    const analyser = analyserRef.current;
    if (!ctx) return;

    const bufferLength = analyser.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);

    const draw = () => {
      animationFrameRef.current = requestAnimationFrame(draw);
      analyser.getByteFrequencyData(dataArray);

      ctx.clearRect(0, 0, canvas.width, canvas.height);
      const width = canvas.width;
      const height = canvas.height;
      const barWidth = (width / bufferLength) * 2.5;
      let x = 0;

      for (let i = 0; i < bufferLength; i++) {
        const barHeight = (dataArray[i] / 255) * height * 0.8;
        ctx.fillStyle = `rgba(129, 140, 248, ${barHeight / height + 0.3})`;
        const y = (height - barHeight) / 2;
        ctx.beginPath();
        ctx.roundRect(x, y, barWidth - 1, barHeight, 5);
        ctx.fill();
        x += barWidth + 1;
      }
    };
    draw();
  };

  const startRecording = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const mediaRecorder = new MediaRecorder(stream);
      mediaRecorderRef.current = mediaRecorder;
      audioChunksRef.current = [];

      const audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
      audioContextRef.current = audioCtx;
      const analyser = audioCtx.createAnalyser();
      analyser.fftSize = 64;
      analyserRef.current = analyser;
      
      const source = audioCtx.createMediaStreamSource(stream);
      source.connect(analyser);

      if (canvasRef.current) {
         const rect = canvasRef.current.getBoundingClientRect();
         canvasRef.current.width = rect.width;
         canvasRef.current.height = rect.height;
         drawWaveform();
      }

      mediaRecorder.ondataavailable = (event) => {
        if (event.data.size > 0) audioChunksRef.current.push(event.data);
      };

      mediaRecorder.onstop = async () => {
        if (animationFrameRef.current) cancelAnimationFrame(animationFrameRef.current);
        if (audioContextRef.current) audioContextRef.current.close();
        
        const audioBlob = new Blob(audioChunksRef.current, { type: 'audio/webm' });
        stream.getTracks().forEach(track => track.stop());
        
        setIsTranscribing(true);
        try {
          const base64 = await blobToBase64(audioBlob);
          const text = await transcribeAudio(base64);
          if (text && text.trim()) {
            const trimmedText = text.trim();
            setEditPrompt(trimmedText);
            onEdit(trimmedText);
            setEditPrompt('');
          }
        } catch (error) {
          console.error("Transcription failed", error);
        } finally {
          setIsTranscribing(false);
        }
      };

      mediaRecorder.start();
      setIsRecording(true);
    } catch (err) {
      console.error("Failed to access microphone", err);
      alert("无法访问麦克风");
    }
  };

  const stopRecording = () => {
    if (mediaRecorderRef.current && isRecording) {
      mediaRecorderRef.current.stop();
      setIsRecording(false);
    }
  };

  const handleSubmitEdit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editPrompt.trim() && !isProcessing && !isTranscribing) {
      onEdit(editPrompt);
      setEditPrompt('');
    }
  };

  const isLoading = isProcessing || isTranscribing;

  const ensureImagePath = async () => {
    if (image.startsWith('/old_pic/')) return image;
    if (image.startsWith('http://') || image.startsWith('https://')) return image;
    if (image.startsWith('data:image')) {
      const timestamp = new Date().toISOString().replace(/[-T:\.Z]/g, '').slice(0, 14);
      const fileName = `img_qr_${timestamp}.png`;
      const response = await fetch('/api/save-image', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: fileName,
          data: image
        })
      });
      if (!response.ok) {
        throw new Error('图片保存失败');
      }
      return `/old_pic/${fileName}`;
    }
    return image;
  };

  const handleGenerateDownloadQr = async () => {
    setQrError('');
    setIsCopied(false);
    try {
      setIsPreparingQr(true);
      const imagePath = await ensureImagePath();
      const configBaseUrl = (siteConfig.siteInfo.baseURL || '').trim();
      const runtimeBaseUrl = typeof window !== 'undefined' ? window.location.origin : '';
      const baseURL = configBaseUrl || runtimeBaseUrl;
      
      // Get the filename directly
      const fileName = decodeURIComponent((imagePath.split('/').pop() || 'download.png').split('?')[0]);
      
      // Generate clean URL with explicit /download route: /download?file=filename.png
      // Normalize base URL to ensure no trailing slash
      const normalizedBaseUrl = baseURL.endsWith('/') ? baseURL.slice(0, -1) : baseURL;
      const finalDownloadPageUrl = `${normalizedBaseUrl}/download?file=${encodeURIComponent(fileName)}`;
      
      const qrBase64 = await generateUrlQrCodeBase64(finalDownloadPageUrl);
      setDownloadPageUrl(finalDownloadPageUrl);
      setQrImageBase64(qrBase64);
      setIsQrModalOpen(true);
    } catch (err: any) {
      setQrError(err?.message || '生成二维码失败');
    } finally {
      setIsPreparingQr(false);
    }
  };

  const handleCopyDownloadPageUrl = async () => {
    if (!downloadPageUrl) return;
    await navigator.clipboard.writeText(downloadPageUrl);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 1500);
  };

  return (
    <div className="fixed inset-0 z-50 bg-black flex items-center justify-center animate-in fade-in duration-500">
      
      {/* Main Image Layer */}
      <div className="absolute inset-0 w-full h-full bg-black flex items-center justify-center">
        <img 
          src={image} 
          alt="Generated Result" 
          className={`w-full h-full object-contain transition-all duration-500 ${isProcessing ? 'opacity-50 blur-sm scale-95' : 'opacity-100 scale-100'}`}
        />
        
        {/* HUD Logo Overlay */}
        <img 
          src={siteConfig.siteInfo.logo || "/ass/111.png"}
          alt="Logo"
          className="absolute top-8 right-8 w-24 lg:w-32 2xl:w-40 object-contain opacity-80 pointer-events-none drop-shadow-lg z-10"
        />
      </div>

      {/* Top Bar */}
      <div className="absolute top-0 left-0 right-0 p-6 lg:p-10 2xl:p-12 flex justify-between items-start bg-gradient-to-b from-black/80 to-transparent z-20 pointer-events-none">
        <button 
          onClick={onClose || (() => window.location.reload())}
          className="pointer-events-auto p-2 lg:p-3 2xl:p-4 rounded-full bg-white/10 backdrop-blur-md text-white hover:bg-white/20 transition-colors"
        >
          <X className="w-6 h-6 lg:w-8 lg:h-8 2xl:w-10 2xl:h-10" />
        </button>
        {/* Download Button Removed */}
      </div>

      {/* Center Loading State */}
      {isProcessing && (
        <div className="absolute inset-0 z-30 flex items-center justify-center pointer-events-none">
           <div className="bg-black/70 backdrop-blur-xl px-8 py-6 lg:px-12 lg:py-8 2xl:px-16 2xl:py-10 rounded-2xl flex flex-col items-center gap-4 lg:gap-6 text-white shadow-2xl border border-white/10">
              <CountdownTimer size={140} duration={120} />
              <p className="font-medium text-lg lg:text-2xl 2xl:text-3xl tracking-wide">AI 正在进行像素级重绘...</p>
           </div>
        </div>
      )}

      {/* Bottom Floating Toolbar */}
      <div className={`absolute bottom-0 left-0 right-0 z-20 transition-transform duration-300 ease-out ${isToolsOpen ? 'translate-y-0' : 'translate-y-[calc(100%-60px)]'}`}>
        
        {/* Toggle Handle */}
        <div className="flex justify-center -mb-4 relative z-30">
          <button 
            onClick={() => setIsToolsOpen(!isToolsOpen)}
            className="px-6 py-2 lg:px-8 lg:py-3 2xl:px-10 2xl:py-4 bg-zinc-900/80 backdrop-blur-md text-zinc-400 hover:text-white rounded-t-xl border-t border-x border-white/10 text-xs lg:text-sm 2xl:text-base font-medium uppercase tracking-wider flex items-center gap-1 shadow-[0_-5px_15px_rgba(0,0,0,0.3)]"
          >
            {isToolsOpen ? <ChevronDown className="w-3 h-3 lg:w-4 lg:h-4" /> : <ChevronUp className="w-3 h-3 lg:w-4 lg:h-4" />}
            {isToolsOpen ? '收起工具栏' : '编辑工具'}
          </button>
        </div>

        {/* Panel Content */}
        <div className="bg-zinc-900/80 backdrop-blur-xl border-t border-white/10 p-6 pb-8 lg:p-8 lg:pb-12 2xl:p-10 2xl:pb-16 shadow-[0_-10px_40px_rgba(0,0,0,0.5)]">
          <div className="w-full max-w-5xl lg:max-w-7xl 2xl:max-w-[90%] mx-auto flex flex-col md:flex-row items-end md:items-center gap-4 lg:gap-6 2xl:gap-8">
            
            {/* Style Buttons */}
            <div className="flex-1 w-full grid grid-cols-2 sm:grid-cols-4 gap-2 lg:gap-4 2xl:gap-6">
               {(siteConfig.resultActions || []).filter((action: any) => action.action === 'style_transfer').map((item: any) => (
                 <button
                    key={item.label}
                    onClick={() => {
                      if (!isLoading) {
                        onEdit(item.value);
                      }
                    }}
                    disabled={isLoading}
                    className="py-3 px-4 lg:py-4 lg:px-6 2xl:py-6 2xl:px-8 bg-transparent border border-white text-white rounded-xl text-sm lg:text-base 2xl:text-xl font-medium transition-all disabled:opacity-50 shadow-none hover:bg-transparent hover:border-white"
                 >
                   {item.label}
                 </button>
               ))}
            </div>

            {/* Upscale Button */}
            {(siteConfig.resultActions || []).filter((action: any) => action.action === 'upscale').map((item: any) => (
            <button
              key={item.label}
              onClick={onUpscale}
              disabled={isLoading}
              className="shrink-0 h-[52px] lg:h-[60px] 2xl:h-[80px] px-6 lg:px-8 2xl:px-12 bg-transparent border border-white text-white rounded-xl font-medium lg:text-lg 2xl:text-2xl flex items-center gap-2 lg:gap-3 transition-all disabled:opacity-50 shadow-none hover:bg-transparent hover:border-white"
            >
              <Zap className="w-4 h-4 lg:w-5 lg:h-5 2xl:w-8 2xl:h-8 text-yellow-400 fill-yellow-400" />
              <span className="hidden sm:inline">{item.label}</span>
            </button>
            ))}

            {/* Download Icon Button in Toolbar */}
            <button
              onClick={handleGenerateDownloadQr}
              disabled={isLoading || isPreparingQr}
              className="shrink-0 w-[52px] h-[52px] lg:w-[60px] lg:h-[60px] 2xl:w-[80px] 2xl:h-[80px] flex items-center justify-center bg-transparent border border-white text-white rounded-xl transition-all disabled:opacity-50 hover:bg-white/10"
              title="生成下载二维码"
            >
              {isPreparingQr ? (
                <div className="w-5 h-5 lg:w-6 lg:h-6 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              ) : (
                <Download className="w-5 h-5 lg:w-6 lg:h-6 2xl:w-8 2xl:h-8" />
              )}
            </button>
          </div>
          {qrError && <p className="mt-4 text-red-400 text-sm text-center">{qrError}</p>}
        </div>
      </div>

      {isQrModalOpen && (
        <div className="absolute inset-0 z-40 bg-black/75 backdrop-blur-sm flex items-center justify-center p-6">
          <div className="w-full max-w-md rounded-2xl border border-white/15 bg-zinc-900/95 p-6 text-white">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold flex items-center gap-2">
                <QrCode className="w-5 h-5 text-emerald-400" />
                下载二维码
              </h3>
              <button
                onClick={() => setIsQrModalOpen(false)}
                className="p-2 rounded-lg hover:bg-white/10"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="rounded-xl bg-white p-4 w-fit mx-auto mb-4">
              <img src={qrImageBase64} alt="下载二维码" className="w-56 h-56 object-contain" />
            </div>

            <p className="text-zinc-300 text-sm leading-relaxed mb-3">
              扫码后会进入下载页面，再点击页面中的下载按钮开始下载图片。
            </p>

            <div className="rounded-xl border border-zinc-700 bg-zinc-950/60 p-3 text-xs text-zinc-300 break-all mb-4">
              {downloadPageUrl}
            </div>

            <div className="flex gap-3">
              <button
                onClick={handleCopyDownloadPageUrl}
                className="flex-1 py-2 rounded-lg border border-zinc-600 hover:border-zinc-400 text-zinc-200 flex items-center justify-center gap-2"
              >
                <Copy className="w-4 h-4" />
                {isCopied ? '已复制' : '复制链接'}
              </button>
              <a
                href={downloadPageUrl}
                target="_blank"
                rel="noreferrer"
                className="flex-1 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-black font-medium flex items-center justify-center gap-2"
              >
                <Download className="w-4 h-4" />
                打开下载页
              </a>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
