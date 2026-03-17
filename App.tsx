
import React, { useState, useEffect } from 'react';
import { ImagePlus, Camera, Sparkles, RefreshCw, Download, ChevronRight, AlertCircle, Upload, FolderOpen, X, Trash2, Clock, Key, FileImage } from 'lucide-react';
import { ImageUploader } from './components/ImageUploader';
import { CameraCapture } from './components/CameraCapture';
import { ResultViewer } from './components/ResultViewer';
import { PhotoGallery } from './components/PhotoGallery';
import { generateComposite, editImage, upscaleImage } from './services/geminiService';
import { generateWanImage } from './services/aliWanService';
import { blobToBase64 } from './utils/fileUtils';
import { logToServer } from './utils/logger';
import siteConfig from './siteConfig.json';

// Application Steps Enum
enum AppStep {
  UPLOAD_BG = 0,
  CAPTURE_SELFIE = 1,
  PROCESSING = 2,
  RESULT = 3,
}

// Local interface for the window object to avoid global conflicts
interface AIStudioWindow {
  aistudio?: {
    hasSelectedApiKey: () => Promise<boolean>;
    openSelectKey: () => Promise<void>;
  };
}

  interface FileItem {
    name: string;
    data: string; // This will now store URL for remote files, or Base64 for local preview if needed
    timestamp: number;
    isRemote?: boolean;
  }
  
// Simple Error Boundary
class ErrorBoundary extends React.Component<React.PropsWithChildren<{}>, {hasError: boolean, error: any}> {
  state: {hasError: boolean, error: any};
  props!: Readonly<React.PropsWithChildren<{}>>;

