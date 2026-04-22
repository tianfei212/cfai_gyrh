import React, { useEffect, useRef, useState } from 'react';
import { SelfieSegmentation } from '@mediapipe/selfie_segmentation';
import { SimpleFrame } from '../components/Layout';
import { CameraIcon, ClockIcon } from '../components/Icons';
import { fetchApi } from '../services/api';

export function CaptureScreen({ onHome, onHistory, onLogout, onToggleModel, model, selectedBg, onPreview }) {
  const videoRef = useRef(null);
  const canvasRef = useRef(null);
  const [opacity, setOpacity] = useState(0.8);
  const [isCapturing, setIsCapturing] = useState(false);
  const selfieSegmentationRef = useRef(null);

  useEffect(() => {
    let stream = null;

    async function startCamera() {
      try {
        stream = await navigator.mediaDevices.getUserMedia({ 
          video: { 
            width: { ideal: 1280 },
            height: { ideal: 720 },
            aspectRatio: 1.7777777778 
          } 
        });
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
        }
      } catch (err) {
        console.error("Error accessing camera:", err);
      }
    }

    // Initialize MediaPipe but don't run it yet
    const selfieSegmentation = new SelfieSegmentation({
      locateFile: (file) => `/models/selfie_segmentation/${file}`
    });
    selfieSegmentation.setOptions({
      modelSelection: 1,
      selfieMode: true,
    });
    selfieSegmentationRef.current = selfieSegmentation;

    startCamera();

    return () => {
      if (stream) {
        stream.getTracks().forEach(track => track.stop());
      }
      if (selfieSegmentationRef.current) {
        selfieSegmentationRef.current.close();
      }
    };
  }, []);

  const renderBackground = () => {
    if (!selectedBg) return <div style={{ position: 'absolute', inset: 0, background: '#000', zIndex: 1 }} />;
    if (typeof selectedBg === 'string') {
      return (
        <img 
          src={selectedBg} 
          alt="Selected Background" 
          style={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            zIndex: 1
          }}
        />
      );
    }
    if (typeof selectedBg === 'object' && selectedBg.image_url) {
      return (
        <img 
          src={selectedBg.image_url} 
          alt="Selected Background" 
          style={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            zIndex: 1
          }}
        />
      );
    }
    return (
      <div 
        className={`tone-${selectedBg.tone}`}
        style={{
          position: 'absolute',
          inset: 0,
          width: '100%',
          height: '100%',
          zIndex: 1
        }}
      />
    );
  };

  const handleCapture = async () => {
    if (!videoRef.current || !selfieSegmentationRef.current) return;
    
    setIsCapturing(true);
    const videoElement = videoRef.current;
    
    selfieSegmentationRef.current.onResults(async (results) => {
      const canvas = document.createElement('canvas');
      canvas.width = results.image.width;
      canvas.height = results.image.height;
      const ctx = canvas.getContext('2d');

      // Extract Person
      const tempCanvas = document.createElement('canvas');
      tempCanvas.width = canvas.width;
      tempCanvas.height = canvas.height;
      const tempCtx = tempCanvas.getContext('2d');
      tempCtx.drawImage(results.segmentationMask, 0, 0, canvas.width, canvas.height);
      tempCtx.globalCompositeOperation = 'source-in';
      tempCtx.drawImage(results.image, 0, 0, canvas.width, canvas.height);

      const foregroundDataUrl = tempCanvas.toDataURL('image/png');
      const foregroundBase64 = foregroundDataUrl.split(',')[1];

      let backgroundBase64 = null;
      let backgroundPromptId = 0;

      if (selectedBg) {
        try {
          if (typeof selectedBg === 'object' && selectedBg.image_url) {
            backgroundPromptId = selectedBg.id;
            const res = await fetch(selectedBg.image_url);
            const blob = await res.blob();
            backgroundBase64 = await new Promise((resolve) => {
              const reader = new FileReader();
              reader.onloadend = () => resolve(reader.result.split(',')[1]);
              reader.readAsDataURL(blob);
            });
          } else if (typeof selectedBg === 'string' && (selectedBg.startsWith('blob:') || selectedBg.startsWith('http'))) {
            // Local loaded image (blob) or remote URL
            backgroundPromptId = 1; // Default fallback for local image
            const res = await fetch(selectedBg);
            const blob = await res.blob();
            backgroundBase64 = await new Promise((resolve) => {
              const reader = new FileReader();
              reader.onloadend = () => resolve(reader.result.split(',')[1]);
              reader.readAsDataURL(blob);
            });
          }
        } catch (err) {
          console.error('Failed to process background image:', err);
        }
      }

      try {
        const payload = {
          foreground: foregroundBase64,
          provider: model === 'W' ? 'wan' : 'google'
        };

        if (backgroundBase64) {
          payload.background = backgroundBase64;
          payload.background_prompt_id = backgroundPromptId;
        }

        const data = await fetchApi('/api/v1/images/rewrite', {
          method: 'POST',
          body: JSON.stringify(payload)
        });

        if (data && data.image_url) {
          onPreview(data.image_url);
        } else {
          onPreview(foregroundDataUrl);
        }
      } catch (err) {
        console.error('Failed to rewrite image:', err);
        alert('生成失败: ' + err.message);
        onPreview(foregroundDataUrl);
      } finally {
        setIsCapturing(false);
      }
    });

    await selfieSegmentationRef.current.send({ image: videoElement });
  };

  return (
    <>
      <SimpleFrame 
        title="AI Smart Portrait · 摄像头拍摄"
        onHome={onHome}
        onHistory={onHistory}
        onLogout={onLogout}
        onToggleModel={onToggleModel}
        model={model}
      >
        <section className="glass-section capture-shell">
          <div className="capture-stage" style={{ position: 'relative', overflow: 'hidden', background: '#000' }}>
            {renderBackground()}
            
            <video 
              ref={videoRef}
              autoPlay
              playsInline
              muted
              style={{
                position: 'relative',
                zIndex: 2,
                width: '100%',
                height: '100%',
                objectFit: 'cover',
                opacity: opacity
              }}
            />

            <div className="opacity-slider-wrapper" style={{ zIndex: 4 }}>
              <span className="slider-label">透明度</span>
              <div className="slider-track">
                <input 
                  type="range" 
                  min="0" 
                  max="1" 
                  step="0.01" 
                  value={opacity} 
                  onChange={(e) => setOpacity(parseFloat(e.target.value))}
                  className="vertical-slider"
                />
              </div>
              <span className="slider-value">{Math.round(opacity * 100)}%</span>
            </div>

            <div className="camera-badge" style={{ zIndex: 3 }}>LIVE PREVIEW</div>
            <div className="capture-title" style={{ zIndex: 3 }}>
              <strong>{typeof selectedBg === 'object' ? selectedBg.name : '背景预览'}</strong>
              <span>点击拍摄后，系统将为您生成证件照</span>
            </div>
          </div>
          <div className="capture-actions">
            <button className="soft-button" type="button" onClick={onHome}>X 放弃</button>
            <button className="soft-button primary" type="button" onClick={handleCapture} disabled={isCapturing}>
              <CameraIcon />
              {isCapturing ? '生成中...' : '拍摄'}
            </button>
            <button className="soft-button" type="button">
              <ClockIcon />
              使用
            </button>
          </div>
        </section>
      </SimpleFrame>

      {/* Full Screen "Generating" Overlay */}
      {isCapturing && (
        <div className="generating-overlay" style={{
          position: 'fixed',
          inset: 0,
          zIndex: 9999,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'rgba(10, 15, 28, 0.9)',
          backdropFilter: 'blur(20px)',
          color: 'white'
        }}>
          <div className="generating-content" style={{ textAlign: 'center' }}>
            <div className="loading-spinner-large" style={{ 
              width: '80px', 
              height: '80px', 
              border: '4px solid rgba(255,255,255,0.1)',
              borderTop: '4px solid #0055ff',
              borderRadius: '50%',
              animation: 'spin 1s linear infinite',
              margin: '0 auto 2rem'
            }} />
            <h2 style={{ fontSize: '2rem', marginBottom: '1rem', fontWeight: '600' }}>生成中</h2>
            <p style={{ color: 'rgba(255,255,255,0.6)', fontSize: '1.1rem' }}>
              正在利用 AI 提取人像并优化背景，请稍候...
            </p>
          </div>
        </div>
      )}
    </>
  );
}
