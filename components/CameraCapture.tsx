import React, { useRef, useState, useEffect } from 'react';
import { RefreshCw, Check, ChevronDown, Camera as CameraIcon, X, RefreshCcw, AlertCircle } from 'lucide-react';

interface CameraCaptureProps {
  onCapture: (base64: string) => void;
  onClose?: () => void; // Optional close handler if needed
}

export const CameraCapture: React.FC<CameraCaptureProps> = ({ onCapture, onClose }) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  
  const [stream, setStream] = useState<MediaStream | null>(null);
  const [capturedImage, setCapturedImage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isFlashing, setIsFlashing] = useState(false);
  
  const [availableDevices, setAvailableDevices] = useState<MediaDeviceInfo[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>('');

  const streamRef = useRef<MediaStream | null>(null);

  useEffect(() => {
    streamRef.current = stream;
  }, [stream]);

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop());
      setStream(null);
      streamRef.current = null;
    }
  };

  const startCamera = async (deviceId?: string) => {
    stopCamera();
    
    // Check if mediaDevices is supported
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setError("您的浏览器不支持访问相机，或者因为未使用 HTTPS 而被阻止。请尝试上传照片。");
      return;
    }

    try {
      const constraints: MediaStreamConstraints = { 
        video: { 
          width: { min: 1280, ideal: 3840, max: 7680 },
          height: { min: 720, ideal: 2160, max: 4320 },
          deviceId: deviceId ? { exact: deviceId } : undefined,
          ...(!deviceId && { facingMode: "user" }) 
        } 
      };

      const mediaStream = await navigator.mediaDevices.getUserMedia(constraints);
      setStream(mediaStream);
      
      if (videoRef.current) {
        videoRef.current.srcObject = mediaStream;
      }
      setError(null);
    } catch (err) {
      setError("无法访问相机，请确保已授予权限。");
      console.error(err);
    }
  };

  const refreshDevices = async () => {
    // Check support first
    if (!navigator.mediaDevices || !navigator.mediaDevices.enumerateDevices) {
      console.warn("Media devices enumeration not supported");
      return [];
    }

    try {
      // 1. Ensure we have permission by requesting a temp stream if needed, 
      // but usually we are already running.
      const devices = await navigator.mediaDevices.enumerateDevices();
      const videoDevices = devices.filter(d => d.kind === 'videoinput');
      setAvailableDevices(videoDevices);
      return videoDevices;
    } catch (e) {
      console.error("Failed to enumerate devices", e);
      return [];
    }
  };

  // Initial setup
  useEffect(() => {
    const init = async () => {
      // Early check for support
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
         setError("您的浏览器不支持访问相机，或者因为未使用 HTTPS 而被阻止。");
         return;
      }

      try {
        // Request permission first
        const tempStream = await navigator.mediaDevices.getUserMedia({ video: true });
        
        const videoDevices = await refreshDevices();
        
        // Stop temp stream
        tempStream.getTracks().forEach(track => track.stop());

        // Default logic: Index 2 for Mac if available
        let targetId = '';
        if (videoDevices.length > 0) {
            const indexToUse = videoDevices.length > 2 ? 2 : 0;
            targetId = videoDevices[indexToUse].deviceId;
        }

        setSelectedDeviceId(targetId);
        startCamera(targetId);

      } catch (err) {
        console.error("Error initializing camera:", err);
        setError("无法访问相机，请确保已授予权限。");
      }
    };

    init();
    
    // Listen for device changes (plugging in iPhone, etc.)
    const handleDeviceChange = async () => {
      const videoDevices = await refreshDevices();
      // Optionally switch if the current device is no longer available
    };
    
    if (navigator.mediaDevices && navigator.mediaDevices.addEventListener) {
        navigator.mediaDevices.addEventListener('devicechange', handleDeviceChange);
    }

    return () => {
      stopCamera();
      if (navigator.mediaDevices && navigator.mediaDevices.removeEventListener) {
          navigator.mediaDevices.removeEventListener('devicechange', handleDeviceChange);
      }
    };
  }, []);

  const handleDeviceChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
      const newDeviceId = e.target.value;
      setSelectedDeviceId(newDeviceId);
      startCamera(newDeviceId);
  };

  const handleCapture = () => {
    // Trigger visual flash
    setIsFlashing(true);
    setTimeout(() => setIsFlashing(false), 150);

    if (videoRef.current && canvasRef.current) {
      const video = videoRef.current;
      const canvas = canvasRef.current;
      
      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;
      
      const ctx = canvas.getContext('2d');
      if (ctx) {
        // Correctly handle mirroring
        ctx.translate(canvas.width, 0);
        ctx.scale(-1, 1);
        ctx.drawImage(video, 0, 0);
        
        const base64 = canvas.toDataURL('image/jpeg', 0.95);
        setCapturedImage(base64);
      }
    } else {
      console.error("Capture failed: Missing video or canvas ref");
    }
  };

  const handleRetake = () => {
    setCapturedImage(null);
    if (!streamRef.current) {
        startCamera(selectedDeviceId);
    }
  };

  const handleConfirm = () => {
    if (capturedImage) {
      stopCamera();
      onCapture(capturedImage);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center animate-in fade-in duration-300 p-4 lg:p-8">
      <div className="relative w-[95%] max-w-[1280px] aspect-video h-auto max-h-[85vh] bg-black rounded-3xl overflow-hidden shadow-2xl border border-white/10">
      
      {/* Error Message Overlay */}
      {error && (
        <div className="absolute inset-0 z-[70] bg-black/80 flex flex-col items-center justify-center p-6 text-center">
          <div className="bg-zinc-900 p-6 rounded-2xl border border-red-500/30 max-w-sm">
            <AlertCircle className="w-10 h-10 text-red-500 mx-auto mb-4" />
            <h3 className="text-xl font-bold text-white mb-2">无法访问相机</h3>
            <p className="text-zinc-400 mb-6">{error}</p>
            <div className="flex flex-col gap-3">
              <button 
                onClick={onClose}
                className="px-6 py-3 rounded-full bg-white text-black font-bold hover:bg-zinc-200 transition-colors"
              >
                改为上传照片
              </button>
              <button 
                onClick={() => window.location.reload()}
                className="text-zinc-500 text-sm hover:text-zinc-300"
              >
                刷新页面重试
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Hidden Canvas for Capture Logic - CRITICAL FIX */}
      <canvas ref={canvasRef} className="hidden" />

      {/* Flash Effect Overlay */}
      {isFlashing && (
        <div className="absolute inset-0 z-[60] bg-white animate-out fade-out duration-150 pointer-events-none" />
      )}
      
      {/* Full Screen Video Feed */}
      <div className="absolute inset-0 w-full h-full">
        <video 
          ref={videoRef}
          autoPlay 
          playsInline 
          muted
          className={`w-full h-full object-cover transform -scale-x-100 ${capturedImage ? 'hidden' : 'block'}`}
        />
        {capturedImage && (
          <img 
            src={capturedImage} 
            alt="Capture" 
            className="w-full h-full object-contain bg-black"
          />
        )}
      </div>

      {/* Top Controls Overlay */}
      <div className="absolute top-0 left-0 right-0 p-6 flex justify-between items-start z-10 bg-gradient-to-b from-black/60 to-transparent">
        <button 
          onClick={onClose || (() => window.location.reload())} // Fallback reload if no close provided
          className="p-2 rounded-full bg-black/30 backdrop-blur-md text-white hover:bg-white/20 transition-colors"
        >
          <X className="w-6 h-6" />
        </button>

        {!capturedImage && (
          <div className="flex items-center gap-2">
            <button 
              onClick={() => { refreshDevices(); }}
              className="p-2 rounded-full bg-black/30 backdrop-blur-md text-white hover:bg-white/20 transition-colors"
              title="刷新设备列表"
            >
              <RefreshCcw className="w-5 h-5" />
            </button>
            {availableDevices.length > 0 && (
              <div className="relative group/select">
                 <select
                    value={selectedDeviceId}
                    onChange={handleDeviceChange}
                    className="appearance-none bg-black/30 backdrop-blur-md text-white pl-10 pr-10 py-2 rounded-full border border-white/20 hover:bg-black/50 hover:border-white/40 focus:outline-none focus:border-indigo-500 text-sm cursor-pointer transition-all"
                  >
                    {availableDevices.map((device, index) => (
                      <option key={device.deviceId || index} value={device.deviceId} className="bg-zinc-900 text-white">
                        {device.label || `摄像头 ${index + 1}`}
                      </option>
                    ))}
                  </select>
                  <CameraIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/90 pointer-events-none" />
                  <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/70 pointer-events-none" />
              </div>
            )}
          </div>
        )}
      </div>

      {/* Bottom Controls Overlay */}
      <div className="absolute bottom-0 left-0 right-0 p-8 flex justify-center items-center z-10 bg-gradient-to-t from-black/80 to-transparent">
        {!capturedImage ? (
          <button
            onClick={handleCapture}
            className="group relative flex items-center justify-center w-16 h-16 rounded-full border-4 border-white hover:border-indigo-400 hover:scale-105 transition-all shadow-lg shadow-black/50"
          >
            <div className="w-12 h-12 bg-white rounded-full group-hover:bg-indigo-400 transition-colors" />
          </button>
        ) : (
          <div className="flex gap-4 animate-in slide-in-from-bottom-4 fade-in">
            <button
              onClick={handleRetake}
              className="flex items-center gap-2 px-5 py-2.5 rounded-full bg-white/10 backdrop-blur-md hover:bg-white/20 text-white font-medium transition-colors border border-white/10"
            >
              <RefreshCw className="w-4 h-4" />
              重拍
            </button>
            <button
              onClick={handleConfirm}
              className="flex items-center gap-2 px-6 py-2.5 rounded-full bg-indigo-600 hover:bg-indigo-500 text-white font-bold shadow-xl shadow-indigo-500/25 transition-all transform hover:scale-105"
            >
              <Check className="w-4 h-4" />
              使用照片
            </button>
          </div>
        )}
      </div>
    </div>
  </div>
  );
};
