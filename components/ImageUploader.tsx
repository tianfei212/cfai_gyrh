
import React, { useCallback, useState, useRef } from 'react';
import { UploadCloud, Image as ImageIcon } from 'lucide-react';
import { blobToBase64, resizeImage } from '../utils/fileUtils';

interface ImageUploaderProps {
  onImageSelected: (base64: string) => void;
  label?: string;
  subLabel?: string;
}

export const ImageUploader: React.FC<ImageUploaderProps> = ({ 
  onImageSelected,
  label = "点击或拖拽上传图片",
  subLabel = "支持 JPG, PNG, WebP (最大 10MB)"
}) => {
  const [isDragging, setIsDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = async (file: File) => {
    if (!file.type.startsWith('image/')) return;
    
    try {
      const base64 = await blobToBase64(file);
      // Resizing to 1280px ensures the Base64 payload is compact enough for third-party proxies
      const resized = await resizeImage(base64, 1280); 
      onImageSelected(resized);
    } catch (e) {
      console.error("File reading error", e);
    }
  };

  const onDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const onDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const onDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFile(e.dataTransfer.files[0]);
    }
  }, []);

  const onInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      handleFile(e.target.files[0]);
    }
  };

  return (
    <div
      onClick={() => inputRef.current?.click()}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      className={`
        w-full h-full relative group cursor-pointer
        border-2 border-dashed rounded-xl
        flex flex-col items-center justify-center
        transition-all duration-200 ease-in-out
        bg-zinc-900/30 hover:bg-zinc-800/50
        p-2 sm:p-4 lg:p-6
        ${isDragging ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700 hover:border-indigo-400/50'}
      `}
    >
      <input
        type="file"
        ref={inputRef}
        className="hidden"
        accept="image/*"
        onChange={onInputChange}
      />
      
      <div className={`
        p-2 lg:p-3 2xl:p-4 rounded-full mb-2 lg:mb-4 transition-colors shrink-0
        ${isDragging ? 'bg-indigo-500 text-white' : 'bg-zinc-800 text-zinc-400 group-hover:text-indigo-400'}
      `}>
        {isDragging ? <UploadCloud className="w-5 h-5 lg:w-6 lg:h-6 2xl:w-8 2xl:h-8" /> : <ImageIcon className="w-5 h-5 lg:w-6 lg:h-6 2xl:w-8 2xl:h-8" />}
      </div>
      
      <div className="text-center w-full px-4">
        <p className="text-sm lg:text-base 2xl:text-lg font-medium mb-1 text-zinc-200 group-hover:text-white transition-colors truncate">
          {isDragging ? '松开以上传' : label}
        </p>
        <p className="text-zinc-500 text-[10px] lg:text-xs 2xl:text-sm hidden sm:block truncate">
          {subLabel}
        </p>
      </div>
    </div>
  );
};
