import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

interface VideoListPlayModalProps {
  isOpen: boolean;
  videoPaths: string[];
  onClose: () => void;
  onSelectVideo?: (videoPath: string) => void;
}

export function VideoListPlayModal({ isOpen, videoPaths, onClose, onSelectVideo }: VideoListPlayModalProps) {
  const { t } = useTranslation();
  const [isPlaying, setIsPlaying] = useState(false);
  const [videoUrl, setVideoUrl] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [isMaximized, setIsMaximized] = useState(false);
  const [currentIndex, setCurrentIndex] = useState(0);

  const toggleMaximize = () => {
    setIsMaximized(!isMaximized);
  };

  useEffect(() => {
    if (isOpen && videoPaths.length > 0) {
      setIsMaximized(false); // 打开弹窗时重置最大化状态
      setCurrentIndex(0); // 重置到第一个视频
      loadVideo(0);
    } else if (!isOpen) {
      // 关闭弹窗时清理视频缓存
      setVideoUrl('');
      setError('');
      setIsMaximized(false); // 关闭弹窗时重置最大化状态
      setCurrentIndex(0);
    }
  }, [isOpen, videoPaths]);

  const loadVideo = (index: number) => {
    if (index < 0 || index >= videoPaths.length) {
      setError('视频不存在');
      return;
    }

    setIsLoading(true);
    setError('');
    try {
      const videoPath = videoPaths[index];
      console.log('loadVideo: loading video at index:', index, 'path:', videoPath);
      
      // 对视频路径进行 Base64 编码
      const encodedPath = btoa(unescape(encodeURIComponent(videoPath)));
      // 构建视频流 URL
      const url = `/api/videoPath/${encodedPath}`;
      console.log('loadVideo: requesting video from:', url);
      
      setVideoUrl(url);
      setCurrentIndex(index);
      if (onSelectVideo) {
        onSelectVideo(videoPath);
      }
      console.log('loadVideo: videoUrl set successfully');
    } catch (err) {
      setError('获取视频流失败');
      console.error('Failed to load video:', err);
    } finally {
      setIsLoading(false);
      console.log('loadVideo: completed');
    }
  };

  const handlePrev = () => {
    const newIndex = (currentIndex - 1 + videoPaths.length) % videoPaths.length;
    loadVideo(newIndex);
  };

  const handleNext = () => {
    const newIndex = (currentIndex + 1) % videoPaths.length;
    loadVideo(newIndex);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-80">
      <div className={`relative w-full ${isMaximized ? 'max-w-none max-h-none' : 'max-w-6xl max-h-[90vh]'}`}>
        <div className="absolute top-4 right-4 z-10 flex space-x-15">
          <button
            onClick={toggleMaximize}
            className="w-12 h-12 flex items-center justify-center rounded-full bg-black bg-opacity-60 text-white hover:bg-opacity-80 transition-colors"
            aria-label={isMaximized ? '恢复大小' : '放大'}
          >
            <span className={`text-2xl text-white ${isMaximized ? 'i-mdi-arrow-collapse-all' : 'i-mdi-arrow-expand-all'}`} />
          </button>
          <button
            onClick={onClose}
            className="w-12 h-12 flex items-center justify-center rounded-full bg-black bg-opacity-60 text-white hover:bg-opacity-80 transition-colors"
            aria-label={t('common.close')}
          >
            <span className="i-mdi-close text-2xl text-white" />
          </button>
        </div>

        <div className={`relative ${isMaximized ? 'w-full h-[90vh]' : 'aspect-video'} bg-black rounded-lg overflow-hidden`}>
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
            <>
              <video
                src={videoUrl}
                className="w-full h-full object-contain"
                controls
                autoPlay
                loop
                onPlay={() => setIsPlaying(true)}
                onPause={() => setIsPlaying(false)}
                onEnded={() => setIsPlaying(false)}
              >
                <track kind="captions" srcLang="en" label="English" />
                Your browser does not support the video tag.
              </video>
              
              {/* 视频切换按钮 */}
              <div className="absolute inset-0 flex items-center justify-between p-4 pointer-events-none">
                <button
                  onClick={handlePrev}
                  className="w-16 h-16 flex items-center justify-center rounded-full bg-black bg-opacity-60 text-white hover:bg-opacity-80 transition-colors pointer-events-auto"
                  aria-label="上一个视频"
                >
                  <span className="i-mdi-chevron-left text-3xl text-white" />
                </button>
                <button
                  onClick={handleNext}
                  className="w-16 h-16 flex items-center justify-center rounded-full bg-black bg-opacity-60 text-white hover:bg-opacity-80 transition-colors pointer-events-auto"
                  aria-label="下一个视频"
                >
                  <span className="i-mdi-chevron-right text-3xl text-white" />
                </button>
              </div>
              
              {/* 视频信息 */}
              <div className="absolute bottom-10 left-0 right-0 flex justify-center pointer-events-none">
                <div className="bg-black bg-opacity-60 text-white px-4 py-2 rounded-full">
                  {currentIndex + 1} / {videoPaths.length}
                </div>
              </div>
            </>
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