  constructor(props: React.PropsWithChildren<{}>) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: any) {
    return { hasError: true, error };
  }

  componentDidCatch(error: any, errorInfo: any) {
    console.error("Uncaught error:", error, errorInfo);
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen flex items-center justify-center bg-zinc-950 text-red-500 p-8">
          <div className="max-w-xl border border-red-500/20 bg-red-500/10 p-6 rounded-xl">
            <h1 className="text-xl font-bold mb-4">应用发生错误</h1>
            <pre className="whitespace-pre-wrap font-mono text-sm">{this.state.error?.toString()}</pre>
            <button 
              onClick={() => window.location.reload()}
              className="mt-6 px-4 py-2 bg-zinc-800 text-white rounded hover:bg-zinc-700"
            >
              刷新页面
            </button>
          </div>
        </div>
      );
    }

    return this.props.children || null;
  }
}

  const App: React.FC = () => {
    return (
      <ErrorBoundary>
        <AppContent />
      </ErrorBoundary>
    );
  };

  const AppContent: React.FC = () => {
    // Auth State
    const [hasApiKey, setHasApiKey] = useState(false);
    const [isCheckingKey, setIsCheckingKey] = useState(true);

    // App State
    const [step, setStep] = useState<AppStep>(AppStep.UPLOAD_BG);
    const [bgImage, setBgImage] = useState<string | null>(null);
    const [selfieImage, setSelfieImage] = useState<string | null>(null);
    const [resultImage, setResultImage] = useState<string | null>(null);
    const [isProcessing, setIsProcessing] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [statusMessage, setStatusMessage] = useState<string>("");
    // Default to 'camera' mode as requested
    const [selfieMode, setSelfieMode] = useState<'camera' | 'upload'>('camera');
    // API Switch state
    const [apiType, setApiType] = useState<'google' | 'wan'>(
      // @ts-ignore
      siteConfig.To_API === 'wan' ? 'wan' : 'google'
    );
    
    // File System State (old_pic directory)
    const [oldPicFiles, setOldPicFiles] = useState<FileItem[]>([]);
    const [isFileSystemOpen, setIsFileSystemOpen] = useState(false);
    const isCameraOpen = step === AppStep.CAPTURE_SELFIE && selfieMode === 'camera';
  
    // Check for API Key on mount
    useEffect(() => {
      checkApiKey();
    }, []);

    const checkApiKey = async () => {
      try {
        // Check for environment variable first (for local development)
        if (process.env.API_KEY || process.env.GEMINI_API_KEY) {
          setHasApiKey(true);
          setIsCheckingKey(false);
          return;
        }

        const win = window as unknown as AIStudioWindow;
        if (win.aistudio && await win.aistudio.hasSelectedApiKey()) {
          setHasApiKey(true);
        } else {
          setHasApiKey(false);
        }
      } catch (e) {
        console.error("Error checking API key:", e);
      } finally {
        setIsCheckingKey(false);
      }
    };

    const handleSelectKey = async () => {
      try {
        const win = window as unknown as AIStudioWindow;
        if (win.aistudio) {
          await win.aistudio.openSelectKey();
          await checkApiKey();
          setHasApiKey(true);
        }
      } catch (e) {
        console.error("Failed to select key:", e);
        setError("API Key 选择失败，请重试。");
      }
    };
  
    // Load old_pic files on mount - fetch from API
    useEffect(() => {
      fetchOldPicFiles();
    }, []);

    const fetchOldPicFiles = async () => {
      try {
        const response = await fetch('/api/list-images');
        if (response.ok) {
          const files = await response.json();
          const mappedFiles: FileItem[] = files.map((f: any) => ({
            name: f.name,
            data: f.url, // Store URL
            timestamp: f.timestamp,
            isRemote: true
          }));
          setOldPicFiles(mappedFiles);
        }
      } catch (e) {
        console.error("Failed to load old_pic files", e);
      }
    };
  
    // Save generated image to "old_pic" (Server API)
    const saveToOldPic = async (base64Data: string) => {
      const now = new Date();
      const timestampStr = now.toISOString().replace(/[-T:\.Z]/g, '').slice(0, 14);
      const fileName = `img_${timestampStr}.png`;
  
      const newFile: FileItem = {
        name: fileName,
        data: base64Data, // Temporarily show base64 until refreshed
        timestamp: now.getTime(),
        isRemote: false
      };
  
      // Optimistic update
      setOldPicFiles((prev) => [newFile, ...prev]);

      try {
        await fetch('/api/save-image', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            name: fileName,
            data: base64Data
          }),
        });
        
        // Refresh list to get URLs
        fetchOldPicFiles();
      } catch (e) {
        console.error("Failed to save image to server", e);
      }
    };
  
    const clearOldPic = () => {
      // Not implemented on server for safety, just clear local view
      setOldPicFiles([]);
    };
  
    const openFile = async (file: FileItem) => {
      if (file.isRemote) {
        // Fetch the image and convert to base64 for the editor
        try {
            const response = await fetch(file.data);
            const blob = await response.blob();
            const reader = new FileReader();
            reader.onloadend = () => {
                setResultImage(reader.result as string);
                setStep(AppStep.RESULT);
                setIsFileSystemOpen(false);
                setError(null);
            };
            reader.readAsDataURL(blob);
        } catch (e) {
            console.error("Failed to load remote image", e);
            setError("无法加载历史图片");
        }
      } else {
        setResultImage(file.data);
        setStep(AppStep.RESULT);
        setIsFileSystemOpen(false);
        setError(null);
      }
    };

  const handleBgUpload = (base64: string) => {
    logToServer("User Action: Upload Background", { imageSize: base64.length });
    setBgImage(base64);
    setStep(AppStep.CAPTURE_SELFIE);
    setError(null);
  };

  const handleSelfieCapture = async (base64: string) => {
    logToServer("User Action: Capture/Upload Selfie", { imageSize: base64.length });
    setSelfieImage(base64);
    setStep(AppStep.PROCESSING);
    await processComposition(bgImage!, base64);
  };

  // Simplified Orchestration Logic: Pure Gemini 3 Pro
  const processComposition = async (bg: string, selfie: string) => {
    logToServer("Process Started: Composition");
    setIsProcessing(true);
    setStatusMessage("正在重建光影与环境...");
    setError(null);

    try {
      let finalResult;
      
      // API Switch Logic
      if (apiType === 'wan') {
         logToServer("Using AliWan Service");
         // AliWan: Person (Selfie) is 1st image, Background is 2nd image
         finalResult = await generateWanImage(selfie, bg);
      } else {
         logToServer("Using Google Gemini Service");
         // Gemini: Standard composition
         finalResult = await generateComposite(bg, selfie);
      }

      logToServer("Process Success: Composition");
      setResultImage(finalResult);
      saveToOldPic(finalResult);
      setStep(AppStep.RESULT);
    } catch (err: any) {
      logToServer("Process Failed: Composition", { error: err.message }, "ERROR");
      console.error(err);
      setError(err.message || "合成处理失败，请重试。");
      setStep(AppStep.CAPTURE_SELFIE);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleEdit = async (prompt: string) => {
    if (!resultImage) return;

    logToServer("User Action: Edit Image", { prompt });
    setIsProcessing(true);
    setStatusMessage("AI 正在解析指令并重绘...");
    setError(null);

    try {
      if (apiType === 'wan') {
        // AliWan supports same edit interface conceptually
        // For now, we reuse Gemini for editing or implement Wan edit if available
        // Assuming we fall back to Gemini for editing as Wan service was mainly for composition
        // OR if you want to use Wan for editing:
        // finalResult = await generateWanImage(resultImage, resultImage, prompt); // This might need adjustment
        
        // Current implementation: Fallback to Gemini for edit to keep it simple unless specified
        const finalResult = await editImage(resultImage, prompt);
        setResultImage(finalResult);
        saveToOldPic(finalResult);
      } else {
        const finalResult = await editImage(resultImage, prompt);
        setResultImage(finalResult);
        saveToOldPic(finalResult);
      }
      
      logToServer("Process Success: Edit Image");
    } catch (err: any) {
      logToServer("Process Failed: Edit Image", { error: err.message }, "ERROR");
      setError("编辑失败，请尝试其他提示词。");
    } finally {
      setIsProcessing(false);
    }
  };

  const handleUpscale = async () => {
    if (!resultImage) return;

    logToServer("User Action: Upscale Image");
    setIsProcessing(true);
    setStatusMessage("正在生成 2K 超清影像...");
    setError(null);

    try {
      const finalResult = await upscaleImage(resultImage);
      logToServer("Process Success: Upscale Image");
      setResultImage(finalResult);
      saveToOldPic(finalResult);
    } catch (err: any) {
      logToServer("Process Failed: Upscale Image", { error: err.message }, "ERROR");
      setError("超分失败，请稍后再试。");
    } finally {
      setIsProcessing(false);
    }
  };

  const handleReset = () => {
    setBgImage(null);
    setSelfieImage(null);
    setResultImage(null);
    setStep(AppStep.UPLOAD_BG);
    setSelfieMode('camera');
    setError(null);
  };

  if (isCheckingKey) {
    return <div className="min-h-screen bg-zinc-950 flex items-center justify-center text-zinc-500">正在初始化...</div>;
  }

  if (!hasApiKey) {
    return (
      <div 
        className="min-h-screen bg-zinc-950 flex flex-col items-center justify-center px-6 relative overflow-hidden bg-cover bg-center bg-no-repeat"
        style={{ backgroundImage: `url(${siteConfig.siteInfo.backgroundImage})` }}
      >
        <div className="absolute inset-0 bg-black/70 pointer-events-none" />
        <div className="absolute top-0 left-0 w-96 h-96 bg-indigo-600/20 rounded-full blur-[128px] pointer-events-none" />
        <div className="absolute bottom-0 right-0 w-96 h-96 bg-purple-600/20 rounded-full blur-[128px] pointer-events-none" />
        <div className="relative z-10 max-w-md w-full bg-zinc-900/50 backdrop-blur-xl border border-zinc-800 p-8 rounded-2xl shadow-2xl text-center">
          <div className="w-16 h-16 bg-gradient-to-tr from-indigo-500 to-purple-500 rounded-2xl mx-auto flex items-center justify-center mb-6 shadow-lg shadow-indigo-500/25">
             {siteConfig.header.showLogo ? (
               <img src={siteConfig.siteInfo.logo} alt="Logo" className="w-10 h-10 object-contain brightness-0 invert" onError={(e) => { e.currentTarget.style.display = 'none'; }} />
             ) : (
               <Sparkles className="w-8 h-8 text-white" />
             )}
          </div>
          <h1 className="text-2xl font-bold text-white mb-3">光影重建 AI</h1>
          <p className="text-zinc-400 mb-8 leading-relaxed">
            利用 <span className="text-indigo-400 font-semibold">Gemini 3 Pro</span> 尖端影像技术，实现像素级的人像光影合成。
          </p>
          <button 
            onClick={handleSelectKey}
            className="w-full py-4 bg-white hover:bg-zinc-100 text-black font-bold rounded-xl flex items-center justify-center gap-3 transition-all hover:scale-[1.02] shadow-xl"
          >
            <Key className="w-5 h-5" />
            连接 API 以开始
          </button>
        </div>
      </div>
    );
  }

  return (
    <div 
      className="min-h-screen flex flex-col bg-zinc-950 text-white selection:bg-indigo-500/30 overflow-x-hidden bg-cover bg-center bg-no-repeat bg-fixed"
      style={{ 
        backgroundImage: `url('${siteConfig.siteInfo.backgroundImage}')`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        backgroundRepeat: 'no-repeat'
      }}
    >
      <div className="fixed inset-0 bg-black/60 pointer-events-none z-0" />
      <header className="border-b border-white/10 backdrop-blur-md sticky top-0 z-40 bg-zinc-950/80">
        <div className="max-w-5xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {siteConfig.header.showLogo && (
              <img src={siteConfig.siteInfo.logo} alt={siteConfig.siteInfo.title} className="h-10 object-contain" onError={(e) => { console.error('Logo failed to load:', siteConfig.siteInfo.logo); e.currentTarget.style.display = 'none'; }} />
            )}
          </div>
          <div className="flex items-center gap-4">
             {step !== AppStep.UPLOAD_BG && (
              <button onClick={handleReset} className="text-sm text-zinc-400 hover:text-white transition-colors flex items-center gap-2">
                <RefreshCw className="w-4 h-4" />
                <span className="hidden sm:inline">重新开始</span>
              </button>
            )}
            
            <button 
              onClick={() => setApiType(prev => prev === 'google' ? 'wan' : 'google')}
              className="w-10 h-10 rounded-full bg-zinc-800 border border-zinc-700 hover:border-zinc-500 hover:bg-zinc-700 text-white font-bold transition-all flex items-center justify-center relative group"
              title={`当前 API: ${apiType === 'google' ? 'Google Gemini' : 'AliWan 2.6'}`}
            >
              {apiType === 'google' ? 'G' : 'W'}
              <span className="absolute -bottom-8 left-1/2 -translate-x-1/2 px-2 py-1 bg-black text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none z-50">
                切换到 {apiType === 'google' ? 'AliWan' : 'Google'}
              </span>
            </button>

            <button onClick={() => setIsFileSystemOpen(true)} className="p-2 text-zinc-400 hover:text-white hover:bg-white/10 rounded-full transition-colors relative flex items-center gap-2" title="资源管理器">
              <FolderOpen className="w-5 h-5" />
              {oldPicFiles.length > 0 && <span className="absolute top-1 right-1 w-2 h-2 bg-indigo-500 rounded-full ring-2 ring-zinc-900"></span>}
            </button>
          </div>
        </div>
      </header>

      {/* old_pic File System Sidebar */}
      {isFileSystemOpen && (
        <>
          <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 transition-opacity" onClick={() => setIsFileSystemOpen(false)} />
          <div className="fixed right-0 top-0 h-full w-80 bg-zinc-900 border-l border-white/10 z-50 shadow-2xl transform transition-transform duration-300 ease-out flex flex-col font-mono text-sm">
            <div className="p-4 border-b border-white/10 flex items-center justify-between bg-zinc-800/50">
              <h2 className="font-semibold flex items-center gap-2 text-zinc-300">
                <FolderOpen className="w-4 h-4 text-yellow-500" />
                old_pic/
              </h2>
              <div className="flex items-center gap-2">
                {oldPicFiles.length > 0 && (
                  <button onClick={clearOldPic} className="p-2 text-zinc-500 hover:text-red-400 hover:bg-red-500/10 rounded-md transition-colors" title="清空">
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
                <button onClick={() => setIsFileSystemOpen(false)} className="p-2 text-zinc-400 hover:text-white hover:bg-white/10 rounded-md transition-colors">
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>
            <div className="flex-1 overflow-y-auto p-2 space-y-1 bg-zinc-950">
              {oldPicFiles.length === 0 ? (
                <div className="text-center text-zinc-600 py-10 flex flex-col items-center gap-3">
                  <FolderOpen className="w-10 h-10 opacity-20" />
                  <p className="text-xs">文件夹为空</p>
                </div>
              ) : (
                oldPicFiles.map((file, index) => (
                  <div 
                    key={index}
                    onClick={() => openFile(file)}
                    className={`group flex items-center gap-3 p-2 rounded cursor-pointer border border-transparent ${resultImage === file.data ? 'bg-indigo-900/30 border-indigo-500/30' : 'hover:bg-zinc-800 hover:border-zinc-700'}`}
                  >
                    <div className="w-10 h-10 shrink-0 bg-zinc-800 rounded overflow-hidden border border-zinc-700">
                       <img src={file.data} alt="" className="w-full h-full object-cover" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-zinc-300 truncate font-mono text-xs">{file.name}</p>
                      <p className="text-zinc-600 text-[10px]">{new Date(file.timestamp).toLocaleTimeString()}</p>
                    </div>
                  </div>
                ))
              )}
            </div>
            <div className="p-2 border-t border-white/10 bg-zinc-900 text-[10px] text-zinc-500 flex justify-between px-4">
              <span>{oldPicFiles.length} 对象</span>
              <span>本地存储</span>
            </div>
          </div>
        </>
      )}

      <main className="flex-1 flex flex-col w-full max-w-[95%] xl:max-w-[90%] 2xl:max-w-[85%] mx-auto px-4 sm:px-6 py-8 relative z-10 pt-20">
        {error && (
          <div className="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/20 flex items-start gap-3 text-red-200 animate-in fade-in slide-in-from-top-2">
            <AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
            <p className="text-sm font-medium whitespace-pre-wrap">{error}</p>
          </div>
        )}

        <div className="mb-8 lg:mb-12 flex items-center justify-center gap-4 text-sm lg:text-base 2xl:text-xl text-zinc-500">
          <div className={`flex items-center gap-2 ${step === AppStep.UPLOAD_BG ? 'text-indigo-400 font-medium' : ''}`}>
            <span className={`w-6 h-6 lg:w-8 lg:h-8 2xl:w-10 2xl:h-10 rounded-full flex items-center justify-center border ${step === AppStep.UPLOAD_BG ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700'}`}>1</span>
            {siteConfig.ui.steps.uploadBg}
          </div>
          <ChevronRight className="w-4 h-4 lg:w-5 lg:h-5 opacity-30" />
          <div className={`flex items-center gap-2 ${step === AppStep.CAPTURE_SELFIE ? 'text-indigo-400 font-medium' : ''}`}>
            <span className={`w-6 h-6 lg:w-8 lg:h-8 2xl:w-10 2xl:h-10 rounded-full flex items-center justify-center border ${step === AppStep.CAPTURE_SELFIE ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700'}`}>2</span>
            {siteConfig.ui.steps.uploadSelfie}
          </div>
          <ChevronRight className="w-4 h-4 lg:w-5 lg:h-5 opacity-30" />
          <div className={`flex items-center gap-2 ${step >= AppStep.PROCESSING ? 'text-indigo-400 font-medium' : ''}`}>
            <span className={`w-6 h-6 lg:w-8 lg:h-8 2xl:w-10 2xl:h-10 rounded-full flex items-center justify-center border ${step >= AppStep.PROCESSING ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700'}`}>3</span>
            {siteConfig.ui.steps.result}
          </div>
        </div>

        <div className="flex-1 flex flex-col items-center justify-center min-h-[400px] lg:min-h-[600px] 2xl:min-h-[800px]">
          {step === AppStep.UPLOAD_BG && (
            <div className="w-full max-w-xl lg:max-w-2xl 2xl:max-w-4xl animate-in zoom-in-95 duration-300">
              <div className="text-center mb-8 lg:mb-12">
                <h2 className="text-3xl lg:text-4xl 2xl:text-5xl font-bold mb-2 lg:mb-4">{siteConfig.ui.steps.uploadBg}</h2>
                <p className="text-zinc-400 lg:text-lg 2xl:text-xl">上传或拖拽一张背景图片。</p>
              </div>
              <ImageUploader onImageSelected={handleBgUpload} label={siteConfig.ui.uploadBgLabel} />
              <PhotoGallery onSelectImage={handleBgUpload} />
            </div>
          )}

          {step === AppStep.CAPTURE_SELFIE && (
            <div className="w-full animate-in zoom-in-95 duration-300 flex flex-col items-center">
              {selfieMode === 'upload' && (
                <div className="text-center mb-6 lg:mb-10">
                  <h2 className="text-3xl lg:text-4xl 2xl:text-5xl font-bold mb-2 lg:mb-4">{siteConfig.ui.steps.uploadSelfie}</h2>
                  <p className="text-zinc-400 lg:text-lg 2xl:text-xl">请拍摄或上传一张正脸清晰的人物照片。</p>
                </div>
              )}
              <div className="w-full flex justify-center">
                {selfieMode === 'camera' ? (
                   <CameraCapture onCapture={handleSelfieCapture} onClose={() => setSelfieMode('upload')} />
                ) : (
                   <div className="w-full max-w-xl lg:max-w-2xl 2xl:max-w-4xl">
                      <ImageUploader onImageSelected={handleSelfieCapture} label={siteConfig.ui.uploadSelfieLabel} subLabel="请确保面部光线均匀" />
                      <div className="mt-8 lg:mt-12 text-center">
                        <button onClick={() => setSelfieMode('camera')} className="px-6 py-3 lg:px-8 lg:py-4 2xl:px-10 2xl:py-5 rounded-full bg-zinc-800 hover:bg-zinc-700 text-white font-medium lg:text-lg 2xl:text-xl transition-all flex items-center justify-center gap-2 mx-auto border border-white/10">
                          <Camera className="w-4 h-4 lg:w-5 lg:h-5 2xl:w-6 2xl:h-6" />
                          切换到相机拍摄
                        </button>
                      </div>
                   </div>
                )}
              </div>
            </div>
          )}

          {step === AppStep.PROCESSING && (
            <div className="flex flex-col items-center justify-center gap-6 lg:gap-10 animate-in fade-in duration-500">
              <div className="relative">
                <div className="w-24 h-24 lg:w-32 lg:h-32 2xl:w-48 2xl:h-48 rounded-full border-4 lg:border-8 border-indigo-500/30 animate-[spin_3s_linear_infinite]"></div>
                <div className="w-24 h-24 lg:w-32 lg:h-32 2xl:w-48 2xl:h-48 rounded-full border-t-4 lg:border-t-8 border-indigo-500 absolute top-0 left-0 animate-spin"></div>
                <Sparkles className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 text-indigo-400 w-8 h-8 lg:w-12 lg:h-12 2xl:w-16 2xl:h-16 animate-pulse" />
              </div>
              <div className="text-center">
                <h3 className="text-xl lg:text-2xl 2xl:text-3xl font-semibold text-white mb-1 lg:mb-2">正在施展 AI 魔法</h3>
                <p className="text-zinc-400 font-medium tracking-tight animate-pulse lg:text-lg 2xl:text-xl">{statusMessage || siteConfig.ui.processingMessage}</p>
              </div>
            </div>
          )}

          {step === AppStep.RESULT && resultImage && (
             <ResultViewer 
                image={resultImage} 
                isProcessing={isProcessing}
                onEdit={handleEdit}
                onUpscale={handleUpscale}
                onDownload={() => {
                  const link = document.createElement('a');
                  link.href = resultImage;
                  const now = new Date();
                  const timestampStr = now.toISOString().replace(/[-T:\.Z]/g, '').slice(0, 14);
                  link.download = `composite_${timestampStr}.png`;
                  link.click();
                }}
                onClose={() => setStep(AppStep.UPLOAD_BG)}
              />
          )}
        </div>
      </main>
      
      {!isCameraOpen && (
        <footer className="py-6 text-center text-zinc-600 text-sm border-t border-white/5 relative z-0 pointer-events-none">
          <p>{siteConfig.footer.copyright} | {siteConfig.footer.poweredBy}</p>
        </footer>
      )}
    </div>
  );
  };

export default App;
