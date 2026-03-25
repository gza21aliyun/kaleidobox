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
  const [isListExpanded, setIsListExpanded] = useState(true);

  const toggleMaximize = () => {
    setIsMaximized(!isMaximized);
  };

  const cleanupVideoCache = async () => {
    try {
      console.log('cleanupVideoCache: called');
      const response = await fetch(`/api/video/cleanup/`);
      if (response.ok) {
        console.log('cleanupVideoCache: video cache cleaned up successfully');
      } else {
        console.error('cleanupVideoCache: failed to clean up video cache:', response.status);
      }
    } catch (err) {
      console.error('cleanupVideoCache: error cleaning up video cache:', err);
    }
  };

  useEffect(() => {
    if (isOpen && videoPaths.length > 0) {
      setIsMaximized(false); // 打开弹窗时重置最大化状态
      setCurrentIndex(0); // 重置到第一个视频
      loadVideo(0);
    } else if (!isOpen) {
      // 关闭弹窗时清理视频缓存
      cleanupVideoCache();
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

  // 获取视频文件名
  const getVideoFileName = (path: string) => {
    return path.split('\\').pop() || path;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-80">
      <div className={`relative w-full ${isMaximized ? 'max-w-none' : 'max-w-7xl'} h-[90vh] flex`}>
        {/* 视频列表 */}
        <div className="relative h-full">
          {/* 折叠按钮 - 始终可见 */}
          <button
            onClick={() => setIsListExpanded(!isListExpanded)}
            className={`absolute top-4 left-0 z-20 w-8 h-8 flex items-center justify-center rounded-full bg-brand-800 text-white hover:bg-brand-700 transition-colors ${isListExpanded ? 'left-72' : 'left-0'}`}
            aria-label={isListExpanded ? '折叠列表' : '展开列表'}
          >
            <span className={`text-lg ${isListExpanded ? 'i-mdi-chevron-left' : 'i-mdi-chevron-right'}`} />
          </button>
          
          {/* 视频列表内容 */}
          <div className={`${isListExpanded ? 'w-80' : 'w-0'} transition-all duration-300 overflow-hidden h-full`}>
            <div className="h-full flex flex-col bg-brand-900 rounded-l-lg border-r border-brand-700">
              {/* 列表标题 */}
              <div className="flex items-center p-4 border-b border-brand-700">
                <h3 className="text-white font-medium">视频列表</h3>
              </div>
              
              {/* 视频列表 */}
              <div className="flex-1 overflow-y-auto p-2">
                {videoPaths.map((path, index) => (
                  <button
                    key={index}
                    onClick={() => loadVideo(index)}
                    className={`w-full text-left p-3 rounded-lg mb-1 transition-colors ${currentIndex === index
                      ? 'bg-primary-600 text-white'
                      : 'bg-brand-800 text-brand-300 hover:bg-brand-700'
                    }`}
                  >
                    <div className="truncate">{getVideoFileName(path)}</div>
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
        
        {/* 视频播放器 */}
        <div className="flex-1 h-full">
          <div className="absolute top-4 right-4 z-10 flex space-x-4">
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

          <div className="relative h-full bg-black rounded-r-lg overflow-hidden">
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
    </div>
  );
}