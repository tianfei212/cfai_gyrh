import React, { useState, useEffect } from 'react';

interface CountdownTimerProps {
  duration?: number; // Duration in seconds, default 120
  size?: number;     // Size in pixels
  strokeWidth?: number;
  onComplete?: () => void;
}

export const CountdownTimer: React.FC<CountdownTimerProps> = ({ 
  duration = 120, 
  size = 192, // default lg:w-48 -> 192px
  strokeWidth = 8,
  onComplete 
}) => {
  const [timeLeft, setTimeLeft] = useState(duration);
  
  // Calculate SVG parameters
  const center = size / 2;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  
  // Calculate progress (goes from 0 to circumference)
  // We want it to start full and empty out, or start empty and fill up?
  // Usually timers "empty" out.
  const progress = ((duration - timeLeft) / duration) * circumference;
  const dashoffset = progress; 

  useEffect(() => {
    if (timeLeft <= 0) {
      onComplete?.();
      return;
    }

    const timer = setInterval(() => {
      setTimeLeft((prev) => Math.max(0, prev - 1));
    }, 1000);

    return () => clearInterval(timer);
  }, [timeLeft, onComplete]);

  // Format time as MM:SS
  const minutes = Math.floor(timeLeft / 60);
  const seconds = timeLeft % 60;
  const timeString = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;

  return (
    <div className="relative flex items-center justify-center" style={{ width: size, height: size }}>
      {/* SVG Circle */}
      <svg className="transform -rotate-90 w-full h-full">
        {/* Background Track */}
        <circle
          className="text-zinc-800"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          fill="transparent"
          r={radius}
          cx={center}
          cy={center}
        />
        {/* Progress Circle */}
        <circle
          className="text-indigo-500 transition-all duration-1000 ease-linear"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          fill="transparent"
          r={radius}
          cx={center}
          cy={center}
          strokeDasharray={circumference}
          strokeDashoffset={dashoffset}
          strokeLinecap="round"
        />
      </svg>
      
      {/* Time Text */}
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-4xl lg:text-5xl font-bold text-white font-mono tracking-wider">
          {timeString}
        </span>
      </div>
    </div>
  );
};
