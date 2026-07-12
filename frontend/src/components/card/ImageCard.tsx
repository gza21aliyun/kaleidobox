import { models } from "../../../wailsjs/go/models";
import { GetImageBackupByUrl, FetchGetchuImages, ScaleImageWithMagpie } from "../../../wailsjs/go/service/ImageService";
import { useEffect, useState, useRef } from "react";
import { createPortal } from "react-dom";
import { useAppStore } from "../../store";
import { u } from "@unocss/preset-wind3/dist/rules-Dd5IWQsx.mjs";
import toast from "react-hot-toast";




interface ImageBackupProps {
  imageBackup: models.ImageBackup;
  className?: string | undefined;
  alt?: string;
  style?: React.CSSProperties | undefined;
  draggable?: boolean | undefined;
  selectMode?: boolean;
  onSelect?: (selected: models.ImageBackup) => void;
  referrerPolicy?: React.HTMLAttributeReferrerPolicy | undefined;  
    onDragStart?: React.DragEventHandler | undefined;
    onError?: React.ReactEventHandler | undefined;
  isShowTime?: boolean;
  clickNext?: (isNext: boolean, currentIndex: number) => Promise<models.ImageBackup | null>;
  hasNext?: boolean;
  hasPrev?: boolean;
  currentIndex?: number;
  onClose?: () => void;
}

export function ImageBackupCard({
    imageBackup, className, alt, style, draggable, referrerPolicy, onDragStart, onError, selectMode = false, onSelect, isShowTime = false, clickNext, hasNext, hasPrev, currentIndex = 0, onClose,
}: ImageBackupProps) { 
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [scale, setScale] = useState(1);
    const { config } = useAppStore();
    const imgRef = useRef<HTMLImageElement>(null);
    const [cImg, setCImg] = useState<models.ImageBackup>(imageBackup);
    const [isHovering, setIsHovering] = useState(false);

    const switchImage = async (isNext: boolean) => { 
        if (!clickNext) return;
        const ib = await clickNext(isNext, currentIndex);
        if (ib) {
            setCImg(ib);
        }
    };

    const handleFullscreen = async () => {
        if (!imgRef.current) return;
        const rect = imgRef.current.getBoundingClientRect();
        try {
            await ScaleImageWithMagpie(
                Math.round(rect.left),
                Math.round(rect.top),
                Math.round(rect.width),
                Math.round(rect.height),
                window.innerWidth,
                window.innerHeight
            );
        } catch (error) {
            console.error("Failed to scale image with Magpie:", error);
        }
    };

    // 格式化时间显示（通用格式，不需要翻译）
    const formatTime = (time: any) => {
        if (!time) return '';
        try {
            const date = new Date(time);
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            const hour = String(date.getHours()).padStart(2, '0');
            const minute = String(date.getMinutes()).padStart(2, '0');
            const second = String(date.getSeconds()).padStart(2, '0');
            return `${year}-${month}-${day} ${hour}:${minute}`;
        } catch {
            return '';
        }
    };

    const getImageUrl = (ib: models.ImageBackup) => {
        if (ib.local_path !== "") {
            return getLocalPath(ib.local_path);
        }
        return ib.url;
    };

    const handleWheel = (e: React.WheelEvent<HTMLDivElement>) => {
        e.preventDefault();
        const delta = e.deltaY > 0 ? 0.9 : 1.1;
        setScale(prev => Math.max(0.1, Math.min(5, prev * delta)));
    };

    const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'ArrowUp') {
            setScale(prev => Math.min(5, prev * 1.1));
            e.preventDefault();
        } else if (e.key === 'ArrowDown') {
            setScale(prev => Math.max(0.1, prev * 0.9));
            e.preventDefault();
        } else if (e.key === 'Escape') {
            setIsModalOpen(false);
            onClose?.();
            e.preventDefault();
        } else if (e.key === 'ArrowRight') {
            if (hasNext && clickNext) {
                switchImage(true);
                e.stopPropagation();
            }
            e.preventDefault();
        } else if (e.key === 'ArrowLeft') {
            if (hasPrev && clickNext) {
                switchImage(false);
                e.stopPropagation();
            }
            e.preventDefault();
        }
    };

    useEffect(() => {
        if (isModalOpen) {
            document.addEventListener('keydown', handleKeyDown);
            return () => {
                document.removeEventListener('keydown', handleKeyDown);
                setScale(1);
            };
        }
    }, [isModalOpen, hasNext, hasPrev, clickNext]);

    const handleClickImg = () => { 
        if (selectMode && onSelect) {
            onSelect(imageBackup);
        } else {
            setCImg(imageBackup)
            setIsModalOpen(true);
        }
        
    };

    return (
        <>
            <div 
                className={`cursor-pointer hover:opacity-80 transition-all ${selectMode ? 'ring-2 ring-brand-500 ring-offset-2 dark:ring-offset-brand-900 rounded-md' : ''}`} 
                onClick={() => handleClickImg()}
            >
                <div 
                    className={`image-card ${className || ''}`}
                    style={style}
                >
                    { imageBackup.local_path !== "" ? 
                    (<img src={getImageUrl(imageBackup)} 
                    alt={alt || "Image"}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                    draggable={false}
                    onDragStart={onDragStart}
                    referrerPolicy={referrerPolicy}
                    onError={onError}
                     />) : 
                    ( <img src={imageBackup.url} alt={alt || "Image"}
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }}         
                        draggable={false}
                        onDragStart={onDragStart}
                        referrerPolicy={referrerPolicy}   
                        onError={onError}    
                    /> )}
                </div>
                {isShowTime && imageBackup.created_at && (
                    <div className="text-xs text-gray-500 dark:text-gray-400 text-center py-1 truncate">
                        {formatTime(imageBackup.created_at)}
                    </div>
                )}
            </div>

            {isModalOpen && createPortal(
                <div 
                    className="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4 image-modal-overlay"
                    onClick={() => {
                        setIsModalOpen(false);
                        onClose?.();
                    }}
                    onWheel={handleWheel}
                >
                    <div 
                        className="relative transition-all duration-200"
                        style={{ 
                            transform: `scale(${scale})`, 
                            transformOrigin: 'center center',
                            maxWidth: '90vw',
                            maxHeight: '90vh'
                        }}
                        onClick={(e) => e.stopPropagation()}
                        onMouseEnter={() => setIsHovering(true)}
                        onMouseLeave={() => setIsHovering(false)}
                    >
                        <div className="absolute -top-10 right-0 flex gap-2">
                            {config?.magpie_path && (
                                <button
                                    className="text-white hover:text-gray-300 text-2xl font-bold"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        handleFullscreen();
                                    }}
                                    title="全屏显示，注意该功能会修改Magpie裁剪设置"
                                >
                                    ⛶
                                </button>
                            )}
                            <button
                                className="text-white hover:text-gray-300 text-3xl font-bold"
                                onClick={() => {
                                    setIsModalOpen(false);
                                    onClose?.();
                                }}
                            >
                                ×
                            </button>
                        </div>

                        {hasPrev && (
                            <button
                                className={`absolute left-0 top-1/2 -translate-y-1/2 -translate-x-12 transition-opacity duration-200 text-white text-4xl font-bold drop-shadow-lg ${isHovering ? 'opacity-100' : 'opacity-0'}`}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    if (clickNext) {
                                        switchImage(false);
                                    }
                                }}
                            >
                                ◀
                            </button>
                        )}

                        {hasNext && (
                            <button
                                className={`absolute right-0 top-1/2 -translate-y-1/2 translate-x-12 transition-opacity duration-200 text-white text-4xl font-bold drop-shadow-lg ${isHovering ? 'opacity-100' : 'opacity-0'}`}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    if (clickNext) {
                                        switchImage(true);
                                    }
                                }}
                            >
                                ▶
                            </button>
                        )}

                        <img
                            ref={imgRef}
                            src={getImageUrl(cImg)}
                            alt="Image"
                            className="max-w-full max-h-[90vh] object-contain"
                        />
                    </div>
                </div>,
                document.body
            )}
        </>
    );
}

