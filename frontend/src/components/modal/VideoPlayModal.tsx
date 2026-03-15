import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

interface VideoPlayModalProps {
  isOpen: boolean;
  gameId: string;
  onClose: () => void;
}

export function VideoPlayModal({ isOpen, gameId, onClose }: VideoPlayModalProps) {
  const { t } = useTranslation();
  const [isPlaying, setIsPlaying] = useState(false);
  const [videoUrl, setVideoUrl] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen && gameId) {
      fetchVideoStream();
    } else if (!isOpen && gameId) {
      // 关闭弹窗时清理视频缓存
      cleanupVideoCache();
      setVideoUrl('');
      setError('');
    }
  }, [isOpen, gameId]);

  const cleanupVideoCache = async () => {
    try {
      console.log('cleanupVideoCache: called with gameId:', gameId);
      const response = await fetch(`/api/video/cleanup/${gameId}`);
      if (response.ok) {
        console.log('cleanupVideoCache: video cache cleaned up successfully');
      } else {
        console.error('cleanupVideoCache: failed to clean up video cache:', response.status);
      }
    } catch (err) {
      console.error('cleanupVideoCache: error cleaning up video cache:', err);
    }
  };

  const fetchVideoStream = async () => {
    setIsLoading(true);
    setError('');
    try {
      console.log('fetchVideoStream: called with gameId:', gameId);
      
      // 构建视频流 URL (使用相对路径)
      const videoUrl = `/api/video/${gameId}`;
      console.log('fetchVideoStream: requesting video from:', videoUrl);
      
      setVideoUrl(videoUrl);
      console.log('fetchVideoStream: videoUrl set successfully');
    } catch (err) {
      setError('获取视频流失败');
      console.error('Failed to fetch video stream:', err);
    } finally {
      setIsLoading(false);
      console.log('fetchVideoStream: completed');
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-80">
      <div className="relative w-full max-w-6xl max-h-[90vh]">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 z-10 w-10 h-10 flex items-center justify-center rounded-full bg-black bg-opacity-50 text-white hover:bg-opacity-70 transition-colors"
          aria-label={t('common.close')}
        >
          <div className="i-mdi-close text-xl" />
        </button>

        <div className="relative aspect-video bg-black rounded-lg overflow-hidden">
          {isLoading ? (
            <div className="w-full h-full flex items-center justify-center text-white">
              <div className="i-mdi-loading text-4xl animate-spin" />
              <span className="ml-2">加载视频中...</span>
            </div>
          ) : error ? (
            <div className="w-full h-full flex items-center justify-center text-white">
              <div className="i-mdi-alert-circle text-4xl text-red-500" />
              <span className="ml-2">{error}</span>
            </div>
          ) : videoUrl ? (
            <video
              src={videoUrl}
              className="w-full h-full object-contain"
              controls
              autoPlay
              onPlay={() => setIsPlaying(true)}
              onPause={() => setIsPlaying(false)}
              onEnded={() => setIsPlaying(false)}
            >
              <track kind="captions" srcLang="en" label="English" />
              Your browser does not support the video tag.
            </video>
          ) : (
            <div className="w-full h-full flex items-center justify-center text-white">
              没有可用的视频
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
