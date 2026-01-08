
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
      className={`
        relative group cursor-pointer
        border-2 border-dashed rounded-2xl
        p-8 lg:p-16 2xl:p-24
        min-h-[300px] lg:min-h-[400px] 2xl:min-h-[500px]
        flex flex-col items-center justify-center
        transition-all duration-200 ease-in-out
        bg-zinc-900/50
        ${isDragging ? 'border-indigo-500 bg-indigo-500/10' : 'border-zinc-700 hover:border-indigo-400/50 hover:bg-zinc-800'}
      `}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      onClick={() => inputRef.current?.click()}
    >
      <input
        type="file"
        ref={inputRef}
        className="hidden"
        accept="image/*"
        onChange={onInputChange}
      />
      
      <div className={`
        p-4 lg:p-6 2xl:p-8 rounded-full mb-4 lg:mb-6 2xl:mb-8 transition-colors
        ${isDragging ? 'bg-indigo-500 text-white' : 'bg-zinc-800 text-zinc-400 group-hover:text-indigo-400'}
      `}>
        {isDragging ? <UploadCloud className="w-8 h-8 lg:w-12 lg:h-12 2xl:w-16 2xl:h-16" /> : <ImageIcon className="w-8 h-8 lg:w-12 lg:h-12 2xl:w-16 2xl:h-16" />}
      </div>
      
      <p className="text-lg lg:text-xl 2xl:text-2xl font-medium mb-2 text-center">
        {isDragging ? '松开以上传' : label}
      </p>
      <p className="text-zinc-500 text-sm lg:text-base 2xl:text-lg text-center">
        {subLabel}
      </p>
    </div>
  );
};