function getLocalPath(localPath: string)  {
        // console.log("local path 01:", localPath)
        if (!localPath) {
            return ""
        }
        const ar = localPath.split("\\")
        if (ar.length < 3) {
            return ""
        }
        const path = `/local/${ar[ar.length - 3]}/${ar[ar.length - 2]}/${ar[ar.length - 1]}`
        return path;
    }


interface ImageCardProps {
  url: string;
  className?: string;
  alt?: string;
  style?: React.CSSProperties;
  draggable?: boolean | undefined;
  referrerPolicy?: React.HTMLAttributeReferrerPolicy | undefined;  
    onDragStart?: React.DragEventHandler | undefined;
    onError?: React.ReactEventHandler | undefined;
  lazyLoad?: boolean;
  selectMode?: boolean;
  onSelect?: (selected: models.ImageBackup) => void;
  tempDownload?: boolean; // 为true时下载到临时文件夹，不保存到数据库
  urls?: string[];
}
export function ImageCard({
    url,
    className,
    alt,
    style,
    draggable,
    referrerPolicy,
    onDragStart,
    onError,
    lazyLoad = true,
    selectMode = false,
    onSelect,
    tempDownload = false,
    urls = [url],
}: ImageCardProps) {
    // const [cUrl, setCUrl] = useState(url);
    const [imageBackup, setImageBackup] = useState<models.ImageBackup | null>(null);
    const [loading, setLoading] = useState(false);
    const [isVisible, setIsVisible] = useState(!lazyLoad); // 非懒加载时默认可见
    const ref = useRef<HTMLDivElement>(null);
    const [mounted, setMounted] = useState(false);
    const [index, setIndex] = useState(urls.indexOf(url));

    useEffect(() => {
        setIndex(urls.indexOf(url));
    }, [url, urls]);

    // 组件挂载完成
    useEffect(() => {
        setMounted(true);
        return () => setMounted(false);
    }, []);

    // 监听元素是否进入可视区域（仅在懒加载时启用）
    useEffect(() => {
        if (!lazyLoad) return;
        
        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting) {
                    setIsVisible(true);
                }
            },
            { 
                threshold: 0.01, // 减小阈值，只要有1%进入视口就触发
                rootMargin: '50px' // 添加预加载区域，提前50px开始加载
            }
        );

        if (ref.current) {
            observer.observe(ref.current);
            
            // 初始状态检查：立即检查元素是否在可视区域内
            const checkVisibility = () => {
                if (!ref.current) return;
                
                const rect = ref.current.getBoundingClientRect();
                const isInView = (
                    rect.top < (window.innerHeight || document.documentElement.clientHeight) &&
                    rect.bottom > 0 &&
                    rect.left < (window.innerWidth || document.documentElement.clientWidth) &&
                    rect.right > 0
                );
                
                if (isInView) {
                    setIsVisible(true);
                }
            };
            
            // 立即检查
            checkVisibility();
            
            // 延迟再次检查，确保DOM已经完全渲染
            const timer = setTimeout(checkVisibility, 100);
            
            return () => {
                clearTimeout(timer);
                if (ref.current) {
                    observer.unobserve(ref.current);
                }
            };
        }

        return () => {
            if (ref.current) {
                observer.unobserve(ref.current);
            }
        };
    }, [lazyLoad]);

    // 加载图片逻辑
    useEffect(() => {
        if (lazyLoad && !mounted) return;
        if ((lazyLoad && !isVisible) || !url || url === "") {
            return;
        }
        setLoading(true);
        
        const fetchImage = async () => {
            try {
                const ib = await FetchImageData(url, tempDownload, index);
                if (ib && mounted) {
                    setImageBackup(ib);
                }
            
                if (mounted) {
                    setLoading(false);
                }
            } catch (error) {
                if (mounted) {
                    setLoading(false);
                }
            }
        };
        
        fetchImage();

        return () => {
            setImageBackup(null);
            setLoading(false);
        };
    }, [url, isVisible, mounted]);

    // var index = urls.indexOf(cUrl);
    const clickN: (isNext: boolean, currentIndex: number) => Promise<models.ImageBackup | null> = async (isNext: boolean, currentIndex: number) => { 
        if(isNext){
            if(currentIndex < urls.length - 1){
                const nUrl = urls[currentIndex + 1];
                setIndex(currentIndex + 1);
                return await FetchImageData(nUrl, tempDownload, currentIndex + 1);
            }
        }else{
            if(currentIndex > 0){
                const nUrl = urls[currentIndex - 1];
                setIndex(currentIndex - 1);
                return await FetchImageData(nUrl, tempDownload, currentIndex - 1);
            }
        }
        return null;
    };

    // 渲染逻辑
    const renderContent = () => {
        if (lazyLoad && !isVisible) {
            // 元素未进入可视区域时，显示占位符
            return (
                <div ref={ref} className="bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
                </div>
            );
        }

        if (loading) {
            // 加载中，显示loading动画
            return (
                <div ref={ref} className="image-card-loading flex items-center justify-center bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
                    <div className="w-8 h-8 border-4 border-gray-200 border-t-blue-500 rounded-full animate-spin"></div>
                </div>
            );
        }

        if (!imageBackup) {
            // 加载失败或无数据，显示占位符
            return (
                <div ref={ref} className="bg-gray-100 rounded-md" style={{ ...style, minHeight: style?.height ? `${parseInt(style.height.toString()) * 3}px` : '120px' }}>
                </div>
            );
        }

        // 加载成功，显示图片
        return (
            <div ref={ref}>
                <ImageBackupCard 
                    imageBackup={imageBackup} 
                    alt={alt}
                    className={className}
                    style={style}
                    draggable={draggable}
                    onDragStart={onDragStart}
                    referrerPolicy={referrerPolicy}
                    onError={onError}
                    selectMode={selectMode}
                    onSelect={onSelect}
                    clickNext={clickN}
                    hasNext={index < urls.length - 1}
                    hasPrev={index > 0}
                    currentIndex={index}
                    onClose={() => setIndex(urls.indexOf(url))}
                    />
            </div>
        );
    };

    return renderContent();
}

export async function FetchImageData(url: string, isTemp: boolean, index?: number): Promise<models.ImageBackup | null> { 
    if (isTemp) {
        // 下载到临时文件夹，不保存到数据库
        const localPaths = await FetchGetchuImages([url]);
        if (localPaths.length > 0) {
            const ar = localPaths[0].split("\\");
            if (ar.length >= 3) {
                const localUrl = `/local/${ar[ar.length - 3]}/${ar[ar.length - 2]}/${ar[ar.length - 1]}`;
                return {
                    url: url,
                    local_path: localPaths[0],
                    subject_id: "",
                    subject_type: 0,
                    image_type: 0,
                    game_id: "",
                    created_at: new Date(),
                    index: index ?? 0
                } as unknown as models.ImageBackup;
            }
        }
    } else {
        const res = await GetImageBackupByUrl(url, true);
        if (res) {
            (res as any).index = index ?? 0;
        }
        return res;
    }
    return null;
}

