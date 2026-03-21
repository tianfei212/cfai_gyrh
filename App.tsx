
import React, { useState, useEffect } from 'react';
import { ImagePlus, Camera, Sparkles, RefreshCw, Download, ChevronRight, AlertCircle, Upload, FolderOpen, X, Trash2, Clock, Key, FileImage, Database } from 'lucide-react';
import { ImageUploader } from './components/ImageUploader';
import { CameraCapture } from './components/CameraCapture';
import { ResultViewer } from './components/ResultViewer';
import { PhotoGallery } from './components/PhotoGallery';
import { HistorySidebar } from './components/HistorySidebar';
import { CountdownTimer } from './components/CountdownTimer';
import { DatabaseManager } from './components/DatabaseManager';
import { generateComposite, editImage, upscaleImage } from './services/geminiService';
import { generateWanImage, editWanImage } from './services/aliWanService';
import { generatePose, PoseServiceType } from './services/poseService';
import { blobToBase64 } from './utils/fileUtils';
import { logToServer } from './utils/logger';
import { addWatermark } from './utils/watermark';
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
    // Check if it's a download request
    const isDownloadRoute = window.location.pathname === '/download';
    const urlParams = new URLSearchParams(window.location.search);
    const downloadFile = urlParams.get('file');
    
    if (isDownloadRoute && downloadFile) {
      const imageUrl = `/old_pic/${downloadFile}`;
      return (
        <div className="min-h-screen flex items-center justify-center bg-zinc-950 text-zinc-200 p-4 font-sans">
          <div className="max-w-md w-full bg-zinc-900 border border-white/10 rounded-2xl p-6 flex flex-col items-center gap-6 shadow-2xl">
            <h1 className="text-2xl font-bold text-white">图片下载</h1>
            <p className="text-sm text-zinc-400 text-center">长按图片保存，或点击下方按钮下载</p>
            <img 
              src={imageUrl} 
              alt="Download Preview" 
              className="w-full rounded-xl border border-white/10 object-contain max-h-[55vh] bg-black" 
            />
            <a 
              href={imageUrl} 
              download={downloadFile}
              className="w-full py-3 rounded-xl bg-green-500 hover:bg-green-400 text-black font-bold text-center transition-colors text-lg"
            >
              直接下载图片
            </a>
          </div>
        </div>
      );
    }

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
    const [isDatabaseOpen, setIsDatabaseOpen] = useState(false);
    
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
      // Now using unified poseService for both Google and AliWan
      // We need to pass 3 images: person (selfie), pose (selfie - using same image for pose reference for now), and background (bg)
      // Assuming 'selfie' contains the person and also serves as the pose reference since user just took a photo
      
      const serviceType: PoseServiceType = apiType === 'wan' ? 'aliWan' : 'google';
      logToServer(`Using ${serviceType} Pose Service`);
      
      // Call unified generatePose interface
      // personImage: selfie (identity)
      // poseImage: selfie (pose reference - user mimics the pose they want)
      // bgImage: bg (scene)
      finalResult = await generatePose(selfie, selfie, bg, serviceType);

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
      let finalResult;
      
      // Determine service based on current API configuration (initially from siteConfig.To_API)
      // If wan, use AliWan model for editing. If google, use Gemini.
      if (apiType === 'wan') {
        logToServer("Using AliWan Service for Edit");
        finalResult = await editWanImage(resultImage, prompt);
      } else {
        logToServer("Using Google Service for Edit");
        finalResult = await editImage(resultImage, prompt);
      }
      
      setResultImage(finalResult);
      saveToOldPic(finalResult);
      
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
      const result = await upscaleImage(resultImage);
      
      // Add Watermark
      logToServer("Adding Watermark to Upscaled Image");
      const watermarkedResult = await addWatermark(
        result, 
        siteConfig.siteInfo.logo || '/ass/111.png', 
        'bottom-right',
        0.9,
        0.2
      );

      logToServer("Process Success: Upscale Image");
      setResultImage(watermarkedResult);
      saveToOldPic(watermarkedResult);
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
            利用 <span className="text-indigo-400 font-semibold">尖端影像技术</span>，实现像素级的人像光影合成。
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
      className="h-screen w-screen flex flex-col bg-zinc-950 text-white selection:bg-indigo-500/30 overflow-hidden bg-cover bg-center bg-no-repeat"
      style={{ 
        backgroundImage: `url('${siteConfig.siteInfo.backgroundImage}')`,
      }}
    >
      <div className="fixed inset-0 bg-black/60 pointer-events-none z-0" />
      
      {/* Invisible Trigger Strip */}
      <div className="fixed top-0 left-0 right-0 h-4 z-[60] peer bg-transparent" />
      
      <header className="fixed top-0 left-0 right-0 z-50 transition-transform duration-300 -translate-y-full peer-hover:translate-y-0 hover:translate-y-0 bg-zinc-950/90 backdrop-blur-md border-b border-white/10 h-16 lg:h-20 shadow-2xl">
        <div className="w-full h-full px-6 lg:px-8 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {siteConfig.header.showLogo && (
              <img src={siteConfig.siteInfo.logo} alt={siteConfig.siteInfo.title} className="h-10 lg:h-12 object-contain" onError={(e) => { console.error('Logo failed to load:', siteConfig.siteInfo.logo); e.currentTarget.style.display = 'none'; }} />
            )}
          </div>
          <div className="flex items-center gap-4">
             {step !== AppStep.UPLOAD_BG && (
              <button onClick={handleReset} className="text-sm lg:text-base text-zinc-400 hover:text-white transition-colors flex items-center gap-2">
                <RefreshCw className="w-4 h-4 lg:w-5 lg:h-5" />
                <span className="hidden sm:inline">重新开始</span>
              </button>
            )}
            
            <button 
              onClick={() => setApiType(prev => prev === 'google' ? 'wan' : 'google')}
              className="w-10 h-10 lg:w-12 lg:h-12 rounded-full bg-zinc-800 border border-zinc-700 hover:border-zinc-500 hover:bg-zinc-700 text-white font-bold transition-all flex items-center justify-center relative group text-sm lg:text-base"
              title={`当前 API: ${apiType === 'google' ? 'ZY20' : 'ZY21'}`}
            >
              {apiType === 'google' ? 'G' : 'W'}
              <span className="absolute -bottom-10 left-1/2 -translate-x-1/2 px-2 py-1 bg-black text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none z-50">
                切换到 {apiType === 'google' ? 'ZY20' : 'ZY21'}
              </span>
            </button>
            
            <button
              onClick={() => setIsDatabaseOpen(true)}
              className="w-10 h-10 lg:w-12 lg:h-12 rounded-full border border-zinc-700 hover:border-zinc-500 hover:bg-zinc-700/50 text-white transition-all flex items-center justify-center relative group backdrop-blur-sm bg-white/5"
              title="图库管理"
            >
              <Database className="w-5 h-5 lg:w-6 lg:h-6 opacity-70 group-hover:opacity-100" />
            </button>
          </div>
        </div>
      </header>
      
      {/* Database Manager Overlay */}
      {isDatabaseOpen && (
        <DatabaseManager 
          onClose={() => {
            setIsDatabaseOpen(false);
            fetchOldPicFiles(); // Refresh sidebar when closing manager
          }} 
        />
      )}

      <main className="flex-1 flex flex-col w-full h-full relative z-10 p-6 lg:p-8 overflow-hidden gap-6 pt-6 lg:pt-8">
        {error && (
          <div className="absolute top-4 left-1/2 -translate-x-1/2 z-50 p-4 rounded-xl bg-red-500/10 border border-red-500/20 flex items-start gap-3 text-red-200 animate-in fade-in slide-in-from-top-2 shadow-2xl backdrop-blur-md">
            <AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
            <p className="text-sm font-medium whitespace-pre-wrap">{error}</p>
          </div>
        )}

        {/* Progress Steps - Compact Version */}
        <div className="shrink-0 flex items-center justify-center gap-4 text-sm lg:text-base text-zinc-500 h-8">
          <div className={`flex items-center gap-2 ${step === AppStep.UPLOAD_BG ? 'text-indigo-400 font-medium' : ''}`}>
            <span className={`w-6 h-6 rounded-full flex items-center justify-center border ${step === AppStep.UPLOAD_BG ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700'}`}>1</span>
            {siteConfig.ui.steps.uploadBg}
          </div>
          <ChevronRight className="w-4 h-4 opacity-30" />
          <div className={`flex items-center gap-2 ${step === AppStep.CAPTURE_SELFIE ? 'text-indigo-400 font-medium' : ''}`}>
            <span className={`w-6 h-6 rounded-full flex items-center justify-center border ${step === AppStep.CAPTURE_SELFIE ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700'}`}>2</span>
            {siteConfig.ui.steps.uploadSelfie}
          </div>
          <ChevronRight className="w-4 h-4 opacity-30" />
          <div className={`flex items-center gap-2 ${step >= AppStep.PROCESSING ? 'text-indigo-400 font-medium' : ''}`}>
            <span className={`w-6 h-6 rounded-full flex items-center justify-center border ${step >= AppStep.PROCESSING ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700'}`}>3</span>
            {siteConfig.ui.steps.result}
          </div>
        </div>

        {/* Content Grid */}
        <div className="flex-1 grid grid-cols-12 gap-6 lg:gap-8 min-h-0">
          
          {/* Left / Main Column */}
          <div className="col-span-8 flex flex-col h-full min-h-0 gap-6">
            
            {step === AppStep.UPLOAD_BG && (
              <>
                {/* Top: Upload Area (45%) */}
                <div className="flex-[0.45] bg-zinc-900/50 border border-white/5 rounded-2xl p-6 lg:p-8 flex flex-col min-h-0">
                   <div className="text-center mb-4 lg:mb-6 shrink-0">
                    <h2 className="text-2xl lg:text-3xl font-bold mb-2">{siteConfig.ui.steps.uploadBg}</h2>
                   
                  </div>
                  <div className="flex-1 min-h-0">
                    <ImageUploader onImageSelected={handleBgUpload} label={siteConfig.ui.uploadBgLabel} />
                  </div>
                </div>
                
                {/* Bottom: Gallery (55%) */}
                <div className="flex-[0.55] bg-zinc-900/50 border border-white/5 rounded-2xl p-6 lg:p-8 flex flex-col min-h-0 overflow-hidden">
                   <PhotoGallery onSelectImage={handleBgUpload} />
                </div>
              </>
            )}

            {step === AppStep.CAPTURE_SELFIE && (
              <div className="flex-1 bg-zinc-900/50 border border-white/5 rounded-2xl p-6 flex flex-col items-center justify-center animate-in zoom-in-95 duration-300">
                {selfieMode === 'upload' && (
                  <div className="text-center mb-8">
                    <h2 className="text-3xl lg:text-4xl font-bold mb-4">{siteConfig.ui.steps.uploadSelfie}</h2>
                    <p className="text-zinc-400 lg:text-lg">请拍摄或上传一张正脸清晰的人物照片。</p>
                  </div>
                )}
                <div className="w-full h-full flex flex-col justify-center max-w-3xl">
                  {selfieMode === 'camera' ? (
                     <CameraCapture 
                        onCapture={handleSelfieCapture} 
                        onClose={() => setSelfieMode('upload')} 
                        bgImage={bgImage} // Pass the selected background for overlay
                     />
                  ) : (
                     <div className="w-full">
                        <ImageUploader onImageSelected={handleSelfieCapture} label={siteConfig.ui.uploadSelfieLabel} subLabel="请确保面部光线均匀" />
                        <div className="mt-8 text-center">
                          <button onClick={() => setSelfieMode('camera')} className="px-8 py-4 rounded-full bg-zinc-800 hover:bg-zinc-700 text-white font-medium text-lg transition-all flex items-center justify-center gap-3 mx-auto border border-white/10 hover:scale-105">
                            <Camera className="w-6 h-6" />
                            切换到相机拍摄
                          </button>
                        </div>
                     </div>
                  )}
                </div>
              </div>
            )}

            {step === AppStep.PROCESSING && (
              <div className="flex-1 flex flex-col items-center justify-center gap-10 bg-zinc-900/30 rounded-2xl border border-white/5">
                <CountdownTimer size={240} duration={120} />
                <div className="text-center">
                  <h3 className="text-2xl lg:text-4xl font-bold text-white mb-4">正在施展 AI 魔法</h3>
                  <p className="text-zinc-400 font-medium tracking-tight animate-pulse text-lg lg:text-xl">{statusMessage || siteConfig.ui.processingMessage}</p>
                </div>
              </div>
            )}

            {step === AppStep.RESULT && resultImage && (
               <div className="flex-1 min-h-0 bg-black rounded-2xl overflow-hidden relative border border-white/10">
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
               </div>
            )}
          </div>

          {/* Right Column: History Sidebar (Always Visible) */}
          <div className="col-span-4 h-full min-h-0">
             <HistorySidebar 
                files={oldPicFiles} 
                onSelect={openFile} 
                resultImage={resultImage}
             />
          </div>
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
